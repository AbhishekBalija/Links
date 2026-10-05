package announcements

// Telling an author what their approver decided (#206): the announcement is
// published, or it came back with a note to fix.

import (
	"context"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
)

// Notifier sends the emails; the mailer is one.
type Notifier interface {
	SendReviewOutcome(to string, letter mailer.ReviewOutcome) error
}

type noNotifier struct{}

func (noNotifier) SendReviewOutcome(string, mailer.ReviewOutcome) error { return nil }

// WithNotifier makes the service email authors about their announcements.
func (s *Service) WithNotifier(notifier Notifier) *Service {
	s.notifier = notifier
	return s
}

// tellAuthor emails the author the decision. It is already saved, so a
// failed email never undoes it.
func (s *Service) tellAuthor(ctx context.Context, authorID, reviewerID, announcementID, title string, published bool, note string) {
	author, err := s.repository.Person(ctx, authorID)
	if err != nil || author == nil {
		return
	}
	reviewer, err := s.repository.Person(ctx, reviewerID)
	if err != nil || reviewer == nil {
		return
	}
	outcome := "sent_back"
	if published {
		outcome = "published"
	}
	_ = s.notifier.SendReviewOutcome(author.Email, mailer.ReviewOutcome{
		FullName:     author.FullName,
		Kind:         "announcement",
		Title:        title,
		Outcome:      outcome,
		ReviewerName: reviewer.FullName,
		Note:         note,
		Path:         "/mine/" + announcementID,
	})
}
