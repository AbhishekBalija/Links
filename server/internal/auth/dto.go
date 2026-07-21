package auth

import (
	"time"
)

type RequestAccessInput struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	FullName       string `json:"full_name" binding:"required"`
	USN            string `json:"usn,omitempty"`
	DepartmentCode string `json:"department_code,omitempty"`
	BatchYear      *int   `json:"batch_year,omitempty"`
	Phone          string `json:"phone,omitempty"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RequestAccessResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type TokenClaims struct {
	UserID string `json:"sub"`
	Roles  []string  `json:"roles,omitempty"`
	Issuer string    `json:"iss"`
	Aud    string    `json:"aud"`
	IAT    int64     `json:"iat"`
	Exp    int64     `json:"exp"`
	JTI    string    `json:"jti"`
}

type RefreshTokenRaw struct {
	Token     string
	TokenHash string
	ExpiresAt time.Time
}