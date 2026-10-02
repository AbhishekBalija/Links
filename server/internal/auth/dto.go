package auth

import (
	"time"
)

type RequestCodeInput struct {
	// Checked as an email after trimming, since phones add spaces.
	Email string `json:"email" binding:"required,max=254"`
}

type RequestCodeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Message     string `json:"message"`
}

type VerifyCodeInput struct {
	ChallengeID string `json:"challenge_id" binding:"required,max=64"`
	// Email is the address the code was asked for. It is needed only when
	// the email is on no list, to offer an Access request.
	Email string `json:"email" binding:"max=254"`
	Code  string `json:"code" binding:"required,max=16"`
}

// ProvenAccessRequestInput is an Access request from someone who proved
// their email (a request token from NOT_ON_LIST). The Department comes from
// the USN.
type ProvenAccessRequestInput struct {
	RequestToken string `json:"request_token" binding:"required,max=2048"`
	USN          string `json:"usn" binding:"required,max=20"`
	FullName     string `json:"full_name" binding:"required,max=200"`
}

// InviteStaffInput adds a staff member by email and role; they wait for
// their first sign-in.
type InviteStaffInput struct {
	Email     string `json:"email" binding:"required,max=254"`
	FullName  string `json:"full_name" binding:"required,max=200"`
	Role      string `json:"role" binding:"required"`
	ScopeType string `json:"scope_type" binding:"required"`
	ScopeID   string `json:"scope_id"`
	Note      string `json:"note" binding:"max=500"`
}

type GoogleSignInInput struct {
	Credential string `json:"credential" binding:"required,max=8192"`
}

type GoogleNonceResponse struct {
	Nonce string `json:"nonce"`
}

type RequestAccessResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	// FirstSignIn is set only when this sign-in completed an account
	// waiting for its first sign-in.
	FirstSignIn *FirstSignInResponse `json:"first_sign_in,omitempty"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type MeProfileResponse struct {
	UserID               string  `json:"user_id"`
	FullName             string  `json:"full_name"`
	Username             string  `json:"username"`
	Headline             *string `json:"headline,omitempty"`
	Bio                  *string `json:"bio,omitempty"`
	AvatarURL            *string `json:"avatar_url,omitempty"`
	PublicProfileEnabled bool    `json:"public_profile_enabled"`
	ShowEmail            bool    `json:"show_email"`
	ShowPhone            bool    `json:"show_phone"`
	LinkedInURL          *string `json:"linkedin_url,omitempty"`
	GitHubURL            *string `json:"github_url,omitempty"`
	PortfolioURL         *string `json:"portfolio_url,omitempty"`
}

type MeStudentIdentityResponse struct {
	USN        string  `json:"usn"`
	BatchYear  int     `json:"batch_year"`
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
	ID              string                `json:"id"`
	Email           *string               `json:"email"`
	Profile         *PendingUserProfile   `json:"profile,omitempty"`
	StudentIdentity *PendingUserStudentID `json:"student_identity,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	// ReportedAt is set when the row came from a class list and the person
	// said "Not you?" on their first sign-in, so the details need checking.
	ReportedAt *time.Time `json:"reported_at"`
}

type PendingUserProfile struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

type PendingUserStudentID struct {
	USN            string `json:"usn"`
	DepartmentCode string `json:"department_code,omitempty"`
	DepartmentName string `json:"department_name,omitempty"`
	// DepartmentHasHOD is false when no HOD is in effect, which is why the
	// request comes to the principal and admins.
	DepartmentHasHOD bool `json:"department_has_hod"`
	BatchYear        int  `json:"batch_year"`
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

const (
	ImportCreated = "created"
	ImportFailed  = "failed"
)

// ImportRowResult is one CSV row's outcome. Row is its spreadsheet row number
// (the header is row 1). Error is set when it failed, or when it was created
// but its Activation email couldn't be sent.
type ImportRowResult struct {
	Row    int    `json:"row"`
	Email  string `json:"email"`
	Status string `json:"status"`
	UserID string `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ImportResponse struct {
	Created int               `json:"created"`
	Failed  int               `json:"failed"`
	Rows    []ImportRowResult `json:"rows"`
}

type GrantRoleInput struct {
	Role      string     `json:"role" binding:"required"`
	ScopeType string     `json:"scope_type" binding:"required"`
	ScopeID   string     `json:"scope_id"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	Note      string     `json:"note" binding:"max=500"`
}

type RoleDepartment struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// RoleAssignmentResponse is one Role assignment. State is active (in effect
// now), scheduled (starts later) or ended.
type RoleAssignmentResponse struct {
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	ScopeType  string          `json:"scope_type"`
	ScopeID    *string         `json:"scope_id"`
	Department *RoleDepartment `json:"department"`
	AssignedBy *string         `json:"assigned_by"`
	StartsAt   time.Time       `json:"starts_at"`
	EndsAt     *time.Time      `json:"ends_at"`
	State      string          `json:"state"`
	CreatedAt  time.Time       `json:"created_at"`
}
