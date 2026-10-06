package directory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/profiles"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	defaultLimit = 20
	maxLimit     = 50
	minBatchYear = 2000
	maxBatchYear = 2100
	// A one-letter q matches nearly everyone, so it is ignored.
	minQueryLength = 2
	maxQueryLength = 100
	searchLimit    = 50
)

// roleOrder is how roles are listed on an entry: most senior first.
var roleOrder = []auth.Role{
	auth.RoleAdmin, auth.RolePrincipal, auth.RoleHOD, auth.RolePlacementOfficer, auth.RoleFaculty,
	auth.RoleStudentCoordinator, auth.RoleClubOrganizer, auth.RoleStudent, auth.RoleAlumni,
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns one page of the directory, alphabetical by name, or with a
// search term the best matches first.
func (s *Service) List(ctx context.Context, viewerID string, query ListQuery) ([]Entry, *ListMeta, error) {
	filter, err := s.parseFilter(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	limit, err := parseLimit(query.Limit)
	if err != nil {
		return nil, nil, err
	}
	q, err := parseQuery(query.Q)
	if err != nil {
		return nil, nil, err
	}
	if q != "" {
		return s.search(ctx, viewerID, filter, q, query.Cursor)
	}
	after, err := decodeCursor(query.Cursor)
	if err != nil {
		return nil, nil, err
	}

	members, err := s.repo.List(ctx, filter, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list members: %w", err)
	}
	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("count members: %w", err)
	}
	meta := &ListMeta{Total: total}
	if len(members) > limit {
		members = members[:limit]
		last := members[len(members)-1]
		meta.NextCursor = encodeCursor(Cursor{SortName: last.SortName, UserID: last.UserID})
	}
	entries, err := s.entries(ctx, viewerID, members)
	return entries, meta, err
}

// search returns the top matches in one response: ranked results have no
// stable order to page through.
func (s *Service) search(ctx context.Context, viewerID string, filter Filter, q, cursor string) ([]Entry, *ListMeta, error) {
	if cursor != "" {
		return nil, nil, apperrors.NewValidation("invalid cursor", map[string]string{"cursor": "search results come in one page; leave cursor out with q"})
	}
	members, err := s.repo.Search(ctx, filter, q, searchLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("search members: %w", err)
	}
	entries, err := s.entries(ctx, viewerID, members)
	return entries, &ListMeta{Total: len(members)}, err
}

// parseQuery lower-cases and trims q, and drops one too short to mean anything.
func parseQuery(value string) (string, error) {
	q := strings.ToLower(strings.Join(strings.Fields(value), " "))
	length := len([]rune(q))
	if length > maxQueryLength {
		return "", apperrors.NewValidation("invalid search", map[string]string{"q": fmt.Sprintf("at most %d characters", maxQueryLength)})
	}
	if length < minQueryLength {
		return "", nil
	}
	return q, nil
}

func (s *Service) parseFilter(ctx context.Context, query ListQuery) (Filter, error) {
	var filter Filter
	problems := map[string]string{}
	if code := strings.ToUpper(strings.TrimSpace(query.Department)); code != "" {
		department, err := s.repo.DepartmentByCode(ctx, code)
		if err != nil {
			return filter, fmt.Errorf("find department: %w", err)
		}
		if department == nil {
			problems["department"] = "no department has this code"
		} else {
			filter.DepartmentID = &department.ID
		}
	}
	if value := strings.TrimSpace(query.Role); value != "" {
		role := auth.Role(value)
		if !knownRole(role) {
			problems["role"] = "not a LINKS role"
		} else {
			filter.Role = &role
		}
	}
	if value := strings.TrimSpace(query.Batch); value != "" {
		year, err := strconv.Atoi(value)
		if err != nil || year < minBatchYear || year > maxBatchYear {
			problems["batch"] = fmt.Sprintf("must be a year from %d to %d", minBatchYear, maxBatchYear)
		} else {
			filter.BatchYear = &year
		}
	}
	if len(problems) > 0 {
		return filter, apperrors.NewValidation("invalid directory filter", problems)
	}
	return filter, nil
}

// entries turns members into what the viewer may see, applying the same
// contact rules as the public profile endpoint.
func (s *Service) entries(ctx context.Context, viewerID string, members []Member) ([]Entry, error) {
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	grants, err := s.repo.Grants(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	byUser := map[string][]Grant{}
	for _, grant := range grants {
		byUser[grant.UserID] = append(byUser[grant.UserID], grant)
	}

	entries := make([]Entry, 0, len(members))
	for _, member := range members {
		entries = append(entries, entryFor(member, byUser[member.UserID], viewerID))
	}
	return entries, nil
}

func entryFor(member Member, grants []Grant, viewerID string) Entry {
	roles, department, batchYear := membershipOf(member, grants)
	entry := Entry{
		Username:   member.Username,
		FullName:   member.FullName,
		Headline:   member.Headline,
		AvatarURL:  member.AvatarURL,
		Roles:      roles,
		Department: department,
		BatchYear:  batchYear,
	}

	privacy := profiles.Privacy{
		OwnerID:              member.UserID,
		PublicProfileEnabled: member.PublicProfileEnabled,
		ShowEmail:            member.ShowEmail,
		ShowPhone:            member.ShowPhone,
	}
	if privacy.ShowsEmailTo(&viewerID) {
		entry.Email = member.Email
	}
	if privacy.ShowsPhoneTo(&viewerID) {
		entry.Phone = member.Phone
	}
	return entry
}

// membershipOf is who a member is at the college: their roles in effect, most
// senior first, their Department and, for students, their Batch.
func membershipOf(member Member, grants []Grant) ([]string, *DepartmentRef, *int) {
	sort.SliceStable(grants, func(i, j int) bool { return rank(grants[i].Role) < rank(grants[j].Role) })
	roles := []string{}
	var department *DepartmentRef
	seen := map[auth.Role]bool{}
	for _, grant := range grants {
		if !seen[grant.Role] {
			seen[grant.Role] = true
			roles = append(roles, string(grant.Role))
		}
		// The most senior Department-scoped role names the Department.
		if department == nil && grant.DepartmentCode != nil {
			department = &DepartmentRef{Code: *grant.DepartmentCode, Name: stringValue(grant.DepartmentName)}
		}
	}
	if department == nil && member.StudentDepartmentCode != nil {
		department = &DepartmentRef{Code: *member.StudentDepartmentCode, Name: stringValue(member.StudentDepartmentName)}
	}
	var batchYear *int
	if seen[auth.RoleStudent] {
		batchYear = member.StudentBatchYear
	}
	return roles, department, batchYear
}

// Membership is the member's roles, Department and Batch for their profile
// page. It reads them whether or not the profile is listed; the profiles
// service decides who may see them.
func (s *Service) Membership(ctx context.Context, userID string) (*profiles.Membership, error) {
	member, err := s.repo.MemberByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find member: %w", err)
	}
	if member == nil {
		return &profiles.Membership{Roles: []string{}}, nil
	}
	grants, err := s.repo.Grants(ctx, []string{userID})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles, department, batchYear := membershipOf(*member, grants)
	membership := &profiles.Membership{Roles: roles, BatchYear: batchYear}
	if department != nil {
		membership.Department = &profiles.MembershipDepartment{Code: department.Code, Name: department.Name}
	}
	return membership, nil
}

func rank(role auth.Role) int {
	for i, candidate := range roleOrder {
		if candidate == role {
			return i
		}
	}
	return len(roleOrder)
}

func knownRole(role auth.Role) bool {
	return rank(role) < len(roleOrder)
}

func parseLimit(value string) (int, error) {
	if value == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, apperrors.NewValidation("invalid limit", map[string]string{"limit": fmt.Sprintf("must be from 1 to %d", maxLimit)})
	}
	return limit, nil
}

