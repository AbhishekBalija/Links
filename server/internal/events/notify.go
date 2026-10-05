package events

// Telling people about an Event they answered (#206): everyone going or
// interested is emailed when it is cancelled, or when its date, time or
// place changes. A description or seat change isn't worth an email.

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
)

// Notifier sends the emails; the mailer is one.
type Notifier interface {
	SendEventNotice(to []mailer.Recipient, letter mailer.EventNotice) error
	SendReviewOutcome(to string, letter mailer.ReviewOutcome) error
}

type noNotifier struct{}

func (noNotifier) SendEventNotice([]mailer.Recipient, mailer.EventNotice) error { return nil }

func (noNotifier) SendReviewOutcome(string, mailer.ReviewOutcome) error { return nil }

// WithNotifier makes the service email people about Events they answered.
func (s *Service) WithNotifier(notifier Notifier) *Service {
	s.notifier = notifier
	return s
}

// collegeTime is India's time, the college's, whatever the server's zone.
var collegeTime = time.FixedZone("IST", 5*60*60+30*60)

// whenLine reads "Fri 9 Oct, 11 am" or "Fri 9 Oct, 11:30 am".
func whenLine(at time.Time) string {
	local := at.In(collegeTime)
	clock := local.Format("3:04 pm")
	if local.Minute() == 0 {
		clock = local.Format("3 pm")
	}
	return local.Format("Mon 2 Jan") + ", " + clock
}

// tell emails everyone who answered going or interested. The change it
// reports is already saved, so a failed email never undoes it.
func (s *Service) tell(ctx context.Context, eventID string, notice mailer.EventNotice) {
	recipients, err := s.repository.Answerers(ctx, eventID)
	if err != nil || len(recipients) == 0 {
		return
	}
	notice.EventPath = fmt.Sprintf("/events/%s", eventID)
	_ = s.notifier.SendEventNotice(recipients, notice)
}

// tellCancelled is the cancellation notice, with the organiser's reason.
func (s *Service) tellCancelled(ctx context.Context, event *EventResponse) {
	notice := mailer.EventNotice{Title: event.Title, Cancelled: true, When: whenLine(event.StartsAt), Where: event.Location}
	if event.CancelReason != nil {
		notice.Reason = *event.CancelReason
	}
	notice.OrganiserName = event.ProposerName
	if event.Organiser != nil {
		notice.OrganiserName = event.Organiser.FullName
	}
	s.tell(ctx, event.ID, notice)
}

// tellChanged is the change notice, sent only when the date, time or place
// moved.
func (s *Service) tellChanged(ctx context.Context, before Event, event *EventResponse) {
	notice := mailer.EventNotice{Title: event.Title, When: whenLine(event.StartsAt), Where: event.Location}
	if !before.StartsAt.Equal(event.StartsAt) || !before.EndsAt.Equal(event.EndsAt) {
		notice.WasWhen = whenLine(before.StartsAt)
		if notice.WasWhen == notice.When {
			notice.WasWhen = whenLine(before.StartsAt) + " to " + before.EndsAt.In(collegeTime).Format("3:04 pm")
			notice.When = whenLine(event.StartsAt) + " to " + event.EndsAt.In(collegeTime).Format("3:04 pm")
		}
	}
	if before.Location != event.Location {
		notice.WasWhere = before.Location
	}
	if notice.WasWhen == "" && notice.WasWhere == "" {
		return
	}
	s.tell(ctx, event.ID, notice)
}

// tellProposer emails the proposer a decision that reaches them: published,
// sent back with a note, or rejected. An HOD's approval only moves the Event
// on to the principal, so it sends nothing. The decision is already saved, so
// a failed email never undoes it.
func (s *Service) tellProposer(ctx context.Context, reviewerID string, event *EventResponse, note string) {
	outcome := ""
	switch Status(event.Status) {
	case StatusPublished:
		outcome = "published"
	case StatusHODChangesRequested, StatusFinalChangesRequested:
		outcome = "sent_back"
	case StatusHODRejected, StatusFinalRejected:
		outcome = "rejected"
	default:
		return
	}
	proposer, err := s.repository.Person(ctx, event.ProposerID)
	if err != nil || proposer == nil {
		return
	}
	reviewer, err := s.repository.Person(ctx, reviewerID)
	if err != nil || reviewer == nil {
		return
	}
	_ = s.notifier.SendReviewOutcome(proposer.Email, mailer.ReviewOutcome{
		FullName:     proposer.FullName,
		Kind:         "event",
		Title:        event.Title,
		Outcome:      outcome,
		ReviewerName: reviewer.FullName,
		Note:         note,
		Path:         "/mine/events/" + event.ID,
	})
}
