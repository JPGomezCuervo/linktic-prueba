package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"linktic/internal/debug"
	"linktic/internal/telemetry"
)

type storer interface {
	createAccount(ctx context.Context, newAccount *account) (*account, error)
	getAccountByEmail(ctx context.Context, email string) (*account, error)
	getAccountByID(ctx context.Context, id string) (*account, error)
	getAccountByIDWithPassword(ctx context.Context, id string) (*account, error)
	updateAccountByID(ctx context.Context, id string, input *updateAccountInput) (*account, error)
}

var (
	ErrInvalidSignupInput = errors.New("invalid signup input")
	ErrInvalidLoginInput  = errors.New("invalid login input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidUpdateInput = errors.New("invalid update input")
)

var emailRegex = regexp.MustCompile(`^[\w._%+\-]+@[\w.\-]+\.[A-Za-z]{2,}$`)

type Service struct {
	log       *slog.Logger
	store     storer
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(store storer, log *slog.Logger, jwtSecret string, tokenTTL time.Duration) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}

	if log == nil {
		return nil, errors.New("logger is required")
	}

	secret := strings.TrimSpace(jwtSecret)
	if secret == "" {
		return nil, errors.New("jwt secret is required")
	}

	if tokenTTL <= 0 {
		return nil, errors.New("token ttl is required")
	}

	return &Service{
		log:       log.With("component", "auth:service"),
		store:     store,
		jwtSecret: []byte(secret),
		tokenTTL:  tokenTTL,
	}, nil
}

func (s *Service) signup(ctx context.Context, input *signupInput) (*account, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidSignupInput)
	}

	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.Name)
	password := strings.TrimSpace(input.Password)

	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("%w: email is not valid", ErrInvalidSignupInput)
	}

	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidSignupInput)
	}

	if len(password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidSignupInput)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service: hashing password: %w", err)
	}

	newAccount := &account{
		ID:           uuid.NewString(),
		Email:        email,
		Name:         name,
		PasswordHash: string(hashedPassword),
	}

	createdAccount, err := s.store.createAccount(ctx, newAccount)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.email") {
			return nil, fmt.Errorf("%w: %s", ErrEmailAlreadyExists, email)
		}

		if 	debug.DebugAuthService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "accounts"))
		}

		return nil, fmt.Errorf("service: creating account: %w", err)
	}

	if 	debug.DebugAuthService {
		s.log.Debug("account_created", slog.String("account_id", createdAccount.ID))
	}

	return createdAccount, nil
}

func (s *Service) login(ctx context.Context, input *loginInput) (*loginResult, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidLoginInput)
	}

	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)

	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("%w: email is not valid", ErrInvalidLoginInput)
	}

	if password == "" {
		return nil, fmt.Errorf("%w: password is required", ErrInvalidLoginInput)
	}

	selectedAccount, err := s.store.getAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		if 	debug.DebugAuthService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "accounts"))
		}

		return nil, fmt.Errorf("service: selecting account by email: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(selectedAccount.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenTTL)

	claims := jwt.MapClaims{
		"sub":   selectedAccount.ID,
		"email": selectedAccount.Email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("service: signing jwt token: %w", err)
	}

	if 	debug.DebugAuthService {
		s.log.Debug("account_authenticated", slog.String("account_id", selectedAccount.ID))
	}

	return &loginResult{
		Token:     signedToken,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) me(ctx context.Context, accountID string) (*account, error) {
	id := strings.TrimSpace(accountID)
	if id == "" {
		return nil, ErrUnauthorized
	}

	selectedAccount, err := s.store.getAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}

		if 	debug.DebugAuthService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "accounts"))
		}

		return nil, fmt.Errorf("service: selecting account by id: %w", err)
	}

	return selectedAccount, nil
}

func (s *Service) updateMe(ctx context.Context, accountID string, input *updateMeInput) (*account, error) {
	id := strings.TrimSpace(accountID)
	if id == "" {
		return nil, ErrUnauthorized
	}

	if input == nil {
		return nil, fmt.Errorf("%w: payload is required", ErrInvalidUpdateInput)
	}

	if input.Email == nil && input.Name == nil && input.Password == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidUpdateInput)
	}

	validated := &updateAccountInput{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidUpdateInput)
		}
		validated.Name = &name
	}

	if input.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*input.Email))
		if !emailRegex.MatchString(email) {
			return nil, fmt.Errorf("%w: email is not valid", ErrInvalidUpdateInput)
		}
		validated.Email = &email
	}

	if input.Password != nil {
		newPassword := strings.TrimSpace(*input.Password)
		if len(newPassword) < 8 {
			return nil, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidUpdateInput)
		}

		if input.CurrentPassword == nil || strings.TrimSpace(*input.CurrentPassword) == "" {
			return nil, fmt.Errorf("%w: currentPassword is required to change password", ErrInvalidUpdateInput)
		}

		selectedAccount, err := s.store.getAccountByIDWithPassword(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUnauthorized
			}

			if 	debug.DebugAuthService {
				s.log.Debug(telemetry.ErrDBQuery,
					slog.Any(telemetry.KeyErr, err),
					slog.String(telemetry.KeyTable, "accounts"))
			}

			return nil, fmt.Errorf("service: selecting account by id with password: %w", err)
		}

		if err := bcrypt.CompareHashAndPassword([]byte(selectedAccount.PasswordHash), []byte(strings.TrimSpace(*input.CurrentPassword))); err != nil {
			return nil, fmt.Errorf("%w: current password is incorrect", ErrInvalidUpdateInput)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("service: hashing password: %w", err)
		}

		passwordHash := string(hashedPassword)
		validated.PasswordHash = &passwordHash
	}

	updatedAccount, err := s.store.updateAccountByID(ctx, id, validated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}

		if strings.Contains(err.Error(), "UNIQUE constraint failed: accounts.email") {
			return nil, fmt.Errorf("%w", ErrEmailAlreadyExists)
		}

		if 	debug.DebugAuthService {
			s.log.Debug(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err),
				slog.String(telemetry.KeyTable, "accounts"))
		}

		return nil, fmt.Errorf("service: updating account by id: %w", err)
	}

	return updatedAccount, nil
}
