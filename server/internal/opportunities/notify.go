package opportunities

// Telling an applicant their application moved on (#206): shortlisted,
// selected, or not taken further. Moving it back to applied, an undo, sends
// nothing.

import (
	"context"

	"github.com/AbhishekBalija/Links/server/internal/mailer"
)

// Notifier sends the emails; the mailer is one.
type Notifier interface {
	SendApplicationUpdate(to string, letter mailer.ApplicationUpdate) error
}

type noNotifier struct{}

func (noNotifier) SendApplicationUpdate(string, mailer.ApplicationUpdate) error { return nil }

// WithNotifier makes the service email applicants about their applications.
func (s *Service) WithNotifier(notifier Notifier) *Service {
	s.notifier = notifier
	return s
}

// tellApplicant emails the applicant. The change is already saved, so a
// failed email never undoes it.
func (s *Service) tellApplicant(ctx context.Context, row ApplicantRow) {
	if row.Email == nil {
		return
	}
	switch row.Status {
	case ApplicationShortlisted, ApplicationSelected, ApplicationRejected:
	default:
		return
	}
	opportunity, err := s.repository.Find(ctx, row.OpportunityID)
	if err != nil || opportunity == nil {
		return
	}
	_ = s.notifier.SendApplicationUpdate(*row.Email, mailer.ApplicationUpdate{
		FullName: row.FullName,
		Status:   string(row.Status),
		Title:    opportunity.Title,
		Company:  opportunity.Company,
		JobPath:  "/jobs/" + row.OpportunityID,
	})
}
