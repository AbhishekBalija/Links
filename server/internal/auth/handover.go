package auth

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UnfinishedWork is a module's share of ending a role (ADR 0028). Inside the
// role change's transaction it withdraws the work the person can no longer
// author and hands their upcoming Events to a new Organiser. Each module
// decides from Remaining whether the person can still author an item, since
// it owns those rules.
type UnfinishedWork interface {
	HandOver(ctx context.Context, handover Handover) (HandoverSummary, error)
}

// UnfinishedWorkOn builds a module's UnfinishedWork on a transaction, so its
// changes commit or roll back with the role change.
type UnfinishedWorkOn func(tx *gorm.DB) UnfinishedWork

// Handover describes one role ending.
type Handover struct {
	PersonID string
	ActorID  string
	// Remaining are the person's roles still in effect once this one ends.
	Remaining []RoleAssignment
	// OrganiserID takes over the person's upcoming Events. Empty means each
	// Event's Department HOD, when there is one other than the person.
	OrganiserID string
	At          time.Time
}

// HandoverSummary says what ending the role did, or would do in a preview.
type HandoverSummary struct {
	WithdrawnAnnouncements int `json:"withdrawn_announcements"`
	// ClosedEdits are edits to published Announcements that were waiting for
	// approval or sent back; the published version stays.
	ClosedEdits int `json:"closed_edits"`
	// ReturnedEvents are Event proposals sent back to private drafts.
	ReturnedEvents int          `json:"returned_events"`
	MovedEvents    []MovedEvent `json:"moved_events"`
	// OrganiserNeeded is true when some upcoming Event has no one to take it
	// over by default, so whoever ends the role must pick an Organiser.
	OrganiserNeeded bool `json:"organiser_needed"`
}

// MovedEvent is an upcoming Event that changes Organiser. Organiser is nil
// when nobody takes it over by default.
type MovedEvent struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	StartsAt  time.Time  `json:"starts_at"`
	Organiser *PersonRef `json:"organiser"`
}

// PersonRef names a person.
type PersonRef struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
}

func (s *HandoverSummary) add(other HandoverSummary) {
	s.WithdrawnAnnouncements += other.WithdrawnAnnouncements
	s.ClosedEdits += other.ClosedEdits
	s.ReturnedEvents += other.ReturnedEvents
	s.MovedEvents = append(s.MovedEvents, other.MovedEvents...)
	s.OrganiserNeeded = s.OrganiserNeeded || other.OrganiserNeeded
}

// handOver runs every module's share and adds up what they did.
func handOver(ctx context.Context, work []UnfinishedWork, handover Handover) (HandoverSummary, error) {
	summary := HandoverSummary{MovedEvents: []MovedEvent{}}
	for _, module := range work {
		done, err := module.HandOver(ctx, handover)
		if err != nil {
			return HandoverSummary{}, err
		}
		summary.add(done)
	}
	return summary, nil
}
