package dashboard

import (
	"context"
	"sort"

	"github.com/AbhishekBalija/Links/server/internal/shared/authorwork"
)

// MyWorkSection is an author's own Announcements and Events that need them
// (a reviewer sent them back, with a note) or wait on someone (and since
// when). Each list is capped; its has_more says there are more.
type MyWorkSection struct {
	SentBack        []authorwork.SentBack `json:"sent_back"`
	SentBackHasMore bool                  `json:"sent_back_has_more"`
	Waiting         []authorwork.Waiting  `json:"waiting"`
	WaitingHasMore  bool                  `json:"waiting_has_more"`
}

// myWork merges the author's Announcements and Events. It is nil unless the
// user can post (canPost) or already has something sent back or waiting.
func (s *Service) myWork(ctx context.Context, userID string, canPost bool) (*MyWorkSection, error) {
	noticesBack, noticesWaiting, err := s.announcements.AuthorWork(ctx, userID, workOnHome+1)
	if err != nil {
		return nil, err
	}
	eventsBack, eventsWaiting, err := s.events.AuthorWork(ctx, userID, workOnHome+1)
	if err != nil {
		return nil, err
	}
	back := append(noticesBack, eventsBack...)
	waiting := append(noticesWaiting, eventsWaiting...)
	if !canPost && len(back) == 0 && len(waiting) == 0 {
		return nil, nil
	}
	// Newest first for what came back, longest waiting first for the rest.
	sort.SliceStable(back, func(i, j int) bool { return back[i].SentBackAt.After(back[j].SentBackAt) })
	sort.SliceStable(waiting, func(i, j int) bool { return waiting[i].Since.Before(waiting[j].Since) })
	section := &MyWorkSection{
		SentBack:        back,
		SentBackHasMore: len(back) > workOnHome,
		Waiting:         waiting,
		WaitingHasMore:  len(waiting) > workOnHome,
	}
	if section.SentBackHasMore {
		section.SentBack = back[:workOnHome]
	}
	if section.WaitingHasMore {
		section.Waiting = waiting[:workOnHome]
	}
	return section, nil
}
