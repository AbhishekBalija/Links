package events

// RSVPs: readers answer a published Event; organisers see who answered.

import (
	"context"
	"fmt"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// RSVP records the caller's answer to a published Event that hasn't started.
// The Event row is locked, so two people can't both take the last seat.
func (s *Service) RSVP(ctx context.Context, actorID, id string, input RSVPInput) (*RSVPSummary, error) {
	status := RSVPStatus(input.Status)
	if status != RSVPGoing && status != RSVPInterested && status != RSVPNotGoing {
		return nil, apperrors.NewValidation("invalid rsvp", map[string]string{"status": "use going, interested or not_going"})
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event == nil {
			return apperrors.NewNotFound("event not found")
		}
		visible, err := repositories.Events.VisibleTo(ctx, reader, event.ID)
		if err != nil {
			return fmt.Errorf("check audience: %w", err)
		}
		if !visible {
			return apperrors.NewNotFound("event not found")
		}
		if event.Status == StatusCancelled {
			return apperrors.NewConflict("the event was cancelled")
		}
		if !event.StartsAt.After(now) {
			return apperrors.NewConflict("the event has already started")
		}

		existing, err := repositories.Events.FindRSVP(ctx, event.ID, actorID)
		if err != nil {
			return fmt.Errorf("find rsvp: %w", err)
		}
		alreadyGoing := existing != nil && existing.Status == RSVPGoing
		if status == RSVPGoing && !alreadyGoing && event.Capacity != nil {
			counts, err := repositories.Events.RSVPCounts(ctx, event.ID)
			if err != nil {
				return fmt.Errorf("count rsvps: %w", err)
			}
			if counts[RSVPGoing] >= *event.Capacity {
				return apperrors.NewConflict("the event is full")
			}
		}
		if existing == nil {
			existing = &RSVP{EventID: event.ID, UserID: actorID, CreatedAt: now}
		}
		existing.Status = status
		existing.UpdatedAt = now
		if err := repositories.Events.SaveRSVP(ctx, existing); err != nil {
			return fmt.Errorf("save rsvp: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	page, err := s.summary(ctx, actorID, id, false, "", 0)
	if err != nil {
		return nil, err
	}
	return &page.RSVPSummary, nil
}

// RSVPs returns an Event's answer counts to anyone who can see it, and the
// people who answered to its organisers: the proposer, the Department's HOD,
// the principal and admins.
func (s *Service) RSVPs(ctx context.Context, actorID, id, cursor string, limit int) (*RSVPSummary, *ListMeta, error) {
	organiser, err := s.access(ctx, actorID, id)
	if err != nil {
		return nil, nil, err
	}
	summary, err := s.summary(ctx, actorID, id, organiser, cursor, limit)
	if err != nil {
		return nil, nil, err
	}
	meta := &ListMeta{}
	if organiser {
		meta.NextCursor = summary.nextCursor
	}
	return &summary.RSVPSummary, meta, nil
}

type summaryPage struct {
	RSVPSummary
	nextCursor string
}

func (s *Service) summary(ctx context.Context, actorID, id string, withPeople bool, cursor string, limit int) (*summaryPage, error) {
	counts, err := s.repository.RSVPCounts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("count rsvps: %w", err)
	}
	page := &summaryPage{RSVPSummary: RSVPSummary{Counts: RSVPCounts{
		Going: counts[RSVPGoing], Interested: counts[RSVPInterested], NotGoing: counts[RSVPNotGoing],
	}}}
	mine, err := s.repository.FindRSVP(ctx, id, actorID)
	if err != nil {
		return nil, fmt.Errorf("find rsvp: %w", err)
	}
	if mine != nil {
		page.MyStatus = &mine.Status
	}
	if !withPeople {
		return page, nil
	}

	limit, err = checkLimit(limit)
	if err != nil {
		return nil, err
	}
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, err
	}
	people, err := s.repository.RSVPPeople(ctx, id, after, limit+1)
	if err != nil {
		return nil, fmt.Errorf("list rsvps: %w", err)
	}
	if len(people) > limit {
		people = people[:limit]
		last := people[len(people)-1]
		page.nextCursor = encodeCursor(Cursor{At: last.UpdatedAt, ID: last.UserID})
	}
	page.People = make([]RSVPPersonResponse, 0, len(people))
	for _, person := range people {
		page.People = append(page.People, RSVPPersonResponse{
			UserID: person.UserID, FullName: person.FullName, Username: person.Username, Status: person.Status, RespondedAt: person.UpdatedAt,
		})
	}
	return page, nil
}

// access returns whether the caller organises the Event (proposer or
// reviewer), or not found when they can't see it at all.
func (s *Service) access(ctx context.Context, actorID, id string) (bool, error) {
	view, err := s.repository.Find(ctx, id)
	if err != nil {
		return false, fmt.Errorf("load event: %w", err)
	}
	if view == nil {
		return false, apperrors.NewNotFound("event not found")
	}
	organiser, err := s.isOwnerOrReviewer(ctx, actorID, view.Event)
	if err != nil || organiser {
		return organiser, err
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return false, err
	}
	visible, err := s.repository.VisibleTo(ctx, reader, id)
	if err != nil {
		return false, fmt.Errorf("check audience: %w", err)
	}
	if !visible {
		return false, apperrors.NewNotFound("event not found")
	}
	return false, nil
}
