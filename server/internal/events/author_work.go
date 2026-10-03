package events

import (
	"context"
	"fmt"

	"github.com/AbhishekBalija/Links/server/internal/shared/authorwork"
)

// AuthorWork lists the proposer's Events a reviewer sent back for changes
// (newest first) and the ones waiting for a review (longest waiting first),
// up to limit of each.
func (s *Service) AuthorWork(ctx context.Context, actorID string, limit int) ([]authorwork.SentBack, []authorwork.Waiting, error) {
	back, err := s.repository.SentBack(ctx, actorID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("load sent back events: %w", err)
	}
	waiting, err := s.repository.Waiting(ctx, actorID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("load waiting events: %w", err)
	}
	sentBack := make([]authorwork.SentBack, 0, len(back))
	for _, row := range back {
		sentBack = append(sentBack, authorwork.SentBack{
			Kind: authorwork.KindEvent, ID: row.ID, Title: row.Title, Note: row.Note,
			SentBackBy: row.ReviewerName, SentBackAt: row.DecidedAt,
		})
	}
	items := make([]authorwork.Waiting, 0, len(waiting))
	for _, row := range waiting {
		items = append(items, authorwork.Waiting{
			Kind: authorwork.KindEvent, ID: row.ID, Title: row.Title, WaitingOn: reviewer(row), Since: row.Since,
		})
	}
	return sentBack, items, nil
}

// reviewer says who an Event waits on: its Department's HOD at the first
// stage, otherwise the principal or an admin.
func reviewer(row WaitingRow) string {
	if row.Status == StatusSubmitted && row.DepartmentHasHOD && row.DepartmentCode != nil {
		return *row.DepartmentCode + " HOD"
	}
	return authorwork.PrincipalOrAdmin
}
