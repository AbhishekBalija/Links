package auth

// The Not signed in list (#127): who a class list or a staff invite let in
// and who hasn't signed in yet, for the HOD, the principal and admins to
// remind. LINKS doesn't email them, so the list offers their emails.

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	notSignedInPage    = 50
	maxNotSignedInPage = 100
)

// NotSignedInQuery is the list's filters as the address bar holds them.
type NotSignedInQuery struct {
	Department string
	Kind       string
	Cursor     string
	Limit      int
}

// NotSignedInPage is one page of the list and how many match in all.
type NotSignedInPage struct {
	People     []WaitingPerson
	Total      int
	NextCursor string
}

// NotSignedIn lists, oldest first, the people waiting for a first sign-in in
// the actor's scope: anywhere for the principal and admins, the HOD's own
// Departments otherwise. Naming another Department is refused.
func (s *authService) NotSignedIn(ctx context.Context, actorID string, query NotSignedInQuery) (*NotSignedInPage, error) {
	filter, err := s.notSignedInFilter(ctx, actorID, query)
	if err != nil {
		return nil, err
	}
	filter.Limit = notSignedInPage
	if query.Limit > 0 {
		filter.Limit = min(query.Limit, maxNotSignedInPage)
	}
	if query.Cursor != "" {
		after, err := decodeNotSignedInCursor(query.Cursor)
		if err != nil {
			return nil, apperrors.NewValidation("invalid cursor", map[string]string{"cursor": "start again from the first page"})
		}
		filter.After = after
	}
	list, err := s.userRepo.NotSignedIn(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list people not signed in: %w", err)
	}
	page := &NotSignedInPage{People: list.People, Total: list.Total}
	if page.People == nil {
		page.People = []WaitingPerson{}
	}
	if len(page.People) == filter.Limit {
		last := page.People[len(page.People)-1]
		next := NotSignedInCursor{At: last.AddedAt, ID: last.UserID}
		// A full page may still be the last one; ask for one more to know.
		probe := filter
		probe.After, probe.Limit = &next, 1
		more, err := s.userRepo.NotSignedIn(ctx, probe)
		if err != nil {
			return nil, fmt.Errorf("look past the page: %w", err)
		}
		if len(more.People) > 0 {
			page.NextCursor = encodeNotSignedInCursor(next)
		}
	}
	return page, nil
}

// NotSignedInEmails is every matching email, for "Copy their emails".
func (s *authService) NotSignedInEmails(ctx context.Context, actorID string, query NotSignedInQuery) ([]string, error) {
	filter, err := s.notSignedInFilter(ctx, actorID, query)
	if err != nil {
		return nil, err
	}
	emails, err := s.userRepo.NotSignedInEmails(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list emails of people not signed in: %w", err)
	}
	if emails == nil {
		emails = []string{}
	}
	return emails, nil
}

func (s *authService) notSignedInFilter(ctx context.Context, actorID string, query NotSignedInQuery) (NotSignedInFilter, error) {
	switch query.Kind {
	case "", "student", "staff":
	default:
		return NotSignedInFilter{}, apperrors.NewValidation("invalid kind", map[string]string{"kind": "use student or staff"})
	}
	anywhere, scope, err := s.departmentScope(ctx, actorID, accessRefusal)
	if err != nil {
		return NotSignedInFilter{}, err
	}
	filter := NotSignedInFilter{Anywhere: anywhere, Kind: query.Kind}
	if code := strings.TrimSpace(query.Department); code != "" {
		department, err := s.userRepo.FindDepartmentByCode(ctx, strings.ToUpper(code))
		if err != nil {
			return NotSignedInFilter{}, fmt.Errorf("find department: %w", err)
		}
		if department == nil {
			return NotSignedInFilter{}, apperrors.NewValidation("unknown department", map[string]string{"department": "no department has the code " + code})
		}
		if !anywhere && !scope[department.ID] {
			return NotSignedInFilter{}, apperrors.NewForbidden("you can only see your own department's list")
		}
		filter.Anywhere, filter.DepartmentIDs = false, []string{department.ID}
		return filter, nil
	}
	for id := range scope {
		filter.DepartmentIDs = append(filter.DepartmentIDs, id)
	}
	sort.Strings(filter.DepartmentIDs)
	return filter, nil
}

func encodeNotSignedInCursor(c NotSignedInCursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

func decodeNotSignedInCursor(raw string) (*NotSignedInCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	at, id, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return nil, fmt.Errorf("cursor has no id")
	}
	when, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return nil, err
	}
	return &NotSignedInCursor{At: when, ID: id}, nil
}
