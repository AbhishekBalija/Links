package events

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	maxTitleLength       = 200
	maxDescriptionLength = 5000
	maxLocationLength    = 200
	maxCapacity          = 100000
	maxAudienceRules     = 20
	defaultLimit         = 20
	maxLimit             = 50
)

var audienceRoles = map[auth.Role]bool{
	auth.RoleStudent: true, auth.RoleStudentCoordinator: true, auth.RoleFaculty: true, auth.RoleHOD: true,
	auth.RolePlacementOfficer: true, auth.RolePrincipal: true, auth.RoleAlumni: true,
	auth.RoleClubOrganizer: true, auth.RoleAdmin: true,
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

// content is the editable part of an Event, validated.
type content struct {
	Title           string
	Description     string
	EventType       Type
	DepartmentID    *string
	FacultyMentorID *string
	Location        string
	StartsAt        time.Time
	EndsAt          time.Time
	Capacity        *int
	Audience        []AudienceRule
}

// Create saves a new Event as a draft, or submits it for review straight away
// unless Draft is set.
func (s *Service) Create(ctx context.Context, actorID string, input CreateEventInput) (*EventResponse, error) {
	details := map[string]string{}
	if input.StartsAt == nil {
		details["starts_at"] = "required"
	}
	if input.EndsAt == nil {
		details["ends_at"] = "required"
	}
	if len(details) > 0 {
		return nil, apperrors.NewValidation("invalid event", details)
	}
	proposed := content{
		Title:           input.Title,
		Description:     input.Description,
		EventType:       Type(input.EventType),
		DepartmentID:    blankToNil(input.DepartmentID),
		FacultyMentorID: blankToNil(input.FacultyMentorID),
		Location:        input.Location,
		StartsAt:        *input.StartsAt,
		EndsAt:          *input.EndsAt,
		Capacity:        input.Capacity,
	}
	proposed, err := s.check(ctx, actorID, proposed, input.Audience)
	if err != nil {
		return nil, err
	}

	now := s.now()
	event := Event{
		ProposerID:  actorID,
		OrganiserID: actorID,
		Status:      StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	apply(&event, proposed)
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		if err := checkReferences(ctx, repositories.Events, proposed); err != nil {
			return err
		}
		if err := repositories.Events.Create(ctx, &event, proposed.Audience); err != nil {
			return fmt.Errorf("create event: %w", err)
		}
		if err := audit(ctx, repositories, actorID, "event_created", event.ID, map[string]string{"status": string(event.Status)}, now); err != nil {
			return err
		}
		if input.Draft {
			return nil
		}
		return s.submitLocked(ctx, repositories, actorID, &event, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, event.ID)
}

// Update edits an Event the caller proposed that is still a draft or was
// sent back for changes. Someone else's Event is not found, so drafts stay
// private. A published Event takes logistics edits from its organisers.
func (s *Service) Update(ctx context.Context, actorID, id string, input UpdateEventInput) (*EventResponse, error) {
	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		event, err := repositories.Events.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find event: %w", err)
		}
		if event != nil && event.Status == StatusPublished {
			return s.editLogistics(ctx, repositories, actorID, event, input, now)
		}
		if event == nil || event.ProposerID != actorID {
			return apperrors.NewNotFound("event not found")
		}
		if !editable(event.Status) {
			return apperrors.NewConflict("only a draft or an event sent back for changes can be edited")
		}
		audience, err := repositories.Events.Audience(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("load audience: %w", err)
		}

		changed, audienceInput := merge(*event, audience, input)
		checked, err := s.check(ctx, actorID, changed, audienceInput)
		if err != nil {
			return err
		}
		if err := checkReferences(ctx, repositories.Events, checked); err != nil {
			return err
		}
		apply(event, checked)
		event.UpdatedAt = now
		if err := repositories.Events.Update(ctx, event); err != nil {
			return fmt.Errorf("update event: %w", err)
		}
		if input.Audience != nil {
			if err := repositories.Events.ReplaceAudience(ctx, event.ID, checked.Audience); err != nil {
				return fmt.Errorf("replace audience: %w", err)
			}
		}
		return audit(ctx, repositories, actorID, "event_updated", event.ID, map[string]string{"status": string(event.Status)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id)
}

// Mine lists the caller's own Events in any status, newest first.
func (s *Service) Mine(ctx context.Context, actorID, filter, cursor string, limit int) ([]EventResponse, *ListMeta, error) {
	mineFilter := MineFilter(filter)
	if !validMineFilter(mineFilter) {
		return nil, nil, apperrors.NewValidation("invalid status", map[string]string{"status": "use draft, waiting, attention, live or ended"})
	}
	limit, err := checkLimit(limit)
	if err != nil {
		return nil, nil, err
	}
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	views, err := s.repository.Authored(ctx, actorID, mineFilter, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list events: %w", err)
	}
	meta := &ListMeta{}
	if len(views) > limit {
		views = views[:limit]
		last := views[len(views)-1]
		meta.NextCursor = encodeCursor(Cursor{At: last.CreatedAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, views)
	return responses, meta, err
}

func editable(status Status) bool {
	return status == StatusDraft || status == StatusHODChangesRequested || status == StatusFinalChangesRequested
}

// merge applies the fields a PATCH sent onto the Event's current content.
// It returns the Audience input to validate, or nil to keep the current one.
func merge(event Event, audience []AudienceRule, input UpdateEventInput) (content, []AudienceRuleInput) {
	changed := content{
		Title:           event.Title,
		Description:     event.Description,
		EventType:       event.EventType,
		DepartmentID:    event.DepartmentID,
		FacultyMentorID: event.FacultyMentorID,
		Location:        event.Location,
		StartsAt:        event.StartsAt,
		EndsAt:          event.EndsAt,
		Capacity:        event.Capacity,
		Audience:        audience,
	}
	if input.Title != nil {
		changed.Title = *input.Title
	}
	if input.Description != nil {
		changed.Description = *input.Description
	}
	if input.EventType != nil {
		changed.EventType = Type(*input.EventType)
	}
	if input.DepartmentID.Set {
		changed.DepartmentID = blankToNil(input.DepartmentID.Value)
	}
	if input.FacultyMentorID.Set {
		changed.FacultyMentorID = blankToNil(input.FacultyMentorID.Value)
	}
	if input.Location != nil {
		changed.Location = *input.Location
	}
	if input.StartsAt != nil {
		changed.StartsAt = *input.StartsAt
	}
	if input.EndsAt != nil {
		changed.EndsAt = *input.EndsAt
	}
	if input.Capacity.Set {
		changed.Capacity = input.Capacity.Value
	}
	if input.Audience == nil {
		return changed, nil
	}
	return changed, *input.Audience
}

// check validates the fields and whether the proposer may propose this Event.
// audienceInput replaces the Audience when not nil.
func (s *Service) check(ctx context.Context, actorID string, proposed content, audienceInput []AudienceRuleInput) (content, error) {
	details := map[string]string{}
	proposed.Title = strings.TrimSpace(proposed.Title)
	proposed.Description = strings.TrimSpace(proposed.Description)
	proposed.Location = strings.TrimSpace(proposed.Location)
	if length := utf8.RuneCountInString(proposed.Title); length < 3 || length > maxTitleLength {
		details["title"] = fmt.Sprintf("use 3 to %d characters", maxTitleLength)
	}
	if utf8.RuneCountInString(proposed.Description) > maxDescriptionLength {
		details["description"] = fmt.Sprintf("use at most %d characters", maxDescriptionLength)
	}
	if !validType(proposed.EventType) {
		details["event_type"] = "use talk, workshop, competition, cultural, sports, training or other"
	}
	if length := utf8.RuneCountInString(proposed.Location); length < 1 || length > maxLocationLength {
		details["location"] = fmt.Sprintf("use 1 to %d characters", maxLocationLength)
	}
	if !proposed.EndsAt.After(proposed.StartsAt) {
		details["ends_at"] = "must be after starts_at"
	}
	if proposed.Capacity != nil && (*proposed.Capacity < 1 || *proposed.Capacity > maxCapacity) {
		details["capacity"] = fmt.Sprintf("use a whole number from 1 to %d, or leave it out for no limit", maxCapacity)
	}
	if proposed.DepartmentID != nil {
		if _, err := uuid.Parse(*proposed.DepartmentID); err != nil {
			details["department_id"] = "no department has this ID"
		}
	}
	if proposed.FacultyMentorID != nil {
		if _, err := uuid.Parse(*proposed.FacultyMentorID); err != nil {
			details["faculty_mentor_id"] = "must be a faculty member"
		}
	}
	if audienceInput != nil {
		audience, problem := toAudience(audienceInput)
		if problem != "" {
			details["audience"] = problem
		}
		proposed.Audience = audience
	}
	if len(details) > 0 {
		return proposed, apperrors.NewValidation("invalid event", details)
	}

	grants, err := s.grants(ctx, actorID)
	if err != nil {
		return proposed, err
	}
	if field, problem := proposalProblem(grants, proposed.EventType, proposed.DepartmentID); problem != "" {
		return proposed, apperrors.NewValidation("you can't propose this event", map[string]string{field: problem})
	}
	return proposed, nil
}

// checkReferences makes sure the Department, the Audience's Departments and
// the faculty mentor exist, share-locking the Departments for the rest of the
// transaction.
func checkReferences(ctx context.Context, repository Repository, proposed content) error {
	if proposed.DepartmentID != nil {
		found, err := repository.LockDepartments(ctx, []string{*proposed.DepartmentID})
		if err != nil {
			return fmt.Errorf("check department: %w", err)
		}
		if found != 1 {
			return apperrors.NewValidation("invalid event", map[string]string{"department_id": "no department has this ID"})
		}
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, rule := range proposed.Audience {
		if rule.DepartmentID != nil && !seen[*rule.DepartmentID] {
			seen[*rule.DepartmentID] = true
			ids = append(ids, *rule.DepartmentID)
		}
	}
	if len(ids) > 0 {
		found, err := repository.LockDepartments(ctx, ids)
		if err != nil {
			return fmt.Errorf("check audience departments: %w", err)
		}
		if found != len(ids) {
			return apperrors.NewValidation("invalid event", map[string]string{"audience": "a department in the audience doesn't exist"})
		}
	}
	if proposed.FacultyMentorID != nil {
		faculty, err := repository.IsFaculty(ctx, *proposed.FacultyMentorID)
		if err != nil {
			return fmt.Errorf("check mentor: %w", err)
		}
		if !faculty {
			return apperrors.NewValidation("invalid event", map[string]string{"faculty_mentor_id": "must be a faculty member"})
		}
	}
	return nil
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

func apply(event *Event, proposed content) {
	event.Title = proposed.Title
	event.Description = proposed.Description
	event.EventType = proposed.EventType
	event.DepartmentID = proposed.DepartmentID
	event.FacultyMentorID = proposed.FacultyMentorID
	event.Location = proposed.Location
	event.StartsAt = proposed.StartsAt
	event.EndsAt = proposed.EndsAt
	event.Capacity = proposed.Capacity
}

// grants loads the roles the user holds right now from the database, not the
// token, so Department scopes are known and ended roles stop counting.
func (s *Service) grants(ctx context.Context, userID string) ([]Grant, error) {
	assignments, err := s.roles.GetRoleAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}
	return grantsOf(assignments), nil
}

func (s *Service) response(ctx context.Context, id string) (*EventResponse, error) {
	view, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load event: %w", err)
	}
	if view == nil {
		return nil, apperrors.NewNotFound("event not found")
	}
	responses, err := s.toResponses(ctx, []View{*view})
	if err != nil {
		return nil, err
	}
	return &responses[0], nil
}

func (s *Service) toResponses(ctx context.Context, views []View) ([]EventResponse, error) {
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
	}
	rules, err := s.repository.AudienceRules(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load audiences: %w", err)
	}
	reviews, err := s.repository.Reviews(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load reviews: %w", err)
	}
	reviewsByEvent := map[string][]ReviewResponse{}
	for _, review := range reviews {
		reviewsByEvent[review.EventID] = append(reviewsByEvent[review.EventID], ReviewResponse{
			Stage: review.Stage, Decision: review.Decision, Note: review.Note, ReviewerName: review.ReviewerName, DecidedAt: review.CreatedAt,
		})
	}
	byEvent := map[string][]AudienceRuleResponse{}
	for _, rule := range rules {
		byEvent[rule.TargetID] = append(byEvent[rule.TargetID], AudienceRuleResponse{
			DepartmentID: rule.DepartmentID, DepartmentCode: rule.DepartmentCode, BatchYear: rule.BatchYear, Role: rule.Role,
		})
	}
	responses := make([]EventResponse, 0, len(views))
	for _, view := range views {
		response := EventResponse{
			ID:           view.ID,
			Title:        view.Title,
			Description:  view.Description,
			EventType:    view.EventType,
			Status:       view.Status,
			ProposerID:   view.ProposerID,
			ProposerName: view.ProposerName,
			Organiser:    &OrganiserRef{UserID: view.OrganiserID, FullName: view.OrganiserName},
			Location:     view.Location,
			StartsAt:     view.StartsAt,
			EndsAt:       view.EndsAt,
			Capacity:     view.Capacity,
			Audience:     byEvent[view.ID],
			SubmittedAt:  view.SubmittedAt,
			PublishedAt:  view.PublishedAt,
			CancelledAt:  view.CancelledAt,
			CancelReason: view.CancelReason,
			CreatedAt:    view.CreatedAt,
			UpdatedAt:    view.UpdatedAt,
			Reviews:      reviewsByEvent[view.ID],
		}
		if response.Reviews == nil {
			response.Reviews = []ReviewResponse{}
		}
		if response.Audience == nil {
			response.Audience = []AudienceRuleResponse{}
		}
		if view.DepartmentID != nil && view.DepartmentCode != nil {
			response.Department = &DepartmentRef{ID: *view.DepartmentID, Code: *view.DepartmentCode}
		}
		if view.FacultyMentorID != nil {
			response.FacultyMentor = &MentorRef{UserID: *view.FacultyMentorID}
			if view.MentorName != nil {
				response.FacultyMentor.FullName = *view.MentorName
			}
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func audit(ctx context.Context, repositories Repositories, actorID, action, eventID string, metadata map[string]string, now time.Time) error {
	id := eventID
	if err := repositories.AuditLogs.Create(ctx, &auth.AuditLog{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "event",
		ResourceID:   &id,
		Metadata:     metadata,
		CreatedAt:    now,
	}); err != nil {
		return fmt.Errorf("audit %s: %w", action, err)
	}
	return nil
}

func blankToNil(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func checkLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > maxLimit {
		return 0, apperrors.NewValidation("invalid limit", map[string]string{"limit": fmt.Sprintf("use 1 to %d", maxLimit)})
	}
	return limit, nil
}

func encodeCursor(cursor Cursor) string {
	raw := cursor.At.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
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
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, invalid
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, invalid
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return nil, invalid
	}
	return &Cursor{At: at, ID: parts[1]}, nil
}
