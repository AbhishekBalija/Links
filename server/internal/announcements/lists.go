package announcements

// Lists: the approver's queue and the author's own Announcements.

import (
	"context"
	"fmt"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

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
	scope := approverScope(actorID, grants)
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
func (s *Service) Mine(ctx context.Context, actorID string, filter MineFilter, cursor string, limit int) ([]AnnouncementResponse, *FeedMeta, error) {
	if !validMineFilter(filter) {
		return nil, nil, apperrors.NewValidation("invalid status filter", map[string]string{"status": "use attention, draft, waiting, live or ended"})
	}
	limit = pageLimit(limit)
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.repository.Authored(ctx, actorID, filter, after, limit+1)
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

	if err := s.decorate(ctx, responses, true); err != nil {
		return nil, nil, err
	}
	return responses, meta, nil
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
		approver, nameErr := s.approverName(ctx, entry.ApproverDepartmentID)
		if nameErr != nil {
			return nil, nameErr
		}
		response := AnnouncementResponse{
			Kind:          "new",
			SubmittedAt:   entry.SubmittedAt,
			Approver:      &approver,
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
		}
		if entry.AnnouncementStatus == StatusPublished {
			response.Kind = "edit"
			response.Live = &LiveContent{Title: entry.LiveTitle, Body: entry.LiveBody}
		}
		responses = append(responses, response)
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
