package integration

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/AbhishekBalija/Links/server/test/apitest"
)

type feedAnswer struct {
	Title string `json:"title"`
	RSVP  *struct {
		Counts struct {
			Going      int `json:"going"`
			Interested int `json:"interested"`
			NotGoing   int `json:"not_going"`
		} `json:"counts"`
		MyStatus *string `json:"my_status"`
	} `json:"rsvp"`
}

func feedAnswers(t *testing.T, h *apitest.Harness, token string, query url.Values) map[string]feedAnswer {
	t.Helper()
	response := h.Do(t, http.MethodGet, "/api/v1/events?"+query.Encode(), token, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("feed status = %d: %s", response.Status, response.Body)
	}
	var page struct {
		Data []feedAnswer `json:"data"`
	}
	response.Decode(t, &page)
	byTitle := map[string]feedAnswer{}
	for _, item := range page.Data {
		byTitle[item.Title] = item
	}
	return byTitle
}

func TestFeedItemsCarryAnswerCountsAndTheReadersAnswer(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	other := student(t, h, "CS", 2023)
	talk := publishedEvent(t, h, principal, "Guest talk", 2, nil)
	publishedEvent(t, h, principal, "Quiet event", 3, nil)
	rsvp(t, h, reader.Token, talk, "going")
	rsvp(t, h, other.Token, talk, "interested")

	items := feedAnswers(t, h, reader.Token, url.Values{})
	got := items["Guest talk"].RSVP
	if got == nil || got.Counts.Going != 1 || got.Counts.Interested != 1 || got.MyStatus == nil || *got.MyStatus != "going" {
		t.Fatalf("Guest talk rsvp = %+v, want 1 going, 1 interested and my answer going", got)
	}
	quiet := items["Quiet event"].RSVP
	if quiet == nil || quiet.Counts.Going != 0 || quiet.MyStatus != nil {
		t.Errorf("Quiet event rsvp = %+v, want zero counts and no answer", quiet)
	}
}

func TestCancelledEventsStayForThoseWhoAnsweredUntilTheyEnd(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	answered := student(t, h, "CS", 2023)
	silent := student(t, h, "CS", 2023)
	open := publishedEvent(t, h, principal, "Open day", 2, nil)
	over := publishedEvent(t, h, principal, "Old meetup", 4, nil)
	rsvp(t, h, answered.Token, open, "interested")
	rsvp(t, h, answered.Token, over, "going")
	for _, id := range []string{open, over} {
		if response := cancelEvent(t, h, principal.Token, id, "Speaker unwell"); response.Status != http.StatusOK {
			t.Fatalf("cancel status = %d: %s", response.Status, response.Body)
		}
	}
	// The second one has since ended.
	if err := h.DB().Exec(`UPDATE events SET starts_at = now() - interval '3 hours', ends_at = now() - interval '1 hour' WHERE id = ?`, over).Error; err != nil {
		t.Fatalf("end event: %v", err)
	}

	if titles := eventTitles(eventFeedItems(t, h, answered.Token, url.Values{})); !sameTitles(titles, []string{"Open day"}) {
		t.Errorf("feed for someone who answered = %v, want the cancelled Open day only", titles)
	}
	if titles := eventTitles(eventFeedItems(t, h, silent.Token, url.Values{})); len(titles) != 0 {
		t.Errorf("feed for someone who never answered = %v, want nothing", titles)
	}
}

func TestFeedShowsGoingAndPastViews(t *testing.T) {
	h := apitest.New(t)
	principal := h.SeedUser(t, apitest.UserSeed{Roles: []apitest.RoleSeed{{Role: "principal"}}})
	reader := student(t, h, "CS", 2023)
	going := publishedEvent(t, h, principal, "Hackathon", 5, nil)
	interested := publishedEvent(t, h, principal, "Quiz", 2, nil)
	publishedEvent(t, h, principal, "Fest", 9, nil)
	older := publishedEvent(t, h, principal, "Orientation", 6, nil)
	newer := publishedEvent(t, h, principal, "Workshop", 7, nil)
	rsvp(t, h, reader.Token, going, "going")
	rsvp(t, h, reader.Token, interested, "interested")
	for id, daysAgo := range map[string]int{older: 10, newer: 3} {
		if err := h.DB().Exec(`UPDATE events SET starts_at = now() - make_interval(days => ?), ends_at = now() - make_interval(days => ?) + interval '2 hours' WHERE id = ?`, daysAgo, daysAgo, id).Error; err != nil {
			t.Fatalf("move event into the past: %v", err)
		}
	}

	if titles := eventTitles(eventFeedItems(t, h, reader.Token, url.Values{"show": {"going"}})); !sameTitles(titles, []string{"Hackathon"}) {
		t.Errorf("going = %v, want only the event answered going", titles)
	}
	if titles := eventTitles(eventFeedItems(t, h, reader.Token, url.Values{"show": {"past"}})); !sameTitles(titles, []string{"Workshop", "Orientation"}) {
		t.Errorf("past = %v, want ended events, most recent first", titles)
	}
	if titles := eventTitles(eventFeedItems(t, h, reader.Token, url.Values{})); !sameTitles(titles, []string{"Quiz", "Hackathon", "Fest"}) {
		t.Errorf("upcoming = %v, want the three still to come, soonest first", titles)
	}

	// Past pages run backwards in time without repeats.
	first, cursor := eventFeed(t, h, reader.Token, url.Values{"show": {"past"}, "limit": {"1"}})
	second, _ := eventFeed(t, h, reader.Token, url.Values{"show": {"past"}, "limit": {"1"}, "cursor": {cursor}})
	if !sameTitles(append(eventTitles(first), eventTitles(second)...), []string{"Workshop", "Orientation"}) {
		t.Errorf("past pages = %v then %v", eventTitles(first), eventTitles(second))
	}

	if bad := h.Do(t, http.MethodGet, "/api/v1/events?show=someday", reader.Token, nil); bad.Status != http.StatusBadRequest {
		t.Errorf("unknown show = %d, want %d", bad.Status, http.StatusBadRequest)
	}
}

func eventFeedItems(t *testing.T, h *apitest.Harness, token string, query url.Values) []eventItem {
	t.Helper()
	items, _ := eventFeed(t, h, token, query)
	return items
}
