package events

// Submitting an Event and the two review stages (ADR 0023). Every step locks
// the Event row, so a second decision on the same Event sees the first and
// gets a conflict.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Submit sends the proposer's draft, or an Event sent back for changes, to
// the review stage it needs.
func (s *Service) Submit(ctx context.Context, actorID, id string) (*EventResponse, error) {
	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event == nil || event.ProposerID != actorID {
			return apperrors.NewNotFound("event not found")
		}
		if !editable(event.Status) {
			return apperrors.NewConflict("only a draft or an event sent back for changes can be submitted")
		}
		return s.submitLocked(ctx, repositories, actorID, event, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id)
}

// submitLocked moves a locked Event to its next status. The proposer's roles
// are read again, since they may have changed since the draft was saved.
func (s *Service) submitLocked(ctx context.Context, repositories Repositories, actorID string, event *Event, now time.Time) error {
	if !event.StartsAt.After(now) {
		return apperrors.NewValidation("invalid event", map[string]string{"starts_at": "must be in the future to submit"})
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return err
	}
	if field, problem := proposalProblem(grants, event.EventType, event.DepartmentID); problem != "" {
		return apperrors.NewValidation("you can't propose this event", map[string]string{field: problem})
	}
	event.Status = nextStatus(grants, *event)
	event.SubmittedAt = &now
	if event.Status == StatusPublished {
		event.PublishedAt = &now
	}
	event.UpdatedAt = now
	if err := repositories.Events.Update(ctx, event); err != nil {
		return fmt.Errorf("submit event: %w", err)
	}
	return audit(ctx, repositories, actorID, "event_submitted", event.ID, map[string]string{"status": string(event.Status)}, now)
}

// Review records an HOD review or final approval decision.
func (s *Service) Review(ctx context.Context, actorID, id string, stage Stage, input ReviewInput) (*EventResponse, error) {
	decision := Decision(input.Decision)
	note := strings.TrimSpace(input.Note)
	switch decision {
	case DecisionApprove, DecisionRequestChanges, DecisionReject:
	default:
		return nil, apperrors.NewValidation("invalid decision", map[string]string{"decision": "use approve, request_changes or reject"})
	}
	if decision != DecisionApprove && note == "" {
		return nil, apperrors.NewValidation("a note is required", map[string]string{"note": "say what to change or why it is rejected"})
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event == nil {
			return apperrors.NewNotFound("event not found")
		}
		if err := s.mayReview(ctx, repositories, grants, stage, event); err != nil {
			return err
		}
		if event.ProposerID == actorID {
			return apperrors.NewForbidden("you can't review your own event")
		}
		waiting := StatusSubmitted
		if stage == StageFinal {
			waiting = StatusHODApproved
		}
		if event.Status != waiting {
			return apperrors.NewConflict("the event isn't waiting for this review")
		}
		if stage == StageFinal && decision == DecisionApprove && !event.StartsAt.After(now) {
			return apperrors.NewConflict("the event has already started")
		}

		event.Status = afterDecision(stage, decision)
		if event.Status == StatusPublished {
			event.PublishedAt = &now
		}
		event.UpdatedAt = now
		if err := repositories.Events.Update(ctx, event); err != nil {
			return fmt.Errorf("update event: %w", err)
		}
		review := &Review{EventID: event.ID, Stage: stage, ReviewerID: actorID, Decision: decision, CreatedAt: now}
		if note != "" {
			review.Note = &note
		}
		if err := repositories.Events.CreateReview(ctx, review); err != nil {
			return fmt.Errorf("record review: %w", err)
		}
		return audit(ctx, repositories, actorID, "event_reviewed", event.ID,
			map[string]string{"stage": string(stage), "decision": string(decision), "status": string(event.Status)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id)
}

// mayReview checks the reviewer's authority for the stage: the HOD stage
// belongs to the Event's Department HOD, or to the principal or an admin when
// that Department has none; final approval to the principal or an admin.
func (s *Service) mayReview(ctx context.Context, repositories Repositories, grants []Grant, stage Stage, event *Event) error {
	forbidden := apperrors.NewForbidden("you are not a reviewer for this event")
	if stage == StageFinal {
		if !privileged(grants) {
			return forbidden
		}
		return nil
	}
	if hodOf(grants, event.DepartmentID) {
		return nil
	}
	if !privileged(grants) {
		return forbidden
	}
	if event.DepartmentID == nil {
		return nil
	}
	hasHOD, err := repositories.Events.DepartmentHasHOD(ctx, *event.DepartmentID)
	if err != nil {
		return fmt.Errorf("check department HOD: %w", err)
	}
	if hasHOD {
		return forbidden
	}
	return nil
}

func afterDecision(stage Stage, decision Decision) Status {
	switch {
	case stage == StageHOD && decision == DecisionApprove:
		return StatusHODApproved
	case stage == StageHOD && decision == DecisionRequestChanges:
		return StatusHODChangesRequested
	case stage == StageHOD:
		return StatusHODRejected
	case decision == DecisionApprove:
		return StatusPublished
	case decision == DecisionRequestChanges:
		return StatusFinalChangesRequested
	default:
		return StatusFinalRejected
	}
}

// Queue lists what waits for the caller at either stage, oldest first.
func (s *Service) Queue(ctx context.Context, actorID, cursor string, limit int) ([]EventResponse, *ListMeta, error) {
	limit, err := checkLimit(limit)
	if err != nil {
		return nil, nil, err
	}
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	scope, err := s.reviewerScope(ctx, actorID)
	if err != nil {
		return nil, nil, err
	}
	views, err := s.repository.Queue(ctx, scope, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list review queue: %w", err)
	}
	meta := &ListMeta{}
	if len(views) > limit {
		views = views[:limit]
		last := views[len(views)-1]
		meta.NextCursor = encodeCursor(Cursor{At: *last.SubmittedAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, views)
	if err != nil {
		return nil, nil, err
	}
	for i := range responses {
		responses[i].Stage = StageHOD
		if responses[i].Status == StatusHODApproved {
			responses[i].Stage = StageFinal
		}
	}
	return responses, meta, nil
}

// ReviewSummary is what waits for a reviewer.
type ReviewSummary struct {
	PendingCount      int
	OldestSubmittedAt *time.Time
}

// ReviewSummary counts what waits for the caller in the review queue, or
// returns nil if they review no Events.
func (s *Service) ReviewSummary(ctx context.Context, actorID string) (*ReviewSummary, error) {
	scope, err := s.reviewerScope(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !scope.All && len(scope.HODDepartments) == 0 {
		return nil, nil
	}
	count, oldest, err := s.repository.QueueSummary(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("summarize review queue: %w", err)
	}
	return &ReviewSummary{PendingCount: count, OldestSubmittedAt: oldest}, nil
}

// reviewerScope is what the caller reviews: the HOD stage of their
// Departments and, as principal or admin, everything else.
func (s *Service) reviewerScope(ctx context.Context, actorID string) (ReviewerScope, error) {
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return ReviewerScope{}, err
	}
	scope := ReviewerScope{UserID: actorID, All: privileged(grants), HODDepartments: []string{}}
	for _, grant := range grants {
		if grant.Role == auth.RoleHOD && grant.DepartmentID != "" {
			scope.HODDepartments = append(scope.HODDepartments, grant.DepartmentID)
		}
	}
	return scope, nil
}
