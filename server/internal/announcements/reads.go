package announcements

import (
	"context"
	"fmt"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const principalOrAdmin = "Principal or admin"

// EditResponse is an edit to a published Announcement that hasn't gone live.
type EditResponse struct {
	Status     RevisionStatus `json:"status"`
	ReviewNote *string        `json:"review_note,omitempty"`
	Approver   *string        `json:"approver,omitempty"`
}

// PreviewInput asks how a category and Audience would be handled if posted.
type PreviewInput struct {
	Category string              `json:"category" binding:"required"`
	Audience []AudienceRuleInput `json:"audience"`
}

// PreviewResponse says whether posting would publish directly, and if not, who approves.
type PreviewResponse struct {
	PublishesDirectly bool    `json:"publishes_directly"`
	Approver          *string `json:"approver"`
}

// ApprovalSummary is what's waiting for an approver.
type ApprovalSummary struct {
	PendingCount      int        `json:"pending_count"`
	OldestSubmittedAt *time.Time `json:"oldest_submitted_at"`
}

// AuthorSummary counts an author's own Announcements that need their attention.
type AuthorSummary struct {
	Draft        int `json:"draft"`
	Pending      int `json:"pending"`
	Rejected     int `json:"rejected"`
	EditsWaiting int `json:"edits_waiting"`
}

// Get returns one Announcement. Its author can always read it; an approver can
// read one waiting for them; a reader can read a published one in their feed.
// Anyone else gets 404, so unpublished Announcements stay private.
func (s *Service) Get(ctx context.Context, actorID, id string) (*AnnouncementResponse, error) {
	entry, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find announcement: %w", err)
	}
	notFound := apperrors.NewNotFound("announcement not found")
	if entry == nil {
		return nil, notFound
	}
	if entry.PublisherID == actorID {
		return s.decorated(ctx, *entry, true)
	}

	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if entry.Status == StatusPending {
		revision, revisionErr := s.repository.OpenRevision(ctx, entry.ID)
		if revisionErr != nil {
			return nil, fmt.Errorf("load revision: %w", revisionErr)
		}
		if revision != nil && revision.Status == RevisionPending && canApprove(grants, revision) {
			return s.decorated(ctx, *entry, false)
		}
		return nil, notFound
	}
	if entry.Status != StatusPublished {
		return nil, notFound
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, err
	}
	visible, err := s.repository.VisibleTo(ctx, reader, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("check visibility: %w", err)
	}
	if visible == nil {
		return nil, notFound
	}
	return s.decorated(ctx, *entry, false)
}

