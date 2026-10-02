package events

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// HandOverOn is the events' share of ending a role (ADR 0028), built on the
// role change's transaction.
func HandOverOn(tx *gorm.DB) auth.UnfinishedWork {
	return handover{
		repositories: Repositories{Events: NewGormRepository(tx), AuditLogs: auth.NewGormAuditLogRepository(tx)},
		users:        auth.NewGormUserRepository(tx),
	}
}

type handover struct {
	repositories Repositories
	users        auth.UserRepository
}

// HandOver looks at the person's Event proposals under review or sent back,
// and the Events they organise that aren't over. Whatever their other roles
// can't propose any more is handed on: a proposal returns to a private
// draft, and a published Event moves to a new Organiser (the one picked, or
// else the Department's HOD). Drafts stay as they are.
func (h handover) HandOver(ctx context.Context, change auth.Handover) (auth.HandoverSummary, error) {
	summary := auth.HandoverSummary{}
	remaining := grantsOf(change.Remaining)
	unfinished, err := h.repositories.Events.UnfinishedOf(ctx, change.PersonID, change.At)
	if err != nil {
		return summary, fmt.Errorf("find unfinished events: %w", err)
	}
	var picked *auth.PersonRef
	var pickedGrants []Grant
	if change.OrganiserID != "" {
		picked, pickedGrants, err = h.organiser(ctx, change)
		if err != nil {
			return summary, err
		}
	}

	for _, event := range unfinished {
		if _, problem := proposalProblem(remaining, event.EventType, event.DepartmentID); problem == "" {
			continue
		}
		if event.Status != StatusPublished {
			if err := h.returnToDraft(ctx, change, &event); err != nil {
				return summary, err
			}
			summary.ReturnedEvents++
			continue
		}

		moved := auth.MovedEvent{ID: event.ID, Title: event.Title, StartsAt: event.StartsAt}
		if picked != nil {
			if _, problem := proposalProblem(pickedGrants, event.EventType, event.DepartmentID); problem != "" {
				return summary, apperrors.NewValidation("the organiser can't run this event", map[string]string{
					"organiser_id": fmt.Sprintf("%s can't run %q; pick someone who could propose it", picked.FullName, event.Title),
				})
			}
			moved.Organiser = picked
		} else if event.DepartmentID != nil {
			moved.Organiser, err = h.repositories.Events.DepartmentHOD(ctx, *event.DepartmentID, change.PersonID)
			if err != nil {
				return summary, fmt.Errorf("find department HOD: %w", err)
			}
		}
		summary.MovedEvents = append(summary.MovedEvents, moved)
		if moved.Organiser == nil {
			summary.OrganiserNeeded = true
			continue
		}
		if err := h.moveTo(ctx, change, &event, moved.Organiser.UserID); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

// organiser checks the picked Organiser is someone else, active, and returns
// their name and the roles they hold now.
func (h handover) organiser(ctx context.Context, change auth.Handover) (*auth.PersonRef, []Grant, error) {
	invalid := apperrors.NewValidation("invalid organiser", map[string]string{"organiser_id": "pick an active member other than the person whose role ends"})
	if change.OrganiserID == change.PersonID {
		return nil, nil, invalid
	}
	user, err := h.users.FindByID(ctx, change.OrganiserID)
	if err != nil {
		return nil, nil, fmt.Errorf("find organiser: %w", err)
	}
	if user == nil || user.Status != auth.UserStatusActive {
		return nil, nil, invalid
	}
	assignments, err := h.users.GetRoleAssignments(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("load organiser roles: %w", err)
	}
	person := &auth.PersonRef{UserID: user.ID}
	if user.Profile != nil {
		person.FullName = user.Profile.FullName
	}
	return person, grantsOf(assignments), nil
}

// returnToDraft takes a proposal out of review. Its review notes are kept.
func (h handover) returnToDraft(ctx context.Context, change auth.Handover, event *Event) error {
	from := event.Status
	event.Status, event.SubmittedAt, event.UpdatedAt = StatusDraft, nil, change.At
	if err := h.repositories.Events.Update(ctx, event); err != nil {
		return fmt.Errorf("return event to draft: %w", err)
	}
	return audit(ctx, h.repositories, change.ActorID, "event_returned_to_draft", event.ID, map[string]string{"from": string(from), "reason": "role_ended"}, change.At)
}

func (h handover) moveTo(ctx context.Context, change auth.Handover, event *Event, organiserID string) error {
	from := event.OrganiserID
	event.OrganiserID, event.UpdatedAt = organiserID, change.At
	if err := h.repositories.Events.Update(ctx, event); err != nil {
		return fmt.Errorf("change event organiser: %w", err)
	}
	return audit(ctx, h.repositories, change.ActorID, "event_organiser_changed", event.ID, map[string]string{"from": from, "to": organiserID, "reason": "role_ended"}, change.At)
}

// grantsOf turns role assignments into the Grants the authority rules read.
func grantsOf(assignments []auth.RoleAssignment) []Grant {
	grants := make([]Grant, 0, len(assignments))
	for _, assignment := range assignments {
		grant := Grant{Role: assignment.Role}
		if assignment.ScopeType == auth.ScopeDepartment && assignment.ScopeID != nil {
			grant.DepartmentID = *assignment.ScopeID
		}
		grants = append(grants, grant)
	}
	return grants
}

// unfinishedStatuses are proposals under review or sent back.
var unfinishedStatuses = []Status{StatusSubmitted, StatusHODChangesRequested, StatusHODApproved, StatusFinalChangesRequested}

// UnfinishedOf locks the person's proposals under review or sent back, and
// the published Events they organise that aren't over at the given time.
func (r *GormRepository) UnfinishedOf(ctx context.Context, personID string, at time.Time) ([]Event, error) {
	var events []Event
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(`(proposer_id = ? AND status IN ?) OR (organiser_id = ? AND status = ? AND ends_at > ?)`,
			personID, unfinishedStatuses, personID, StatusPublished, at).
		Order("starts_at, id").
		Find(&events).Error
	return events, err
}

// DepartmentHOD is the Department's HOD now, other than the given user, or
// nil when there is none.
func (r *GormRepository) DepartmentHOD(ctx context.Context, departmentID, exceptUserID string) (*auth.PersonRef, error) {
	var people []auth.PersonRef
	err := r.db.WithContext(ctx).Raw(`
		SELECT h.user_id, p.full_name
		FROM role_assignments h
		JOIN profiles p ON p.user_id = h.user_id
		WHERE h.role = 'hod' AND h.scope_type = 'department' AND h.scope_id = ? AND h.user_id <> ?
		  AND h.starts_at <= now() AND (h.ends_at IS NULL OR h.ends_at > now())
		LIMIT 1`, departmentID, exceptUserID).Scan(&people).Error
	if err != nil || len(people) == 0 {
		return nil, err
	}
	return &people[0], nil
}
