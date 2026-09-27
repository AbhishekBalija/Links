package opportunities

import (
	"context"
	"fmt"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Publish opens a draft to its Eligibility. Its apply_by must still be ahead.
func (s *Service) Publish(ctx context.Context, actorID, id string) (*OpportunityResponse, error) {
	return s.transition(ctx, actorID, id, func(opportunity *Opportunity, now time.Time) (string, error) {
		if opportunity.Status != StatusDraft {
			return "", apperrors.NewConflict("only a draft can be published")
		}
		if !opportunity.ApplyBy.After(now) {
			return "", apperrors.NewValidation("invalid opportunity", map[string]string{"apply_by": "must be in the future to publish"})
		}
		opportunity.Status = StatusPublished
		opportunity.PublishedAt = &now
		return "opportunity_published", nil
	})
}

// Close stops a published Opportunity from taking Applications before its
// apply_by. Closing is final.
func (s *Service) Close(ctx context.Context, actorID, id string) (*OpportunityResponse, error) {
	return s.transition(ctx, actorID, id, func(opportunity *Opportunity, now time.Time) (string, error) {
		if opportunity.Status != StatusPublished {
			return "", apperrors.NewConflict("only a published opportunity can be closed")
		}
		opportunity.Status = StatusClosed
		opportunity.ClosedAt = &now
		opportunity.ClosedBy = &actorID
		return "opportunity_closed", nil
	})
}

// transition applies one status change to a locked Opportunity and audits
// it under the action the change returns.
func (s *Service) transition(ctx context.Context, actorID, id string, change func(*Opportunity, time.Time) (string, error)) (*OpportunityResponse, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		opportunity, err := repositories.Opportunities.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find opportunity: %w", err)
		}
		if opportunity == nil {
			return apperrors.NewNotFound("opportunity not found")
		}
		action, err := change(opportunity, now)
		if err != nil {
			return err
		}
		opportunity.UpdatedAt = now
		if err := repositories.Opportunities.Update(ctx, opportunity); err != nil {
			return fmt.Errorf("update opportunity: %w", err)
		}
		return audit(ctx, repositories, actorID, action, opportunity.ID, map[string]string{"status": string(opportunity.Status)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id)
}

// checkLiveEdit keeps what Students rely on once an Opportunity is out: a
// closed one no longer changes, and a published one keeps its application
// mode and a deadline still ahead.
func checkLiveEdit(current Opportunity, changed content, now time.Time) error {
	switch current.Status {
	case StatusClosed:
		return apperrors.NewConflict("a closed opportunity can't be edited")
	case StatusPublished:
		if changed.ApplicationMode != current.ApplicationMode {
			return apperrors.NewConflict("a published opportunity keeps its application mode")
		}
		if !changed.ApplyBy.Equal(current.ApplyBy) && !changed.ApplyBy.After(now) {
			return apperrors.NewValidation("invalid opportunity", map[string]string{"apply_by": "must stay in the future while it is published"})
		}
	}
	return nil
}
