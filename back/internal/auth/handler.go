package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"modernc.org/sqlite"

	"linktic/internal/common"
	"linktic/internal/debug"
	"linktic/internal/middleware"
	"linktic/internal/telemetry"
)

type auther interface {
	signup(ctx context.Context, input *signupInput) (*account, error)
	login(ctx context.Context, input *loginInput) (*loginResult, error)
	me(ctx context.Context, accountID string) (*account, error)
	updateMe(ctx context.Context, accountID string, input *updateMeInput) (*account, error)
}

type Handler struct {
	log     *slog.Logger
	service auther
}

func NewHandler(svc auther, log *slog.Logger) (*Handler, error) {
	if svc == nil {
		return nil, errors.New("service is required")
	}

	if log == nil {
		return nil, errors.New("logger is required")
	}

	return &Handler{
		log:     log.With("component", "auth:handler"),
		service: svc,
	}, nil
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugAuthHandler {
		h.log.Debug("handling_signup", slog.String("path", r.URL.Path))
	}

	input, err := common.DecodeJSON[signupInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	createdAccount, err := h.service.signup(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidSignupInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrEmailAlreadyExists) {
			str := fmt.Sprintf("%s: email already in use\n", http.StatusText(http.StatusConflict));
			http.Error(w, str, http.StatusConflict)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusCreated, createdAccount)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugAuthHandler {
		h.log.Debug("handling_login", slog.String("path", r.URL.Path))
	}

	input, err := common.DecodeJSON[loginInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	result, err := h.service.login(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidLoginInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrInvalidCredentials) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		Expires:  result.ExpiresAt,
		MaxAge:   int(time.Until(result.ExpiresAt).Seconds()),
	})

	err = common.EncodeJSON(w, http.StatusOK, &messageResponse{Message: "logged_in"})
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugAuthHandler {
		h.log.Debug("handling_me", slog.String("path", r.URL.Path))
	}

	accountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	selectedAccount, err := h.service.me(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, selectedAccount)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	if 	debug.DebugAuthHandler {
		h.log.Debug("handling_update_me", slog.String("path", r.URL.Path))
	}

	accountID, ok := middleware.AccountIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	input, err := common.DecodeJSON[updateMeInput](r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	updatedAccount, err := h.service.updateMe(r.Context(), accountID, input)
	if err != nil {
		if errors.Is(err, ErrInvalidUpdateInput) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrEmailAlreadyExists) {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			h.log.Error(telemetry.ErrDBQuery,
				slog.Any(telemetry.KeyErr, err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.log.Error(telemetry.ErrInternal,
			slog.Any(telemetry.KeyErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = common.EncodeJSON(w, http.StatusOK, updatedAccount)
	if err != nil {
		h.log.Error(telemetry.ErrJSONFailure,
			slog.Any(telemetry.KeyInternalErr, err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
