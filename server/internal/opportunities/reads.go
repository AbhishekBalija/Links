package opportunities

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Feed lists published and closed Opportunities the reader is eligible for.
func (s *Service) Feed(ctx context.Context, actorID string, query FeedQuery) ([]OpportunityResponse, *ListMeta, error) {
	filter, err := s.feedFilter(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	limit, err := checkLimit(query.Limit)
	if err != nil {
		return nil, nil, err
	}
	after, err := decodeCursor(query.Cursor)
	if err != nil {
		return nil, nil, err
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, nil, err
	}
	views, err := s.repository.Feed(ctx, actorID, reader, filter, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list feed: %w", err)
	}
	meta := &ListMeta{}
	if len(views) > limit {
		views = views[:limit]
		last := views[len(views)-1]
		meta.NextCursor = encodeCursor(Cursor{At: last.ApplyBy, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, views, actorID)
	return responses, meta, err
}

func (s *Service) feedFilter(ctx context.Context, query FeedQuery) (FeedFilter, error) {
	var filter FeedFilter
	details := map[string]string{}
	switch state := State(strings.TrimSpace(query.State)); state {
	case "", StateOpen:
		filter.State = StateOpen
	case StateClosed, StateApplied:
		filter.State = state
	default:
		details["state"] = "use open, closed or applied"
	}
	if value := strings.TrimSpace(query.Type); value != "" {
		opportunityType := Type(value)
		if !validType(opportunityType) {
			details["type"] = "use job, internship or training"
		}
		filter.Type = &opportunityType
	}
	if code := strings.ToUpper(strings.TrimSpace(query.Department)); code != "" {
		id, err := s.repository.DepartmentIDByCode(ctx, code)
		if err != nil {
			return filter, fmt.Errorf("find department: %w", err)
		}
		if id == nil {
			details["department"] = "no department has this code"
		}
		filter.DepartmentID = id
	}
	if len(details) > 0 {
		return filter, apperrors.NewValidation("invalid opportunity filter", details)
	}
	return filter, nil
}

// Get returns one Opportunity. Placement staff see every one; a member sees
// a published or closed one they are eligible for or applied to. Anyone else
// gets not found, so drafts stay private.
func (s *Service) Get(ctx context.Context, actorID, id string) (*OpportunityResponse, error) {
	notFound := apperrors.NewNotFound("opportunity not found")
	if _, err := uuid.Parse(id); err != nil {
		return nil, notFound
	}
	staff, err := s.isStaff(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if staff {
		response, err := s.response(ctx, id, actorID)
		if err != nil {
			return nil, err
		}
		responses := []OpportunityResponse{*response}
		if err := s.addApplicantCounts(ctx, responses); err != nil {
			return nil, err
		}
		return &responses[0], nil
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, err
	}
	visible, err := s.repository.VisibleTo(ctx, actorID, reader, id)
	if err != nil {
		return nil, fmt.Errorf("check eligibility: %w", err)
	}
	if !visible {
		return nil, notFound
	}
	return s.response(ctx, id, actorID)
}

// reader describes a member. Each role is paired with the Department it
// belongs to: its scope for a Department-scoped role, otherwise the
// Department of the member's Student identity.
func (s *Service) reader(ctx context.Context, userID string) (Reader, error) {
	assignments, err := s.roles.GetRoleAssignments(ctx, userID)
	if err != nil {
		return Reader{}, fmt.Errorf("load roles: %w", err)
	}
	studentDepartment, batchYear, err := s.repository.StudentPlacement(ctx, userID)
	if err != nil {
		return Reader{}, fmt.Errorf("load student identity: %w", err)
	}
	reader := Reader{BatchYear: batchYear, Memberships: make([]Membership, 0, len(assignments))}
	for _, assignment := range assignments {
		membership := Membership{Role: string(assignment.Role), DepartmentID: studentDepartment}
		if assignment.ScopeType == auth.ScopeDepartment && assignment.ScopeID != nil {
			department := *assignment.ScopeID
			membership.DepartmentID = &department
		}
		reader.Memberships = append(reader.Memberships, membership)
	}
	return reader, nil
}
