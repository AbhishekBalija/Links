package announcements

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const maxReviewNoteLength = 1000

// announcementContent is validated Announcement content.
type announcementContent struct {
	Title     string
	Body      string
	Category  Category
	Audience  []AudienceRule
	ExpiresAt *time.Time
}

// revision starts a draft revision holding this content.
func (c announcementContent) revision(announcementID, authorID string, now time.Time) Revision {
	stored := make([]StoredRule, 0, len(c.Audience))
	for _, rule := range c.Audience {
		entry := StoredRule{DepartmentID: rule.DepartmentID, BatchYear: rule.BatchYear}
		if rule.Role != nil {
			role := string(*rule.Role)
			entry.Role = &role
		}
		stored = append(stored, entry)
	}
	return Revision{
		AnnouncementID: announcementID,
		Title:          c.Title,
		Body:           c.Body,
		Category:       c.Category,
		Audience:       stored,
		ExpiresAt:      c.ExpiresAt,
		Status:         RevisionDraft,
		SubmittedBy:    authorID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// submit sends the revision for approval to the approver the decision names.
func (r *Revision) submit(decision Decision, now time.Time) {
	r.Status = RevisionPending
	r.SubmittedAt = &now
	r.ApproverDepartmentID = nil
	if decision.ApproverDepartmentID != "" {
		department := decision.ApproverDepartmentID
		r.ApproverDepartmentID = &department
	}
	r.ReviewedBy, r.ReviewedAt, r.ReviewNote = nil, nil, nil
	r.UpdatedAt = now
}

// content reads a revision's content back as validated Announcement content.
func (r Revision) content() announcementContent {
	audience := make([]AudienceRule, 0, len(r.Audience))
	for _, stored := range r.Audience {
		rule := AudienceRule{DepartmentID: stored.DepartmentID, BatchYear: stored.BatchYear}
		if stored.Role != nil {
			role := auth.Role(*stored.Role)
			rule.Role = &role
		}
		audience = append(audience, rule)
	}
	return announcementContent{Title: r.Title, Body: r.Body, Category: r.Category, Audience: audience, ExpiresAt: r.ExpiresAt}
}

// Update replaces the content of the author's draft or rejected Announcement.
func (s *Service) Update(ctx context.Context, actorID, id string, input UpdateAnnouncementInput) (*AnnouncementResponse, error) {
	content, err := s.validate(input.contentInput)
	if err != nil {
		return nil, err
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !CanPost(grants, content.Category) {
		return nil, apperrors.NewForbidden("you can't post this kind of announcement")
	}

	var updated Announcement
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		announcement, revision, loadErr := authoredWithRevision(ctx, repositories, actorID, id)
		if loadErr != nil {
			return loadErr
		}
		switch announcement.Status {
		case StatusDraft, StatusRejected:
		case StatusPending:
			return apperrors.NewConflict("this announcement is waiting for approval and can't be edited now")
		default:
			return apperrors.NewConflict("editing a published announcement isn't available yet")
		}
		if checkErr := checkDepartments(ctx, repositories.Announcements, content.Audience); checkErr != nil {
			return checkErr
		}

		now := s.now().UTC()
		announcement.Title, announcement.Body, announcement.Category = content.Title, content.Body, content.Category
		announcement.ExpiresAt, announcement.UpdatedAt = content.ExpiresAt, now
		if saveErr := repositories.Announcements.UpdateAnnouncement(ctx, announcement); saveErr != nil {
			return saveErr
		}
		if audienceErr := repositories.Announcements.ReplaceAudience(ctx, announcement.ID, content.Audience); audienceErr != nil {
			return audienceErr
		}
		fresh := content.revision(announcement.ID, actorID, now)
		revision.Title, revision.Body, revision.Category = fresh.Title, fresh.Body, fresh.Category
		revision.Audience, revision.ExpiresAt, revision.UpdatedAt = fresh.Audience, fresh.ExpiresAt, now
		if revisionErr := repositories.Announcements.UpdateRevision(ctx, revision); revisionErr != nil {
			return revisionErr
		}
		updated = *announcement
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return s.single(ctx, updated)
}

// Submit sends the author's draft or rejected Announcement for approval, or
// publishes it straight away when the author has Publishing authority.
func (s *Service) Submit(ctx context.Context, actorID, id string) (*AnnouncementResponse, error) {
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}

	var submitted Announcement
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		announcement, revision, loadErr := authoredWithRevision(ctx, repositories, actorID, id)
		if loadErr != nil {
			return loadErr
		}
		if revision.Status != RevisionDraft && revision.Status != RevisionRejected {
			return apperrors.NewConflict("this announcement has nothing to submit")
		}
		content := revision.content()
		if !CanPost(grants, content.Category) {
			return apperrors.NewForbidden("you can't post this kind of announcement")
		}
		if checkErr := checkDepartments(ctx, repositories.Announcements, content.Audience); checkErr != nil {
			return checkErr
		}

		now := s.now().UTC()
		decision := DecidePublishing(grants, content.Category, content.Audience)
		if decision.PublishDirectly {
			revision.Status, revision.SubmittedAt, revision.UpdatedAt = RevisionApproved, &now, now
			if publishErr := publishRevision(ctx, repositories, announcement, revision, now); publishErr != nil {
				return publishErr
			}
			submitted = *announcement
			return audit(ctx, repositories, actorID, "announcement.published", announcement.ID, nil)
		}

		revision.submit(decision, now)
		if revisionErr := repositories.Announcements.UpdateRevision(ctx, revision); revisionErr != nil {
			return revisionErr
		}
		if announcement.Status != StatusPublished {
			announcement.Status, announcement.UpdatedAt = StatusPending, now
			if saveErr := repositories.Announcements.UpdateAnnouncement(ctx, announcement); saveErr != nil {
				return saveErr
			}
		}
		submitted = *announcement
		return audit(ctx, repositories, actorID, "announcement.submitted", announcement.ID, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("submit announcement: %w", err)
	}
	return s.single(ctx, submitted)
}

// Review approves or rejects an Announcement waiting for approval. Only its
// approver may act: the Department's HOD for a single-Department Audience, or
// the principal or an admin for anything. Nobody reviews their own submission.
func (s *Service) Review(ctx context.Context, actorID, id string, input ReviewInput) (*AnnouncementResponse, error) {
	note := strings.TrimSpace(input.Note)
	switch input.Decision {
	case "approve":
	case "reject":
		if note == "" {
			return nil, apperrors.NewValidation("invalid review", map[string]string{"note": "say what the author should change"})
		}
	default:
		return nil, apperrors.NewValidation("invalid review", map[string]string{"decision": "use approve or reject"})
	}
	if utf8.RuneCountInString(note) > maxReviewNoteLength {
		return nil, apperrors.NewValidation("invalid review", map[string]string{"note": fmt.Sprintf("use at most %d characters", maxReviewNoteLength)})
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}

	var reviewed Announcement
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		announcement, findErr := repositories.Announcements.FindForUpdate(ctx, id)
		if findErr != nil {
			return findErr
		}
		if announcement == nil {
			return apperrors.NewNotFound("announcement not found")
		}
		revision, revisionErr := repositories.Announcements.OpenRevision(ctx, announcement.ID)
		if revisionErr != nil {
			return revisionErr
		}
		if revision == nil || revision.Status != RevisionPending {
			return apperrors.NewConflict("this announcement isn't waiting for approval")
		}
		if revision.SubmittedBy == actorID || !canApprove(grants, revision) {
			return apperrors.NewForbidden("you can't review this announcement")
		}

		now := s.now().UTC()
		revision.ReviewedBy, revision.ReviewedAt, revision.UpdatedAt = &actorID, &now, now
		if input.Decision == "reject" {
			revision.Status, revision.ReviewNote = RevisionRejected, &note
			if saveErr := repositories.Announcements.UpdateRevision(ctx, revision); saveErr != nil {
				return saveErr
			}
			if announcement.Status == StatusPending {
				announcement.Status, announcement.UpdatedAt = StatusRejected, now
				if saveErr := repositories.Announcements.UpdateAnnouncement(ctx, announcement); saveErr != nil {
					return saveErr
				}
			}
			reviewed = *announcement
			return audit(ctx, repositories, actorID, "announcement.rejected", announcement.ID, map[string]interface{}{"note": note})
		}

		if checkErr := checkDepartments(ctx, repositories.Announcements, revision.content().Audience); checkErr != nil {
			return checkErr
		}
		revision.Status = RevisionApproved
		if note != "" {
			revision.ReviewNote = &note
		}
		if publishErr := publishRevision(ctx, repositories, announcement, revision, now); publishErr != nil {
			return publishErr
		}
		reviewed = *announcement
		return audit(ctx, repositories, actorID, "announcement.approved", announcement.ID, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("review announcement: %w", err)
	}
	return s.single(ctx, reviewed)
}

// Queue lists the Announcements waiting for this approver, oldest first.
func (s *Service) Queue(ctx context.Context, actorID, cursor string, limit int) ([]AnnouncementResponse, *FeedMeta, error) {
	limit = pageLimit(limit)
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, nil, err
	}
	scope := ApproverScope{UserID: actorID}
	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin:
			scope.All = true
		case auth.RoleHOD:
			if grant.DepartmentID != "" {
				scope.DepartmentIDs = append(scope.DepartmentIDs, grant.DepartmentID)
			}
		}
	}
	if !scope.All && len(scope.DepartmentIDs) == 0 {
		return nil, nil, apperrors.NewForbidden("you don't approve announcements")
	}

	entries, err := s.repository.Queue(ctx, scope, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("load approval queue: %w", err)
	}
	meta := &FeedMeta{}
	if len(entries) > limit {
		entries = entries[:limit]
		last := entries[len(entries)-1]
		meta.NextCursor = encodeCursor(FeedCursor{PublishedAt: *last.SubmittedAt, ID: last.ID})
	}
	responses, err := s.queueResponses(ctx, entries)
	if err != nil {
		return nil, nil, err
	}
	return responses, meta, nil
}

