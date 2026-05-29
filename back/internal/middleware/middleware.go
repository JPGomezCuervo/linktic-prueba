package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	accountIDKey contextKey = "accountID"
	// AuthCookieName is the name of the JWT authentication cookie.
	AuthCookieName = "auth_token"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		slog.Debug("http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.String("duration", time.Since(start).String()))
	})
}

func AuthMiddleware(next http.Handler, jwtSecret string) http.Handler {
	secret := strings.TrimSpace(jwtSecret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		authCookie, err := r.Cookie(AuthCookieName)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(authCookie.Value, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		accountIDValue, ok := claims["sub"]
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		accountID, ok := accountIDValue.(string)
		if !ok || strings.TrimSpace(accountID) == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), accountIDKey, accountID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AccountIDFromContext(ctx context.Context) (string, bool) {
	accountID, ok := ctx.Value(accountIDKey).(string)
	if !ok || strings.TrimSpace(accountID) == "" {
		return "", false
	}

	return accountID, true
}

func WithAccountID(ctx context.Context, accountID string) context.Context {
	trimmed := strings.TrimSpace(accountID)
	if trimmed == "" {
		return ctx
	}

	return context.WithValue(ctx, accountIDKey, trimmed)
}
