package announcements

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/AbhishekBalija/Links/server/internal/auth"
)

// HandOverOn is the announcements' share of ending a role (ADR 0028), built
// on the role change's transaction.
func HandOverOn(tx *gorm.DB) auth.UnfinishedWork {
	return handover{repositories: Repositories{
		Announcements: NewGormRepository(tx),
		AuditLogs:     auth.NewGormAuditLogRepository(tx),
	}}
}

type handover struct {
	repositories Repositories
}

// HandOver withdraws the person's Announcements waiting for approval or sent
// back, and closes their edits waiting on published ones, when their other
// roles can't post that category. Drafts and published versions stay.
func (h handover) HandOver(ctx context.Context, change auth.Handover) (auth.HandoverSummary, error) {
	summary := auth.HandoverSummary{}
	grants := grantsOf(change.Remaining)
	unfinished, err := h.repositories.Announcements.UnfinishedOf(ctx, change.PersonID)
	if err != nil {
		return summary, fmt.Errorf("find unfinished announcements: %w", err)
	}
	for _, announcement := range unfinished {
		if CanPost(grants, announcement.Category) {
			continue
		}
		open, err := h.repositories.Announcements.OpenRevision(ctx, announcement.ID)
		if err != nil {
			return summary, err
		}
		if open != nil {
			open.Status, open.UpdatedAt = RevisionClosed, change.At
			if err := h.repositories.Announcements.UpdateRevision(ctx, open); err != nil {
				return summary, err
			}
		}
		action := "announcement.edit_closed"
		if announcement.Status == StatusPublished {
			summary.ClosedEdits = append(summary.ClosedEdits, auth.WorkItem{ID: announcement.ID, Title: announcement.Title})
		} else {
			announcement.Status, announcement.UpdatedAt = StatusWithdrawn, change.At
			if err := h.repositories.Announcements.UpdateAnnouncement(ctx, &announcement); err != nil {
				return summary, err
			}
			action = "announcement.withdrawn"
			summary.WithdrawnAnnouncements = append(summary.WithdrawnAnnouncements, auth.WorkItem{ID: announcement.ID, Title: announcement.Title})
		}
		if err := audit(ctx, h.repositories, change.ActorID, action, announcement.ID, map[string]interface{}{"reason": "role_ended"}); err != nil {
			return summary, err
		}
	}
	return summary, nil
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