// Mine lists the author's own Announcements in any status, newest first, with
// the reviewer's note on rejected ones.
func (s *Service) Mine(ctx context.Context, actorID, cursor string, limit int) ([]AnnouncementResponse, *FeedMeta, error) {
	limit = pageLimit(limit)
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.repository.Authored(ctx, actorID, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("load authored announcements: %w", err)
	}
	meta := &FeedMeta{}
	if len(entries) > limit {
		entries = entries[:limit]
		last := entries[len(entries)-1]
		meta.NextCursor = encodeCursor(FeedCursor{PublishedAt: last.CreatedAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, entries)
	if err != nil {
		return nil, nil, err
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	revisions, err := s.repository.LatestRevisions(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("load revisions: %w", err)
	}
	notes := map[string]*string{}
	for _, revision := range revisions {
		if revision.Status == RevisionRejected {
			notes[revision.AnnouncementID] = revision.ReviewNote
		}
	}
	for i := range responses {
		responses[i].ReviewNote = notes[responses[i].ID]
	}
	return responses, meta, nil
}

// canApprove reports whether the grants cover this revision's approver.
func canApprove(grants []Grant, revision *Revision) bool {
	for _, grant := range grants {
		switch grant.Role {
		case auth.RolePrincipal, auth.RoleAdmin:
			return true
		case auth.RoleHOD:
			if grant.DepartmentID != "" && revision.ApproverDepartmentID != nil && *revision.ApproverDepartmentID == grant.DepartmentID {
				return true
			}
		}
	}
	return false
}

// authoredWithRevision locks the author's Announcement and loads its open
// revision. Someone else's Announcement looks like it doesn't exist.
func authoredWithRevision(ctx context.Context, repositories Repositories, actorID, id string) (*Announcement, *Revision, error) {
	announcement, err := repositories.Announcements.FindForUpdate(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if announcement == nil || announcement.PublisherID != actorID {
		return nil, nil, apperrors.NewNotFound("announcement not found")
	}
	revision, err := repositories.Announcements.OpenRevision(ctx, announcement.ID)
	if err != nil {
		return nil, nil, err
	}
	if revision == nil {
		return nil, nil, apperrors.NewConflict("this announcement has nothing to change")
	}
	return announcement, revision, nil
}

// publishRevision copies an approved revision onto its Announcement and
// publishes it. The first publish sets published_at; later edits keep it.
func publishRevision(ctx context.Context, repositories Repositories, announcement *Announcement, revision *Revision, now time.Time) error {
	content := revision.content()
	if err := repositories.Announcements.UpdateRevision(ctx, revision); err != nil {
		return err
	}
	announcement.Title, announcement.Body, announcement.Category = content.Title, content.Body, content.Category
	announcement.ExpiresAt, announcement.UpdatedAt = content.ExpiresAt, now
	announcement.Status = StatusPublished
	if announcement.PublishedAt == nil {
		announcement.PublishedAt = &now
	}
	if err := repositories.Announcements.UpdateAnnouncement(ctx, announcement); err != nil {
		return err
	}
	return repositories.Announcements.ReplaceAudience(ctx, announcement.ID, content.Audience)
}

func audit(ctx context.Context, repositories Repositories, actorID, action, announcementID string, metadata map[string]interface{}) error {
	return repositories.AuditLogs.Create(ctx, &auth.AuditLog{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "announcement",
		ResourceID:   &announcementID,
		Metadata:     metadata,
	})
}

// single builds the response for one Announcement after a change.
func (s *Service) single(ctx context.Context, announcement Announcement) (*AnnouncementResponse, error) {
	publisherName, err := s.repository.FullName(ctx, announcement.PublisherID)
	if err != nil {
		return nil, fmt.Errorf("load publisher: %w", err)
	}
	responses, err := s.toResponses(ctx, []FeedEntry{{Announcement: announcement, PublisherName: publisherName}})
	if err != nil {
		return nil, err
	}
	return &responses[0], nil
}

// queueResponses shows each pending revision as it would be published.
func (s *Service) queueResponses(ctx context.Context, entries []QueueEntry) ([]AnnouncementResponse, error) {
	departmentIDs := []string{}
	for _, entry := range entries {
		for _, rule := range entry.Audience {
			if rule.DepartmentID != nil {
				departmentIDs = append(departmentIDs, *rule.DepartmentID)
			}
		}
	}
	codes, err := s.repository.DepartmentCodes(ctx, departmentIDs)
	if err != nil {
		return nil, fmt.Errorf("load department codes: %w", err)
	}

	responses := make([]AnnouncementResponse, 0, len(entries))
	for _, entry := range entries {
		audience := make([]AudienceRuleResponse, 0, len(entry.Audience))
		for _, rule := range entry.Audience {
			view := AudienceRuleResponse{DepartmentID: rule.DepartmentID, BatchYear: rule.BatchYear, Role: rule.Role}
			if rule.DepartmentID != nil {
				if code, ok := codes[*rule.DepartmentID]; ok {
					view.DepartmentCode = &code
				}
			}
			audience = append(audience, view)
		}
		responses = append(responses, AnnouncementResponse{
			ID:            entry.AnnouncementID,
			Title:         entry.Title,
			Body:          entry.Body,
			Category:      entry.Category,
			Status:        StatusPending,
			PublisherID:   entry.SubmittedBy,
			PublisherName: entry.SubmitterName,
			Audience:      audience,
			ExpiresAt:     entry.ExpiresAt,
			CreatedAt:     entry.CreatedAt,
		})
	}
	return responses, nil
}

func pageLimit(limit int) int {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return limit
}
