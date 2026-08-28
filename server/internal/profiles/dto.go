package profiles

type UpdateProfileInput struct {
	Headline     *string `json:"headline" binding:"omitempty,max=200"`
	Bio          *string `json:"bio" binding:"omitempty,max=2000"`
	AvatarURL    *string `json:"avatar_url" binding:"omitempty,max=2048"`
	ShowEmail    *bool   `json:"show_email"`
	ShowPhone    *bool   `json:"show_phone"`
	LinkedInURL  *string `json:"linkedin_url" binding:"omitempty,max=2048"`
	GitHubURL    *string `json:"github_url" binding:"omitempty,max=2048"`
	PortfolioURL *string `json:"portfolio_url" binding:"omitempty,max=2048"`
}

type ProfileResponse struct {
	UserID               string  `json:"user_id"`
	Username             string  `json:"username"`
	FullName             string  `json:"full_name"`
	Headline             *string `json:"headline,omitempty"`
	Bio                  *string `json:"bio,omitempty"`
	AvatarURL            *string `json:"avatar_url,omitempty"`
	PublicProfileEnabled bool    `json:"public_profile_enabled"`
	ShowEmail            bool    `json:"show_email"`
	ShowPhone            bool    `json:"show_phone"`
	Email                *string `json:"email,omitempty"`
	Phone                *string `json:"phone,omitempty"`
	LinkedInURL          *string `json:"linkedin_url,omitempty"`
	GitHubURL            *string `json:"github_url,omitempty"`
	PortfolioURL         *string `json:"portfolio_url,omitempty"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
