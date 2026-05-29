package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"linktic/internal/middleware"
)

type authHandlerServiceStub struct {
	signupFn func(ctx context.Context, input *signupInput) (*account, error)
	loginFn  func(ctx context.Context, input *loginInput) (*loginResult, error)
	meFn     func(ctx context.Context, accountID string) (*account, error)
	updateMeFn func(ctx context.Context, accountID string, input *updateMeInput) (*account, error)
}

func (s *authHandlerServiceStub) signup(ctx context.Context, input *signupInput) (*account, error) {
	return s.signupFn(ctx, input)
}

func (s *authHandlerServiceStub) login(ctx context.Context, input *loginInput) (*loginResult, error) {
	return s.loginFn(ctx, input)
}

func (s *authHandlerServiceStub) me(ctx context.Context, accountID string) (*account, error) {
	if s.meFn == nil {
		return nil, nil
	}

	return s.meFn(ctx, accountID)
}

func (s *authHandlerServiceStub) updateMe(ctx context.Context, accountID string, input *updateMeInput) (*account, error) {
	if s.updateMeFn == nil {
		return nil, nil
	}

	return s.updateMeFn(ctx, accountID, input)
}

func testAuthHandlerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSignup_ReturnsConflictOnDuplicateEmail(t *testing.T) {
	h, err := NewHandler(&authHandlerServiceStub{
		signupFn: func(context.Context, *signupInput) (*account, error) {
			return nil, ErrEmailAlreadyExists
		},
	}, testAuthHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", strings.NewReader(`{"email":"u@e.com","name":"n","password":"password123"}`))
	rr := httptest.NewRecorder()

	h.Signup(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestLogin_SetsAuthCookie(t *testing.T) {
	h, err := NewHandler(&authHandlerServiceStub{
		loginFn: func(context.Context, *loginInput) (*loginResult, error) {
			return &loginResult{Token: "signed-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}, testAuthHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	res := rr.Result()
	defer res.Body.Close()

	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == AuthCookieName {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected auth cookie to be set")
	}

	if !cookie.HttpOnly {
		t.Fatal("expected auth cookie to be HttpOnly")
	}
}

func TestLogout_ClearsAuthCookie(t *testing.T) {
	h, err := NewHandler(&authHandlerServiceStub{}, testAuthHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}

	res := rr.Result()
	defer res.Body.Close()

	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == AuthCookieName {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected auth cookie to be cleared")
	}

	if cookie.MaxAge != -1 {
		t.Fatalf("expected MaxAge -1, got %d", cookie.MaxAge)
	}
}

func TestUpdateMe_ReturnsOK(t *testing.T) {
	h, err := NewHandler(&authHandlerServiceStub{
		updateMeFn: func(_ context.Context, accountID string, input *updateMeInput) (*account, error) {
			if accountID != "acc-1" {
				t.Fatalf("expected account id acc-1, got %s", accountID)
			}

			if input.Name == nil || *input.Name != "New Name" {
				t.Fatalf("unexpected input name")
			}

			return &account{ID: "acc-1", Name: "New Name", Email: "user@example.com"}, nil
		},
	}, testAuthHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/auth/me", strings.NewReader(`{"name":"New Name"}`))
	req = req.WithContext(middleware.WithAccountID(req.Context(), "acc-1"))
	rr := httptest.NewRecorder()

	h.UpdateMe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestUpdateMe_ReturnsUnauthorizedWithoutContextAccount(t *testing.T) {
	h, err := NewHandler(&authHandlerServiceStub{}, testAuthHandlerLogger())
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/auth/me", strings.NewReader(`{"name":"New Name"}`))
	rr := httptest.NewRecorder()

	h.UpdateMe(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}
