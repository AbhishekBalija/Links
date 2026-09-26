package announcements

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/google/uuid"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
	maxAudienceRules = 20
	maxTitleLength   = 200
	maxBodyLength    = 10000
)

var audienceRoles = map[auth.Role]bool{
	auth.RoleStudent: true, auth.RoleStudentCoordinator: true, auth.RoleFaculty: true,
	auth.RoleHOD: true, auth.RolePlacementOfficer: true, auth.RolePrincipal: true,
	auth.RoleAlumni: true, auth.RoleClubOrganizer: true, auth.RoleAdmin: true,
}

type Service struct {
	repository Repository
	roles      RoleReader
	unitOfWork UnitOfWork
	now        func() time.Time
}

func NewService(repository Repository, roles RoleReader, unitOfWork UnitOfWork) *Service {
	return &Service{repository: repository, roles: roles, unitOfWork: unitOfWork, now: time.Now}
}

// Create posts an Announcement (ADR 0017). An author with Publishing authority
// over the Audience publishes immediately. Anyone else's Announcement waits for
// Announcement approval. Either can save a draft instead.
func (s *Service) Create(ctx context.Context, actorID string, input CreateAnnouncementInput) (*AnnouncementResponse, error) {
	content, err := s.validate(input.content())
	if err != nil {
		return nil, err
	}
	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !CanPost(grants, content.Category) {
		return nil, apperrors.NewForbidden("you can't post this kind of announcement")
	}
	decision := DecidePublishing(grants, content.Category, content.Audience)

	now := s.now().UTC()
	announcement := Announcement{
		Title:       content.Title,
		Body:        content.Body,
		Category:    content.Category,
		PublisherID: actorID,
		ExpiresAt:   content.ExpiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	switch {
	case input.Draft:
		announcement.Status = StatusDraft
	case decision.PublishDirectly:
		announcement.Status = StatusPublished
		announcement.PublishedAt = &now
	default:
		announcement.Status = StatusPending
	}

	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		if checkErr := checkDepartments(ctx, repositories.Announcements, content.Audience); checkErr != nil {
			return checkErr
		}
		// An unpublished Announcement keeps its working copy (including its
		// Audience rules) in step with its open revision.
		if createErr := repositories.Announcements.Create(ctx, &announcement, content.Audience); createErr != nil {
			return createErr
		}
		if announcement.Status == StatusPublished {
			return audit(ctx, repositories, actorID, "announcement.published", announcement.ID, nil)
		}

		revision := content.revision(announcement.ID, actorID, now)
		if announcement.Status == StatusPending {
			revision.submit(decision, now)
		}
		if revisionErr := repositories.Announcements.CreateRevision(ctx, &revision); revisionErr != nil {
			return revisionErr
		}
		if announcement.Status == StatusPending {
			return audit(ctx, repositories, actorID, "announcement.submitted", announcement.ID, nil)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return s.single(ctx, actorID, announcement)
}

// Feed returns one page of the Announcements whose Audience includes the reader.
func (s *Service) Feed(ctx context.Context, actorID, category, cursor string, limit int) ([]AnnouncementResponse, *FeedMeta, error) {
	if category != "" && !validCategory(Category(category)) {
		return nil, nil, apperrors.NewValidation("invalid category", map[string]string{"category": "use official, department or placement"})
	}
	limit = pageLimit(limit)
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	reader, err := s.reader(ctx, actorID)
	if err != nil {
		return nil, nil, err
	}

	// Ask for one extra row to know whether another page exists.
	entries, err := s.repository.Feed(ctx, reader, Category(category), after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("load feed: %w", err)
	}
	meta := &FeedMeta{}
	if len(entries) > limit {
		entries = entries[:limit]
		last := entries[len(entries)-1]
		meta.NextCursor = encodeCursor(FeedCursor{PublishedAt: *last.PublishedAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, entries)
	if err != nil {
		return nil, nil, err
	}
	return responses, meta, nil
}

func (s *Service) validate(input contentInput) (announcementContent, error) {
	details := map[string]string{}
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	if length := utf8.RuneCountInString(title); length < 3 || length > maxTitleLength {
		details["title"] = fmt.Sprintf("use 3 to %d characters", maxTitleLength)
	}
	if length := utf8.RuneCountInString(body); length < 1 || length > maxBodyLength {
		details["body"] = fmt.Sprintf("use 1 to %d characters", maxBodyLength)
	}
	category := Category(input.Category)
	if !validCategory(category) {
		details["category"] = "use official, department or placement"
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(s.now()) {
		details["expires_at"] = "must be in the future"
	}
	audience, audienceProblem := toAudience(input.Audience)
	if audienceProblem != "" {
		details["audience"] = audienceProblem
	}
	if len(details) > 0 {
		return announcementContent{}, apperrors.NewValidation("invalid announcement", details)
	}
	return announcementContent{Title: title, Body: body, Category: category, Audience: audience, ExpiresAt: input.ExpiresAt}, nil
}

// toAudience validates Audience rules. An empty list means the whole college.
func toAudience(inputs []AudienceRuleInput) ([]AudienceRule, string) {
	if len(inputs) > maxAudienceRules {
		return nil, fmt.Sprintf("use at most %d rules", maxAudienceRules)
	}
	rules := make([]AudienceRule, 0, len(inputs))
	for _, input := range inputs {
		if input.DepartmentID == nil && input.BatchYear == nil && input.Role == nil {
			return nil, "each rule needs a department, batch year or role; leave the audience empty for the whole college"
		}
		rule := AudienceRule{BatchYear: input.BatchYear}
		if input.DepartmentID != nil {
			if _, err := uuid.Parse(*input.DepartmentID); err != nil {
				return nil, "a department in the audience doesn't exist"
			}
			id := *input.DepartmentID
			rule.DepartmentID = &id
		}
		if input.BatchYear != nil && (*input.BatchYear < 2000 || *input.BatchYear > 2100) {
			return nil, "batch year must be between 2000 and 2100"
		}
		if input.Role != nil {
			role := auth.Role(*input.Role)
			if !audienceRoles[role] {
				return nil, fmt.Sprintf("%q is not a LINKS role", *input.Role)
			}
			rule.Role = &role
		}
		rules = append(rules, rule)
	}
	return rules, ""
}

// checkDepartments makes sure every Department in the Audience exists, holding
// a share lock on them for the rest of the transaction.
func checkDepartments(ctx context.Context, repository Repository, audience []AudienceRule) error {
	seen := map[string]bool{}
	ids := []string{}
	for _, rule := range audience {
		if rule.DepartmentID != nil && !seen[*rule.DepartmentID] {
			seen[*rule.DepartmentID] = true
			ids = append(ids, *rule.DepartmentID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	found, err := repository.LockDepartments(ctx, ids)
	if err != nil {
		return fmt.Errorf("check departments: %w", err)
	}
	if found != len(ids) {
		return apperrors.NewValidation("invalid announcement", map[string]string{"audience": "a department in the audience doesn't exist"})
	}
	return nil
}

// grants loads the roles the user holds right now from the database, not the
// token, so Department scopes are known and revoked roles stop counting.
func (s *Service) grants(ctx context.Context, userID string) ([]Grant, error) {
	assignments, err := s.roles.GetRoleAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}
	grants := make([]Grant, 0, len(assignments))
	for _, assignment := range assignments {
		grant := Grant{Role: assignment.Role}
		if assignment.ScopeType == auth.ScopeDepartment && assignment.ScopeID != nil {
			grant.DepartmentID = *assignment.ScopeID
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

// reader describes the feed reader. Each role is paired with the Department
// it belongs to: its scope for a Department-scoped role, otherwise the
// Department of the reader's Student identity. Batch year comes from the
// Student identity.
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

func (s *Service) toResponses(ctx context.Context, entries []FeedEntry) ([]AnnouncementResponse, error) {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	rules, err := s.repository.AudienceRules(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load audiences: %w", err)
	}
	byAnnouncement := map[string][]AudienceRuleResponse{}
	for _, rule := range rules {
		byAnnouncement[rule.TargetID] = append(byAnnouncement[rule.TargetID], AudienceRuleResponse{
			DepartmentID:   rule.DepartmentID,
			DepartmentCode: rule.DepartmentCode,
			BatchYear:      rule.BatchYear,
			Role:           rule.Role,
		})
	}

	responses := make([]AnnouncementResponse, 0, len(entries))
	for _, entry := range entries {
		audience := byAnnouncement[entry.ID]
		if audience == nil {
			audience = []AudienceRuleResponse{}
		}
		responses = append(responses, AnnouncementResponse{
			ID:            entry.ID,
			Title:         entry.Title,
			Body:          entry.Body,
			Category:      entry.Category,
			Status:        entry.Status,
			PublisherID:   entry.PublisherID,
			PublisherName: entry.PublisherName,
			Audience:      audience,
			PublishedAt:   entry.PublishedAt,
			ExpiresAt:     entry.ExpiresAt,
			CreatedAt:     entry.CreatedAt,
		})
	}
	return responses, nil
}

func encodeCursor(cursor FeedCursor) string {
	raw := cursor.PublishedAt.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(value string) (*FeedCursor, error) {
	if value == "" {
		return nil, nil
	}
	invalid := apperrors.NewValidation("invalid cursor", map[string]string{"cursor": "use the next_cursor from the previous page"})
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, invalid
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, invalid
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, invalid
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return nil, invalid
	}
	return &FeedCursor{PublishedAt: publishedAt, ID: parts[1]}, nil
}

func validCategory(category Category) bool {
	return category == CategoryOfficial || category == CategoryDepartment || category == CategoryPlacement
}