func encodeCursor(cursor Cursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(value string) (*Cursor, error) {
	if value == "" {
		return nil, nil
	}
	invalid := apperrors.NewValidation("invalid cursor", map[string]string{"cursor": "use the next_cursor from the previous page"})
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, invalid
	}
	var cursor Cursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return nil, invalid
	}
	if _, err := uuid.Parse(cursor.UserID); err != nil {
		return nil, invalid
	}
	return &cursor, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// Overview returns a Department's page. Counts include everyone on the
// lists, signed in or not yet, visible or not, since a number reveals no one; the HOD and staff list show
// only members the directory would list.
func (s *Service) Overview(ctx context.Context, viewerID, code string) (*Overview, error) {
	department, err := s.repo.DepartmentByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, fmt.Errorf("find department: %w", err)
	}
	if department == nil {
		return nil, apperrors.NewNotFound("department not found")
	}

	batches, err := s.repo.StudentsByBatch(ctx, department.ID)
	if err != nil {
		return nil, fmt.Errorf("count students: %w", err)
	}
	faculty, err := s.repo.FacultyCount(ctx, department.ID)
	if err != nil {
		return nil, fmt.Errorf("count faculty: %w", err)
	}
	staffCount, err := s.repo.StaffCount(ctx, department.ID)
	if err != nil {
		return nil, fmt.Errorf("count staff: %w", err)
	}
	overview := &Overview{
		Department: OverviewDepartment{Code: department.Code, Name: department.Name, Description: department.Description},
		Counts:     OverviewCounts{Faculty: faculty, Staff: staffCount, StudentsByBatch: batches},
		Staff:      []Entry{},
	}
	if overview.Counts.StudentsByBatch == nil {
		overview.Counts.StudentsByBatch = []BatchCount{}
	}
	for _, batch := range batches {
		overview.Counts.Students += batch.Count
	}

	holder, err := s.repo.HODHolder(ctx, department.ID)
	if err != nil {
		return nil, fmt.Errorf("find HOD: %w", err)
	}
	overview.Holder, overview.HasHOD = holder, holder != nil

	staff, err := s.repo.Staff(ctx, department.ID)
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	ids := make([]string, 0, len(staff))
	for _, member := range staff {
		ids = append(ids, member.UserID)
	}
	grants, err := s.repo.Grants(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	byUser := map[string][]Grant{}
	for _, grant := range grants {
		byUser[grant.UserID] = append(byUser[grant.UserID], grant)
	}

	// Staff are ordered by their most senior role in this Department, then
	// by name (the repository's order, kept by the stable sort).
	ranks := map[string]int{}
	for _, member := range staff {
		ranks[member.UserID] = len(roleOrder)
		for _, grant := range byUser[member.UserID] {
			if grant.DepartmentCode != nil && *grant.DepartmentCode == department.Code && rank(grant.Role) < ranks[member.UserID] {
				ranks[member.UserID] = rank(grant.Role)
			}
		}
	}
	sort.SliceStable(staff, func(i, j int) bool { return ranks[staff[i].UserID] < ranks[staff[j].UserID] })
	for _, member := range staff {
		entry := entryFor(member, byUser[member.UserID], viewerID)
		overview.Staff = append(overview.Staff, entry)
		if overview.HOD == nil && ranks[member.UserID] == rank(auth.RoleHOD) {
			hod := entry
			overview.HOD = &hod
		}
	}
	return overview, nil
}
