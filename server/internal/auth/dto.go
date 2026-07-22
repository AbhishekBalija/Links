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

type ActivateInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ResendActivationInput struct {
	Email string `json:"email" binding:"required,email"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type MeProfileResponse struct {
	UserID    string               `json:"user_id"`
	FullName  string               `json:"full_name"`
	Username  string               `json:"username"`
	Headline  *string              `json:"headline,omitempty"`
	AvatarURL *string              `json:"avatar_url,omitempty"`
}

type MeStudentIdentityResponse struct {
	USN       string  `json:"usn"`
	BatchYear int     `json:"batch_year"`
	RollNumber *string `json:"roll_number,omitempty"`
}

type MeResponse struct {
	UserID          string                     `json:"user_id"`
	Email           *string                    `json:"email"`
	Phone           *string                    `json:"phone,omitempty"`
	Roles           []string                   `json:"roles"`
	Profile         *MeProfileResponse         `json:"profile,omitempty"`
	StudentIdentity *MeStudentIdentityResponse `json:"student_identity,omitempty"`
}

type TokenClaims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles,omitempty"`
	Issuer string   `json:"iss"`
	Aud    string   `json:"aud"`
	IAT    int64    `json:"iat"`
	Exp    int64    `json:"exp"`
	JTI    string   `json:"jti"`
}

type RefreshTokenRaw struct {
	Token     string
	TokenHash string
	ExpiresAt time.Time
}

type VerifyUserInput struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
	Note      string `json:"note"`
}

type UpdateUserStatusInput struct {
	Status string `json:"status" binding:"required,oneof=active suspended rejected"`
	Note   string `json:"note"`
}

type PendingUserResponse struct {
	ID              string                    `json:"id"`
	Email           *string                   `json:"email"`
	Profile         *PendingUserProfile       `json:"profile,omitempty"`
	StudentIdentity *PendingUserStudentID     `json:"student_identity,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
}

type PendingUserProfile struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

type PendingUserStudentID struct {
	USN            string `json:"usn"`
	DepartmentCode string `json:"department_code,omitempty"`
	BatchYear      int    `json:"batch_year"`
}

type ReviewQueueResponse struct {
	Users []PendingUserResponse `json:"users"`
	Total int                   `json:"total"`
}

type VerifyUserResponse struct {
	Message string `json:"message"`
}

type UpdateUserStatusResponse struct {
	Message string `json:"message"`
}