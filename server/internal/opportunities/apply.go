package opportunities

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Apply records the Student's Application: made in LINKS for an internal
// Opportunity, or their note that they applied on the company's site for an
// external one. Only an eligible Student applies, once, while it is open.
func (s *Service) Apply(ctx context.Context, actorID, id string) (*ApplicationResponse, error) {
	notFound := apperrors.NewNotFound("opportunity not found")
	if _, err := uuid.Parse(id); err != nil {
		return nil, notFound
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, err
	}
	eligible, err := s.repository.Eligible(ctx, reader, id)
	if err != nil {
		return nil, fmt.Errorf("check eligibility: %w", err)
	}
	if !eligible {
		return nil, notFound
	}
	if !reader.isStudent() {
		return nil, apperrors.NewForbidden("only students can apply")
	}

	now := s.now()
	var application Application
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		opportunity, err := repositories.Opportunities.LockForShare(ctx, id)
		if err != nil {
			return fmt.Errorf("lock opportunity: %w", err)
		}
		if opportunity == nil {
			return notFound
		}
		if opportunity.Status != StatusPublished || !opportunity.ApplyBy.After(now) {
			return apperrors.NewConflict("applications for this opportunity are closed")
		}
		existing, err := repositories.Opportunities.FindApplicationForUpdate(ctx, id, actorID)
		if err != nil {
			return fmt.Errorf("find application: %w", err)
		}
		if existing != nil {
			return alreadyApplied()
		}
		application = Application{
			OpportunityID: id,
			StudentID:     actorID,
			Mode:          opportunity.ApplicationMode,
			Status:        ApplicationApplied,
			AppliedAt:     now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := repositories.Opportunities.CreateApplication(ctx, &application); err != nil {
			// A second request from the same Student can pass the check at the
			// same moment; the unique index decides.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return alreadyApplied()
			}
			return fmt.Errorf("create application: %w", err)
		}
		return auditApplication(ctx, repositories, actorID, "application_submitted", application, map[string]string{
			"opportunity_id": id, "mode": string(application.Mode),
		})
	})
	if err != nil {
		return nil, err
	}
	return toApplicationResponse(application), nil
}

// Withdraw takes back the Student's Application while it is still only
// applied. Withdrawing is final.
func (s *Service) Withdraw(ctx context.Context, actorID, id string) (*ApplicationResponse, error) {
	now := s.now()
	var application *Application
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		var err error
		application, err = repositories.Opportunities.FindApplicationForUpdate(ctx, id, actorID)
		if err != nil {
			return fmt.Errorf("find application: %w", err)
		}
		if application == nil {
			return apperrors.NewNotFound("you haven't applied to this opportunity")
		}
		if application.Status != ApplicationApplied {
			return apperrors.NewConflict("only an application that is still applied can be withdrawn")
		}
		application.Status = ApplicationWithdrawn
		application.WithdrawnAt = &now
		application.UpdatedAt = now
		if err := repositories.Opportunities.UpdateApplication(ctx, application); err != nil {
			return fmt.Errorf("withdraw application: %w", err)
		}
		return auditApplication(ctx, repositories, actorID, "application_withdrawn", *application, map[string]string{"opportunity_id": id})
	})
	if err != nil {
		return nil, err
	}
	return toApplicationResponse(*application), nil
}

func alreadyApplied() error {
	return apperrors.NewConflict("you have already applied to this opportunity")
}

// isStudent is true for a member with the student role in effect and a
// Student identity.
func (reader Reader) isStudent() bool {
	if reader.BatchYear == nil {
		return false
	}
	for _, membership := range reader.Memberships {
		if membership.Role == string(auth.RoleStudent) {
			return true
		}
	}
	return false
}

func auditApplication(ctx context.Context, repositories Repositories, actorID, action string, application Application, metadata map[string]string) error {
	id := application.ID
	if err := repositories.AuditLogs.Create(ctx, &auth.AuditLog{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "opportunity_application",
		ResourceID:   &id,
		Metadata:     metadata,
		CreatedAt:    application.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("audit %s: %w", action, err)
	}
	return nil
}
