package opportunities

import (
	"context"
	"fmt"
	"time"
)

// drivesOnHome is how many open drives the placement summary lists.
const drivesOnHome = 5

// PlacementSummary is the placement office's state for Home: the open
// drives, soonest deadline first, and the Applications nobody has reviewed.
type PlacementSummary struct {
	OpenCount           int     `json:"open_count"`
	AwaitingReviewCount int     `json:"awaiting_review_count"`
	Drives              []Drive `json:"drives"`
}

// Drive is one open Opportunity on Home with its applicant counts.
type Drive struct {
	ID              string          `json:"id"`
	OpportunityType Type            `json:"opportunity_type"`
	Title           string          `json:"title"`
	Company         string          `json:"company"`
	ApplyBy         time.Time       `json:"apply_by"`
	ApplicantCounts ApplicantCounts `json:"applicant_counts"`
}

// PlacementSummary returns the summary for placement staff, or nil for
// anyone else.
func (s *Service) PlacementSummary(ctx context.Context, actorID string) (*PlacementSummary, error) {
	staff, err := s.isStaff(ctx, actorID)
	if err != nil || !staff {
		return nil, err
	}
	views, openCount, err := s.repository.OpenDrives(ctx, drivesOnHome)
	if err != nil {
		return nil, fmt.Errorf("list open drives: %w", err)
	}
	awaiting, err := s.repository.AwaitingReviewCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("count applications awaiting review: %w", err)
	}
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
	}
	counts, err := s.repository.ApplicantCounts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("count applicants: %w", err)
	}
	summary := &PlacementSummary{OpenCount: openCount, AwaitingReviewCount: awaiting, Drives: make([]Drive, 0, len(views))}
	for _, view := range views {
		summary.Drives = append(summary.Drives, Drive{
			ID:              view.ID,
			OpportunityType: view.OpportunityType,
			Title:           view.Title,
			Company:         view.Company,
			ApplyBy:         view.ApplyBy,
			ApplicantCounts: counts[view.ID],
		})
	}
	return summary, nil
}
