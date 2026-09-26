package announcements

import (
	"context"
	"fmt"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// editPublished changes a published Announcement. With Publishing authority
// over the new Audience the edit applies at once; otherwise it becomes a
// pending revision and readers keep seeing the approved version. A rejected
// edit is reused, so fixing it resubmits it. Only one edit waits at a time.
func (s *Service) editPublished(ctx context.Context, repositories Repositories, grants []Grant, actorID string, announcement *Announcement, content announcementContent) error {
	open, err := repositories.Announcements.OpenRevision(ctx, announcement.ID)
	if err != nil {
		return err
	}
	if open != nil && open.Status == RevisionPending {
		return apperrors.NewConflict("an edit to this announcement is already waiting for approval")
	}
	if checkErr := checkDepartments(ctx, repositories.Announcements, content.Audience); checkErr != nil {
		return checkErr
	}

	now := s.now().UTC()
	decision := DecidePublishing(grants, content.Category, content.Audience)
	if decision.PublishDirectly {
		if open != nil {
			// The direct edit replaces the old edit, which keeps its own history.
			open.Status, open.UpdatedAt = RevisionClosed, now
			if saveErr := repositories.Announcements.UpdateRevision(ctx, open); saveErr != nil {
				return saveErr
			}
		}
		if applyErr := applyContent(ctx, repositories, announcement, content, now); applyErr != nil {
			return applyErr
		}
		return audit(ctx, repositories, actorID, "announcement.edited", announcement.ID, nil)
	}

	if open != nil {
		open.setContent(content, now)
		open.submit(decision, now)
		if saveErr := repositories.Announcements.UpdateRevision(ctx, open); saveErr != nil {
			return saveErr
		}
	} else {
		revision := content.revision(announcement.ID, actorID, now)
		revision.submit(decision, now)
		if createErr := repositories.Announcements.CreateRevision(ctx, &revision); createErr != nil {
			return createErr
		}
	}
	return audit(ctx, repositories, actorID, "announcement.submitted", announcement.ID, map[string]interface{}{"edit": true})
}

// Withdraw takes a published Announcement down. Its author may withdraw it, and
// so may whoever would approve it: the Department's HOD for a single-Department
// Audience, or the principal or an admin. An edit still waiting is closed.
func (s *Service) Withdraw(ctx context.Context, actorID, id string) (*AnnouncementResponse, error) {
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}

	var withdrawn Announcement
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		announcement, findErr := repositories.Announcements.FindForUpdate(ctx, id)
		if findErr != nil {
			return findErr
		}
		isAuthor := announcement != nil && announcement.PublisherID == actorID
		if announcement == nil || (announcement.Status != StatusPublished && !isAuthor) {
			// Unpublished Announcements are private to their author.
			return apperrors.NewNotFound("announcement not found")
		}
		if announcement.Status != StatusPublished {
			return apperrors.NewConflict("only a published announcement can be withdrawn")
		}
		if !isAuthor {
			audience, audienceErr := repositories.Announcements.Audience(ctx, announcement.ID)
			if audienceErr != nil {
				return audienceErr
			}
			approver := DecidePublishing(nil, announcement.Category, audience)
			if !canApprove(grants, &Revision{ApproverDepartmentID: departmentPointer(approver.ApproverDepartmentID)}) {
				return apperrors.NewForbidden("you can't withdraw this announcement")
			}
		}

		now := s.now().UTC()
		open, openErr := repositories.Announcements.OpenRevision(ctx, announcement.ID)
		if openErr != nil {
			return openErr
		}
		if open != nil {
			// Closed, not rejected: a rejected revision can be resubmitted, and a
			// withdrawn Announcement must never come back that way.
			open.Status, open.UpdatedAt = RevisionClosed, now
			if saveErr := repositories.Announcements.UpdateRevision(ctx, open); saveErr != nil {
				return saveErr
			}
		}
		announcement.Status, announcement.UpdatedAt = StatusWithdrawn, now
		if saveErr := repositories.Announcements.UpdateAnnouncement(ctx, announcement); saveErr != nil {
			return saveErr
		}
		withdrawn = *announcement
		return audit(ctx, repositories, actorID, "announcement.withdrawn", announcement.ID, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("withdraw announcement: %w", err)
	}
	return s.single(ctx, actorID, withdrawn)
}

func departmentPointer(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

// audienceFromViews turns stored Audience rules back into rules.
func audienceFromViews(views []AudienceRuleView) []AudienceRule {
	rules := make([]AudienceRule, 0, len(views))
	for _, view := range views {
		rule := AudienceRule{DepartmentID: view.DepartmentID, BatchYear: view.BatchYear}
		if view.Role != nil {
			role := auth.Role(*view.Role)
			rule.Role = &role
		}
		rules = append(rules, rule)
	}
	return rules
}
