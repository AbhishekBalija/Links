package events

// Changes after publishing: logistics edits and cancelling (ADR 0023).

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const afterPublishing = "can't change after publishing; cancel the event and propose it again"

// editLogistics changes a published Event's description, location, times or
// capacity. Only its organisers may, and anything else needs a new proposal,
// since that would change what the reviewers approved.
func (s *Service) editLogistics(ctx context.Context, repositories Repositories, actorID string, event *Event, input UpdateEventInput, now time.Time) error {
	organiser, err := s.organises(ctx, actorID, *event)
	if err != nil {
		return err
	}
	if !organiser {
		return apperrors.NewNotFound("event not found")
	}
	if !event.EndsAt.After(now) {
		return apperrors.NewConflict("the event is over")
	}

	details := map[string]string{}
	if input.Title != nil {
		details["title"] = afterPublishing
	}
	if input.EventType != nil {
		details["event_type"] = afterPublishing
	}
	if input.DepartmentID.Set {
		details["department_id"] = afterPublishing
	}
	if input.FacultyMentorID.Set {
		details["faculty_mentor_id"] = afterPublishing
	}
	if input.Audience != nil {
		details["audience"] = afterPublishing
	}
	if len(details) > 0 {
		return apperrors.NewValidation("only logistics can change after publishing", details)
	}

	changed := []string{}
	if input.Description != nil {
		event.Description = strings.TrimSpace(*input.Description)
		changed = append(changed, "description")
		if utf8.RuneCountInString(event.Description) > maxDescriptionLength {
			details["description"] = fmt.Sprintf("use at most %d characters", maxDescriptionLength)
		}
	}
	if input.Location != nil {
		event.Location = strings.TrimSpace(*input.Location)
		changed = append(changed, "location")
		if length := utf8.RuneCountInString(event.Location); length < 1 || length > maxLocationLength {
			details["location"] = fmt.Sprintf("use 1 to %d characters", maxLocationLength)
		}
	}
	if input.StartsAt != nil {
		event.StartsAt = *input.StartsAt
		changed = append(changed, "starts_at")
		if !event.StartsAt.After(now) {
			details["starts_at"] = "must be in the future"
		}
	}
	if input.EndsAt != nil {
		event.EndsAt = *input.EndsAt
		changed = append(changed, "ends_at")
	}
	if !event.EndsAt.After(event.StartsAt) {
		details["ends_at"] = "must be after starts_at"
	}
	if input.Capacity.Set {
		event.Capacity = input.Capacity.Value
		changed = append(changed, "capacity")
		if event.Capacity != nil && (*event.Capacity < 1 || *event.Capacity > maxCapacity) {
			details["capacity"] = fmt.Sprintf("use a whole number from 1 to %d, or null for no limit", maxCapacity)
		}
	}
	if event.Capacity != nil && details["capacity"] == "" {
		counts, err := repositories.Events.RSVPCounts(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("count rsvps: %w", err)
		}
		if going := counts[RSVPGoing]; going > *event.Capacity {
			details["capacity"] = fmt.Sprintf("%d people are already going", going)
		}
	}
	if len(details) > 0 {
		return apperrors.NewValidation("invalid event", details)
	}

	event.UpdatedAt = now
	if err := repositories.Events.Update(ctx, event); err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	sort.Strings(changed)
	return audit(ctx, repositories, actorID, "event_logistics_updated", event.ID, map[string]string{"fields": strings.Join(changed, ",")}, now)
}

// DeleteDraft removes the proposer's own draft. Only drafts can go: once an
// Event is submitted, reviewers have seen it, so it is cancelled instead and
// its history kept.
func (s *Service) DeleteDraft(ctx context.Context, actorID, id string) error {
	now := s.now()
	return s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event == nil || event.ProposerID != actorID {
			return apperrors.NewNotFound("event not found")
		}
		if event.Status != StatusDraft {
			return apperrors.NewConflict("only a draft can be deleted; cancel a submitted event instead")
		}
		if err := repositories.Events.DeleteDraft(ctx, event.ID); err != nil {
			return fmt.Errorf("delete draft: %w", err)
		}
		return audit(ctx, repositories, actorID, "event_draft_deleted", event.ID, map[string]string{"title": event.Title}, now)
	})
}

// Cancel calls off an Event that isn't over, rejected or already cancelled.
// Its RSVPs are kept, so the people who answered can see it was cancelled.
func (s *Service) Cancel(ctx context.Context, actorID, id string, input CancelInput) (*EventResponse, error) {
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, apperrors.NewValidation("a reason is required", map[string]string{"reason": "say why the event is cancelled"})
	}
	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event == nil {
			return apperrors.NewNotFound("event not found")
		}
		organiser, err := s.organises(ctx, actorID, *event)
		if err != nil {
			return err
		}
		if !organiser {
			if _, err := s.access(ctx, actorID, id); err != nil {
				return err
			}
			return apperrors.NewForbidden("only the event's proposer or its reviewers can cancel it")
		}
		switch event.Status {
		case StatusCancelled, StatusHODRejected, StatusFinalRejected:
			return apperrors.NewConflict("the event can't be cancelled in status " + string(event.Status))
		}
		if !event.EndsAt.After(now) {
			return apperrors.NewConflict("the event is over")
		}

		event.Status = StatusCancelled
		event.CancelledAt = &now
		event.CancelledBy = &actorID
		event.CancelReason = &reason
		event.UpdatedAt = now
		if err := repositories.Events.Update(ctx, event); err != nil {
			return fmt.Errorf("cancel event: %w", err)
		}
		return audit(ctx, repositories, actorID, "event_cancelled", event.ID, map[string]string{"reason": reason}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id)
}

// organises is true for the Event's Organiser and for whoever reviews it: the
// Department's HOD, the principal and admins.
func (s *Service) organises(ctx context.Context, actorID string, event Event) (bool, error) {
	if event.OrganiserID == actorID {
		return true, nil
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return false, err
	}
	return privileged(grants) || hodOf(grants, event.DepartmentID), nil
}