// Preview says how posting this category and Audience would be handled for the
// caller, using the same rule as posting (ADR 0017).
func (s *Service) Preview(ctx context.Context, actorID string, input PreviewInput) (*PreviewResponse, error) {
	category := Category(input.Category)
	if !validCategory(category) {
		return nil, apperrors.NewValidation("invalid preview", map[string]string{"category": "use official, department or placement"})
	}
	audience, problem := toAudience(input.Audience)
	if problem != "" {
		return nil, apperrors.NewValidation("invalid preview", map[string]string{"audience": problem})
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !CanPost(grants, category) {
		return nil, apperrors.NewForbidden("you can't post this kind of announcement")
	}
	if err := checkDepartments(ctx, s.repository, audience); err != nil {
		return nil, err
	}

	decision := DecidePublishing(grants, category, audience)
	if decision.PublishDirectly {
		return &PreviewResponse{PublishesDirectly: true}, nil
	}
	approver, err := s.approverName(ctx, departmentPointer(decision.ApproverDepartmentID))
	if err != nil {
		return nil, err
	}
	return &PreviewResponse{Approver: &approver}, nil
}

// ApprovalSummary returns what's waiting for the user, or nil if they don't approve.
func (s *Service) ApprovalSummary(ctx context.Context, actorID string) (*ApprovalSummary, error) {
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	scope := approverScope(actorID, grants)
	if !scope.All && len(scope.DepartmentIDs) == 0 {
		return nil, nil
	}
	count, oldest, err := s.repository.QueueSummary(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("summarize queue: %w", err)
	}
	return &ApprovalSummary{PendingCount: count, OldestSubmittedAt: oldest}, nil
}

// AuthorSummary returns counts of the user's own Announcements, or nil if they can't post.
func (s *Service) AuthorSummary(ctx context.Context, actorID string) (*AuthorSummary, error) {
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !CanPost(grants, CategoryPlacement) && !CanPost(grants, CategoryOfficial) {
		return nil, nil
	}
	counts, editsWaiting, err := s.repository.AuthorCounts(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("count announcements: %w", err)
	}
	return &AuthorSummary{
		Draft:        counts[StatusDraft],
		Pending:      counts[StatusPending],
		Rejected:     counts[StatusRejected],
		EditsWaiting: editsWaiting,
	}, nil
}

// Grants exposes the user's current roles for modules that build on
// Announcements, such as the dashboard.
func (s *Service) Grants(ctx context.Context, userID string) ([]Grant, error) {
	return s.grants(ctx, userID)
}

func approverScope(actorID string, grants []Grant) ApproverScope {
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
	return scope
}

// decorated builds one response and adds workflow details: who approves a
// pending Announcement and, for its author only, a waiting or rejected edit
// and the rejection note.
func (s *Service) decorated(ctx context.Context, entry FeedEntry, forAuthor bool) (*AnnouncementResponse, error) {
	responses, err := s.toResponses(ctx, []FeedEntry{entry})
	if err != nil {
		return nil, err
	}
	if err := s.decorate(ctx, responses, forAuthor); err != nil {
		return nil, err
	}
	return &responses[0], nil
}

func (s *Service) decorate(ctx context.Context, responses []AnnouncementResponse, forAuthor bool) error {
	ids := make([]string, 0, len(responses))
	for _, response := range responses {
		ids = append(ids, response.ID)
	}
	revisions, err := s.repository.LatestRevisions(ctx, ids)
	if err != nil {
		return fmt.Errorf("load revisions: %w", err)
	}
	latest := map[string]Revision{}
	for _, revision := range revisions {
		latest[revision.AnnouncementID] = revision
	}

	for i := range responses {
		revision, ok := latest[responses[i].ID]
		if !ok {
			continue
		}
		switch responses[i].Status {
		case StatusPending:
			if revision.Status == RevisionPending {
				approver, nameErr := s.approverName(ctx, revision.ApproverDepartmentID)
				if nameErr != nil {
					return nameErr
				}
				responses[i].Approver = &approver
			}
		case StatusRejected:
			if forAuthor && revision.Status == RevisionRejected {
				responses[i].ReviewNote = revision.ReviewNote
			}
		case StatusPublished:
			if forAuthor && (revision.Status == RevisionPending || revision.Status == RevisionRejected) {
				edit := &EditResponse{Status: revision.Status, ReviewNote: revision.ReviewNote}
				if revision.Status == RevisionPending {
					approver, nameErr := s.approverName(ctx, revision.ApproverDepartmentID)
					if nameErr != nil {
						return nameErr
					}
					edit.Approver = &approver
				}
				responses[i].Edit = edit
				// Kept for older clients that read the note from the item itself.
				if revision.Status == RevisionRejected {
					responses[i].ReviewNote = revision.ReviewNote
				}
			}
		}
	}
	return nil
}

// approverName describes an approver for people: "CS HOD", or the principal
// or an admin when no Department is named.
func (s *Service) approverName(ctx context.Context, departmentID *string) (string, error) {
	if departmentID == nil {
		return principalOrAdmin, nil
	}
	codes, err := s.repository.DepartmentCodes(ctx, []string{*departmentID})
	if err != nil {
		return "", fmt.Errorf("load department code: %w", err)
	}
	code, ok := codes[*departmentID]
	if !ok {
		return principalOrAdmin, nil
	}
	return code + " HOD", nil
}
