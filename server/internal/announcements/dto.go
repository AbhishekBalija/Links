package announcements

import "time"

// AudienceRuleInput is one Audience rule in a request. At least one field must be set.
type AudienceRuleInput struct {
	DepartmentID *string `json:"department_id"`
	BatchYear    *int    `json:"batch_year"`
	Role         *string `json:"role"`
}

type CreateAnnouncementInput struct {
	Title     string              `json:"title" binding:"required"`
	Body      string              `json:"body" binding:"required"`
	Category  string              `json:"category" binding:"required"`
	Audience  []AudienceRuleInput `json:"audience"`
	ExpiresAt *time.Time          `json:"expires_at"`
}

type AudienceRuleResponse struct {
	DepartmentID   *string `json:"department_id"`
	DepartmentCode *string `json:"department_code"`
	BatchYear      *int    `json:"batch_year"`
	Role           *string `json:"role"`
}

type AnnouncementResponse struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Body          string                 `json:"body"`
	Category      Category               `json:"category"`
	Status        Status                 `json:"status"`
	PublisherID   string                 `json:"publisher_id"`
	PublisherName string                 `json:"publisher_name"`
	Audience      []AudienceRuleResponse `json:"audience"`
	PublishedAt   *time.Time             `json:"published_at"`
	ExpiresAt     *time.Time             `json:"expires_at"`
	CreatedAt     time.Time              `json:"created_at"`
}

// FeedMeta carries the cursor for the next page; empty when there are no more.
type FeedMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}
