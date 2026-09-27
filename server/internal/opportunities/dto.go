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
	Open        bool       `json:"open"`
	PostedBy    PosterRef  `json:"posted_by"`
	PublishedAt *time.Time `json:"published_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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
