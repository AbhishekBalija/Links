package events

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

// AudienceRuleInput is one Audience rule in a request. At least one field must be set.
type AudienceRuleInput struct {
	DepartmentID *string `json:"department_id"`
	BatchYear    *int    `json:"batch_year"`
	Role         *string `json:"role"`
}

// CreateEventInput proposes an Event.
type CreateEventInput struct {
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	EventType       string              `json:"event_type"`
	DepartmentID    *string             `json:"department_id"`
	FacultyMentorID *string             `json:"faculty_mentor_id"`
	Location        string              `json:"location"`
	StartsAt        *time.Time          `json:"starts_at"`
	EndsAt          *time.Time          `json:"ends_at"`
	Capacity        *int                `json:"capacity"`
	Audience        []AudienceRuleInput `json:"audience"`
}

// UpdateEventInput changes only the fields sent. department_id,
// faculty_mentor_id and capacity can be cleared with null.
type UpdateEventInput struct {
	Title           *string              `json:"title"`
	Description     *string              `json:"description"`
	EventType       *string              `json:"event_type"`
	DepartmentID    Optional[string]     `json:"department_id"`
	FacultyMentorID Optional[string]     `json:"faculty_mentor_id"`
	Location        *string              `json:"location"`
	StartsAt        *time.Time           `json:"starts_at"`
	EndsAt          *time.Time           `json:"ends_at"`
	Capacity        Optional[int]        `json:"capacity"`
	Audience        *[]AudienceRuleInput `json:"audience"`
}

type DepartmentRef struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type MentorRef struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

type AudienceRuleResponse struct {
	DepartmentID   *string `json:"department_id,omitempty"`
	DepartmentCode *string `json:"department_code,omitempty"`
	BatchYear      *int    `json:"batch_year,omitempty"`
	Role           *string `json:"role,omitempty"`
}

// EventResponse is an Event as the API returns it.
type EventResponse struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	EventType     Type                   `json:"event_type"`
	Status        Status                 `json:"status"`
	ProposerID    string                 `json:"proposer_id"`
	ProposerName  string                 `json:"proposer_name"`
	Department    *DepartmentRef         `json:"department"`
	FacultyMentor *MentorRef             `json:"faculty_mentor"`
	Location      string                 `json:"location"`
	StartsAt      time.Time              `json:"starts_at"`
	EndsAt        time.Time              `json:"ends_at"`
	Capacity      *int                   `json:"capacity"`
	Audience      []AudienceRuleResponse `json:"audience"`
	SubmittedAt   *time.Time             `json:"submitted_at,omitempty"`
	PublishedAt   *time.Time             `json:"published_at,omitempty"`
	CancelledAt   *time.Time             `json:"cancelled_at,omitempty"`
	CancelReason  *string                `json:"cancel_reason,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ListMeta carries the cursor for the next page; empty when there are no more.
type ListMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
}
