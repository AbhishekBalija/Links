package events

// Reading Events: the feed and one Event's detail.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// FeedQuery is the feed's query string, parsed in the service so each bad
// field is reported.
type FeedQuery struct {
	From       string
	To         string
	Department string
	EventType  string
	Show       string
	Cursor     string
	Limit      int
}

// Feed lists published Events in the reader's Audience, soonest first.
func (s *Service) Feed(ctx context.Context, actorID string, query FeedQuery) ([]EventResponse, *ListMeta, error) {
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
		return nil, nil, fmt.Errorf("load feed: %w", err)
	}
	meta := &ListMeta{}
	if len(views) > limit {
		views = views[:limit]
		last := views[len(views)-1]
		meta.NextCursor = encodeCursor(Cursor{At: last.StartsAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, views)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(responses))
	for _, response := range responses {
		ids = append(ids, response.ID)
	}
	counts, mine, err := s.repository.RSVPSummaries(ctx, ids, actorID)
	if err != nil {
		return nil, nil, fmt.Errorf("load answers: %w", err)
	}
	for i := range responses {
		responses[i].Reviews = []ReviewResponse{}
		id := responses[i].ID
		summary := &RSVPSummary{Counts: RSVPCounts{
			Going: counts[id][RSVPGoing], Interested: counts[id][RSVPInterested], NotGoing: counts[id][RSVPNotGoing],
		}}
		if status, ok := mine[id]; ok {
			summary.MyStatus = &status
		}
		responses[i].RSVP = summary
	}
	return responses, meta, nil
}

func (s *Service) feedFilter(ctx context.Context, query FeedQuery) (FeedFilter, error) {
	var filter FeedFilter
	details := map[string]string{}
	for field, value := range map[string]string{"from": query.From, "to": query.To} {
		if value == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			details[field] = "use an RFC 3339 time, like 2026-10-04T09:00:00Z"
			continue
		}
		if field == "from" {
			filter.From = &at
		} else {
			filter.To = &at
		}
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
	if value := strings.TrimSpace(query.EventType); value != "" {
		eventType := Type(value)
		if !validType(eventType) {
			details["event_type"] = "use talk, workshop, competition, cultural, sports, training or other"
		}
		filter.EventType = &eventType
	}
	switch show := Show(strings.TrimSpace(query.Show)); show {
	case "", ShowUpcoming:
		filter.Show = ShowUpcoming
	case ShowGoing, ShowPast:
		filter.Show = show
	default:
		details["show"] = "use upcoming, going or past"
	}
	if len(details) > 0 {
		return filter, apperrors.NewValidation("invalid event filter", details)
	}
	return filter, nil
}

// Get returns one Event to someone who may see it: its proposer (any status),
// a reviewer of its Department once it is submitted (with the review notes),
// or a reader in its Audience once it is published. Anyone else gets not
// found, so unpublished Events stay private.
func (s *Service) Get(ctx context.Context, actorID, id string) (*EventResponse, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, apperrors.NewNotFound("event not found")
	}
	view, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load event: %w", err)
	}
	if view == nil {
		return nil, apperrors.NewNotFound("event not found")
	}
	full, err := s.isOwnerOrReviewer(ctx, actorID, view.Event)
	if err != nil {
		return nil, err
	}
	if !full {
		reader, err := s.reader(ctx, actorID)
		if err != nil {
			return nil, err
		}
		visible, err := s.repository.VisibleTo(ctx, reader, id)
		if err != nil {
			return nil, fmt.Errorf("check audience: %w", err)
		}
		if !visible {
			return nil, apperrors.NewNotFound("event not found")
		}
	}
	responses, err := s.toResponses(ctx, []View{*view})
	if err != nil {
		return nil, err
	}
	if !full {
		responses[0].Reviews = []ReviewResponse{}
	}
	return &responses[0], nil
}

// isOwnerOrReviewer is true for the proposer and the Organiser, and for the principal, admins
// and the Department's HOD once the Event has left draft.
func (s *Service) isOwnerOrReviewer(ctx context.Context, actorID string, event Event) (bool, error) {
	if event.ProposerID == actorID || event.OrganiserID == actorID {
		return true, nil
	}
	if event.Status == StatusDraft {
		return false, nil
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return false, err
	}
	return privileged(grants) || hodOf(grants, event.DepartmentID), nil
}

// reader describes an Event reader. Each role is paired with the Department
// it belongs to: its scope for a Department-scoped role, otherwise the
// Department of the reader's Student identity.
func (s *Service) reader(ctx context.Context, userID string) (Reader, error) {
	grants, err := s.grants(ctx, userID)
	if err != nil {
		return Reader{}, err
	}
	studentDepartment, batchYear, err := s.repository.StudentPlacement(ctx, userID)
	if err != nil {
		return Reader{}, fmt.Errorf("load student identity: %w", err)
	}
	reader := Reader{BatchYear: batchYear, Memberships: make([]Membership, 0, len(grants))}
	for _, grant := range grants {
		membership := Membership{Role: string(grant.Role), DepartmentID: studentDepartment}
		if grant.DepartmentID != "" {
			department := grant.DepartmentID
			membership.DepartmentID = &department
		}
		reader.Memberships = append(reader.Memberships, membership)
	}
	return reader, nil
}
