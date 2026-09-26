package announcements

import "time"

// AudienceRuleInput is one Audience rule in a request. At least one field must be set.
type AudienceRuleInput struct {
	DepartmentID *string `json:"department_id"`
	BatchYear    *int    `json:"batch_year"`
	Role         *string `json:"role"`
}

// contentInput is the editable part of an Announcement.
type contentInput struct {
	Title     string              `json:"title" binding:"required"`
	Body      string              `json:"body" binding:"required"`
	Category  string              `json:"category" binding:"required"`
	Audience  []AudienceRuleInput `json:"audience"`
	ExpiresAt *time.Time          `json:"expires_at"`
}

// CreateAnnouncementInput posts an Announcement; Draft saves it without
// publishing or submitting it.
type CreateAnnouncementInput struct {
	contentInput
	Draft bool `json:"draft"`
}

func (i CreateAnnouncementInput) content() contentInput { return i.contentInput }

// UpdateAnnouncementInput replaces a draft or rejected Announcement's content.
type UpdateAnnouncementInput struct {
	contentInput
}

// ReviewInput approves or rejects an Announcement waiting for approval.
type ReviewInput struct {
	Decision string `json:"decision" binding:"required"`
	Note     string `json:"note"`
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
	ReviewNote    *string                `json:"review_note,omitempty"`
	// Approver names who approves a pending Announcement.
	Approver *string `json:"approver,omitempty"`
	// Edit is a waiting or rejected edit to a published one (author only).
	Edit *EditResponse `json:"edit,omitempty"`
}

// FeedMeta carries the cursor for the next page; empty when there are no more.
type FeedMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}
