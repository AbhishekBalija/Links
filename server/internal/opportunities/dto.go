package opportunities

import (
	"encoding/json"
	"time"
)

// Optional is a PATCH field that can be left out (Set is false), cleared
// with null (Value is nil) or set.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

// EligibilityRuleInput is one Eligibility rule in a request. At least one
// field must be set.
type EligibilityRuleInput struct {
	DepartmentID *string `json:"department_id"`
	BatchYear    *int    `json:"batch_year"`
	Role         *string `json:"role"`
}

// CreateOpportunityInput saves a new Opportunity as a draft.
type CreateOpportunityInput struct {
	OpportunityType string                 `json:"opportunity_type"`
	Title           string                 `json:"title"`
	Company         string                 `json:"company"`
	Description     string                 `json:"description"`
	Location        *string                `json:"location"`
	Compensation    *string                `json:"compensation"`
	ApplyBy         *time.Time             `json:"apply_by"`
	ApplicationMode string                 `json:"application_mode"`
	ExternalURL     *string                `json:"external_url"`
	Eligibility     []EligibilityRuleInput `json:"eligibility"`
}

// UpdateOpportunityInput changes only the fields sent. location,
// compensation and external_url can be cleared with null.
type UpdateOpportunityInput struct {
	OpportunityType *string                 `json:"opportunity_type"`
	Title           *string                 `json:"title"`
	Company         *string                 `json:"company"`
	Description     *string                 `json:"description"`
	Location        Optional[string]        `json:"location"`
	Compensation    Optional[string]        `json:"compensation"`
	ApplyBy         *time.Time              `json:"apply_by"`
	ApplicationMode *string                 `json:"application_mode"`
	ExternalURL     Optional[string]        `json:"external_url"`
	Eligibility     *[]EligibilityRuleInput `json:"eligibility"`
}

type EligibilityRuleResponse struct {
	DepartmentID   *string `json:"department_id,omitempty"`
	DepartmentCode *string `json:"department_code,omitempty"`
	BatchYear      *int    `json:"batch_year,omitempty"`
	Role           *string `json:"role,omitempty"`
}

type PosterRef struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

type OpportunityResponse struct {
	ID              string                    `json:"id"`
	OpportunityType Type                      `json:"opportunity_type"`
	Title           string                    `json:"title"`
	Company         string                    `json:"company"`
	Description     string                    `json:"description"`
	Location        *string                   `json:"location"`
	Compensation    *string                   `json:"compensation"`
	ApplyBy         time.Time                 `json:"apply_by"`
	ApplicationMode Mode                      `json:"application_mode"`
	ExternalURL     *string                   `json:"external_url"`
	Eligibility     []EligibilityRuleResponse `json:"eligibility"`
	Status          Status                    `json:"status"`
	// Open is true while it is published and apply_by is ahead.
	Open bool `json:"open"`
	// MyApplication is the caller's own Application, if they made one.
	MyApplication *ApplicationResponse `json:"my_application"`
	PostedBy      PosterRef            `json:"posted_by"`
	PublishedAt   *time.Time           `json:"published_at"`
	ClosedAt      *time.Time           `json:"closed_at"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type ListMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}

// FeedQuery is the feed's query string, unchecked.
type FeedQuery struct {
	State      string
	Type       string
	Department string
	Cursor     string
	Limit      int
}

// ApplicationResponse is a Student's own view of their Application.
type ApplicationResponse struct {
	ID            string            `json:"id"`
	OpportunityID string            `json:"opportunity_id"`
	Mode          Mode              `json:"mode"`
	Status        ApplicationStatus `json:"status"`
	AppliedAt     time.Time         `json:"applied_at"`
	WithdrawnAt   *time.Time        `json:"withdrawn_at"`
}

func toApplicationResponse(application Application) *ApplicationResponse {
	return &ApplicationResponse{
		ID:            application.ID,
		OpportunityID: application.OpportunityID,
		Mode:          application.Mode,
		Status:        application.Status,
		AppliedAt:     application.AppliedAt,
		WithdrawnAt:   application.WithdrawnAt,
	}
}

// ApplicantQuery is the applicant list's query string, unchecked.
type ApplicantQuery struct {
	Status     string
	Department string
	Batch      string
	Cursor     string
	Limit      int
}

// StatusInput moves an Application. From is the status the caller saw, so a
// change that raced theirs is refused instead of overwritten.
type StatusInput struct {
	From   string `json:"from" binding:"required"`
	Status string `json:"status" binding:"required"`
}

type ApplicantStudent struct {
	UserID         string  `json:"user_id"`
	FullName       string  `json:"full_name"`
	Username       string  `json:"username"`
	Email          *string `json:"email"`
	USN            *string `json:"usn"`
	DepartmentCode *string `json:"department_code"`
	BatchYear      *int    `json:"batch_year"`
}

// ApplicantResponse is one Application in the applicant list.
type ApplicantResponse struct {
	ID              string            `json:"id"`
	OpportunityID   string            `json:"opportunity_id"`
	Student         ApplicantStudent  `json:"student"`
	Mode            Mode              `json:"mode"`
	Status          ApplicationStatus `json:"status"`
	AppliedAt       time.Time         `json:"applied_at"`
	WithdrawnAt     *time.Time        `json:"withdrawn_at"`
	StatusChangedAt *time.Time        `json:"status_changed_at"`
}

func toApplicantResponse(row ApplicantRow) ApplicantResponse {
	return ApplicantResponse{
		ID:            row.ID,
		OpportunityID: row.OpportunityID,
		Student: ApplicantStudent{
			UserID:         row.StudentID,
			FullName:       row.FullName,
			Username:       row.Username,
			Email:          row.Email,
			USN:            row.USN,
			DepartmentCode: row.DepartmentCode,
			BatchYear:      row.BatchYear,
		},
		Mode:            row.Mode,
		Status:          row.Status,
		AppliedAt:       row.AppliedAt,
		WithdrawnAt:     row.WithdrawnAt,
		StatusChangedAt: row.StatusChangedAt,
	}
}
