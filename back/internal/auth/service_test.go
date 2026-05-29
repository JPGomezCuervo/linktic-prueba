package auth

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authStoreStub struct {
	createAccountFn func(ctx context.Context, newAccount *account) (*account, error)
	getByEmailFn    func(ctx context.Context, email string) (*account, error)
	getByIDFn       func(ctx context.Context, id string) (*account, error)
	getByIDWithPasswordFn func(ctx context.Context, id string) (*account, error)
	updateByIDFn    func(ctx context.Context, id string, input *updateAccountInput) (*account, error)
	receivedAccount *account
}

func (s *authStoreStub) createAccount(ctx context.Context, newAccount *account) (*account, error) {
	s.receivedAccount = newAccount
	return s.createAccountFn(ctx, newAccount)
}

func (s *authStoreStub) getAccountByEmail(ctx context.Context, email string) (*account, error) {
	return s.getByEmailFn(ctx, email)
}

func (s *authStoreStub) getAccountByID(ctx context.Context, id string) (*account, error) {
	if s.getByIDFn == nil {
		return nil, sql.ErrNoRows
	}

	return s.getByIDFn(ctx, id)
}

func (s *authStoreStub) getAccountByIDWithPassword(ctx context.Context, id string) (*account, error) {
	if s.getByIDWithPasswordFn == nil {
		return nil, sql.ErrNoRows
	}

	return s.getByIDWithPasswordFn(ctx, id)
}

func (s *authStoreStub) updateAccountByID(ctx context.Context, id string, input *updateAccountInput) (*account, error) {
	if s.updateByIDFn == nil {
		return nil, sql.ErrNoRows
	}

	return s.updateByIDFn(ctx, id, input)
}

func testAuthServiceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestAuthService(t *testing.T, store storer) *Service {
	t.Helper()

	svc, err := NewService(store, testAuthServiceLogger(), "secret-key", time.Hour)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	return svc
}

func TestServiceSignup_InvalidEmail(t *testing.T) {
	store := &authStoreStub{}
	svc := newTestAuthService(t, store)

	_, err := svc.signup(context.Background(), &signupInput{Email: "invalid", Name: "Name", Password: "password123"})
	if !errors.Is(err, ErrInvalidSignupInput) {
		t.Fatalf("expected ErrInvalidSignupInput, got %v", err)
	}
}

func TestServiceSignup_HashesPasswordAndNormalizesEmail(t *testing.T) {
	store := &authStoreStub{
		createAccountFn: func(_ context.Context, newAccount *account) (*account, error) {
			return &account{ID: newAccount.ID, Email: newAccount.Email, Name: newAccount.Name}, nil
		},
	}
	svc := newTestAuthService(t, store)

	_, err := svc.signup(context.Background(), &signupInput{
		Email:    "  USER@Example.com  ",
		Name:     "John",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("signup error: %v", err)
	}

	if store.receivedAccount == nil {
		t.Fatal("expected account to be passed to store")
	}

	if store.receivedAccount.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", store.receivedAccount.Email)
	}

	if store.receivedAccount.PasswordHash == "password123" {
		t.Fatal("password should be hashed")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(store.receivedAccount.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("password hash does not match original password: %v", err)
	}
}

func TestServiceSignup_DuplicateEmail(t *testing.T) {
	store := &authStoreStub{
		createAccountFn: func(context.Context, *account) (*account, error) {
			return nil, errors.New("UNIQUE constraint failed: accounts.email")
		},
	}
	svc := newTestAuthService(t, store)

	_, err := svc.signup(context.Background(), &signupInput{Email: "user@example.com", Name: "John", Password: "password123"})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestServiceLogin_InvalidCredentials(t *testing.T) {
	store := &authStoreStub{
		getByEmailFn: func(context.Context, string) (*account, error) {
			return nil, sql.ErrNoRows
		},
	}
	svc := newTestAuthService(t, store)

	_, err := svc.login(context.Background(), &loginInput{Email: "user@example.com", Password: "password123"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceLogin_ReturnsSignedToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	store := &authStoreStub{
		getByEmailFn: func(context.Context, string) (*account, error) {
			return &account{ID: "acc-1", Email: "user@example.com", PasswordHash: string(hash)}, nil
		},
	}
	svc := newTestAuthService(t, store)

	result, err := svc.login(context.Background(), &loginInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}

	if strings.TrimSpace(result.Token) == "" {
		t.Fatal("expected token to be set")
	}

	token, err := jwt.Parse(result.Token, func(token *jwt.Token) (any, error) {
		return []byte("secret-key"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse jwt: %v", err)
	}

	if !token.Valid {
		t.Fatal("expected token to be valid")
	}
}

func TestServiceUpdateMe_RequiresCurrentPasswordForPasswordChange(t *testing.T) {
	store := &authStoreStub{}
	svc := newTestAuthService(t, store)

	newPassword := "newpassword123"
	_, err := svc.updateMe(context.Background(), "acc-1", &updateMeInput{Password: &newPassword})
	if !errors.Is(err, ErrInvalidUpdateInput) {
		t.Fatalf("expected ErrInvalidUpdateInput, got %v", err)
	}
}

func TestServiceUpdateMe_InvalidCurrentPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	store := &authStoreStub{
		getByIDWithPasswordFn: func(context.Context, string) (*account, error) {
			return &account{ID: "acc-1", PasswordHash: string(hash)}, nil
		},
	}
	svc := newTestAuthService(t, store)

	currentPassword := "wrongpassword"
	newPassword := "newpassword123"
	_, err = svc.updateMe(context.Background(), "acc-1", &updateMeInput{
		Password:        &newPassword,
		CurrentPassword: &currentPassword,
	})
	if !errors.Is(err, ErrInvalidUpdateInput) {
		t.Fatalf("expected ErrInvalidUpdateInput, got %v", err)
	}
}

func TestServiceUpdateMe_UpdatesPasswordHash(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	store := &authStoreStub{
		getByIDWithPasswordFn: func(context.Context, string) (*account, error) {
			return &account{ID: "acc-1", PasswordHash: string(hash)}, nil
		},
		updateByIDFn: func(_ context.Context, id string, input *updateAccountInput) (*account, error) {
			if id != "acc-1" {
				t.Fatalf("expected id acc-1, got %s", id)
			}
			if input.PasswordHash == nil || *input.PasswordHash == "" {
				t.Fatal("expected password hash to be set")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(*input.PasswordHash), []byte("newpassword123")); err != nil {
				t.Fatalf("expected password hash for new password, got error: %v", err)
			}
			return &account{ID: "acc-1", Name: "User", Email: "user@example.com"}, nil
		},
	}
	svc := newTestAuthService(t, store)

	currentPassword := "oldpassword123"
	newPassword := "newpassword123"
	updated, err := svc.updateMe(context.Background(), "acc-1", &updateMeInput{
		Password:        &newPassword,
		CurrentPassword: &currentPassword,
	})
	if err != nil {
		t.Fatalf("updateMe error: %v", err)
	}

	if updated.ID != "acc-1" {
		t.Fatalf("expected updated account id acc-1, got %s", updated.ID)
	}
}
