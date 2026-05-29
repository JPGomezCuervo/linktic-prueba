package auth

import (
	"time"

	"linktic/internal/middleware"
)

const AuthCookieName = middleware.AuthCookieName

type account struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	Deleted      bool   `json:"deleted"`
	CreatedAt    int    `json:"createdAt"`
	UpdatedAt    int    `json:"updatedAt"`
}

type signupInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateMeInput struct {
	Email           *string `json:"email"`
	Name            *string `json:"name"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"currentPassword"`
}

type updateAccountInput struct {
	Email        *string
	Name         *string
	PasswordHash *string
}

type loginResult struct {
	Token     string
	ExpiresAt time.Time
}

type messageResponse struct {
	Message string `json:"message"`
}
