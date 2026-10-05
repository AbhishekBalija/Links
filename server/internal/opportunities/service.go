package opportunities

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	maxTitleLength        = 200
	maxCompanyLength      = 200
	maxDescriptionLength  = 10000
	maxLocationLength     = 200
	maxCompensationLength = 200
	maxURLLength          = 2048
	maxEligibilityRules   = 20
	defaultLimit          = 20
	maxLimit              = 50
)

var eligibilityRoles = map[auth.Role]bool{
	auth.RoleStudent: true, auth.RoleStudentCoordinator: true, auth.RoleFaculty: true, auth.RoleHOD: true,
	auth.RolePlacementOfficer: true, auth.RolePrincipal: true, auth.RoleAlumni: true,
	auth.RoleClubOrganizer: true, auth.RoleAdmin: true,
}

type Service struct {
	repository Repository
	roles      RoleReader
	unitOfWork UnitOfWork
	notifier   Notifier
	now        func() time.Time
}

func NewService(repository Repository, roles RoleReader, unitOfWork UnitOfWork) *Service {
	return &Service{repository: repository, roles: roles, unitOfWork: unitOfWork, notifier: noNotifier{}, now: time.Now}
}

// content is the editable part of an Opportunity, validated.
type content struct {
	OpportunityType Type
	Title           string
	Company         string
	Description     string
	Location        *string
	Compensation    *string
	ApplyBy         time.Time
	ApplicationMode Mode
	ExternalURL     *string
	Eligibility     []EligibilityRule
}

// Create saves a new Opportunity as a draft. Only placement staff post.
func (s *Service) Create(ctx context.Context, actorID string, input CreateOpportunityInput) (*OpportunityResponse, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	if input.ApplyBy == nil {
		return nil, apperrors.NewValidation("invalid opportunity", map[string]string{"apply_by": "required"})
	}
	proposed, err := check(content{
		OpportunityType: Type(input.OpportunityType),
		Title:           input.Title,
		Company:         input.Company,
		Description:     input.Description,
		Location:        blankToNil(input.Location),
		Compensation:    blankToNil(input.Compensation),
		ApplyBy:         *input.ApplyBy,
		ApplicationMode: Mode(input.ApplicationMode),
		ExternalURL:     blankToNil(input.ExternalURL),
	}, input.Eligibility)
	if err != nil {
		return nil, err
	}

	now := s.now()
	opportunity := Opportunity{PostedBy: actorID, Status: StatusDraft, CreatedAt: now, UpdatedAt: now}
	apply(&opportunity, proposed)
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		if err := checkDepartments(ctx, repositories.Opportunities, proposed.Eligibility); err != nil {
			return err
		}
		if err := repositories.Opportunities.Create(ctx, &opportunity, proposed.Eligibility); err != nil {
			return fmt.Errorf("create opportunity: %w", err)
		}
		return audit(ctx, repositories, actorID, "opportunity_created", opportunity.ID, map[string]string{"status": string(opportunity.Status)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, opportunity.ID, actorID)
}

// Update edits an Opportunity. Placement staff work as one office, so any
// of them may edit any Opportunity.
func (s *Service) Update(ctx context.Context, actorID, id string, input UpdateOpportunityInput) (*OpportunityResponse, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		opportunity, err := repositories.Opportunities.FindForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("find opportunity: %w", err)
		}
		if opportunity == nil {
			return apperrors.NewNotFound("opportunity not found")
		}
		eligibility, err := repositories.Opportunities.Eligibility(ctx, opportunity.ID)
		if err != nil {
			return fmt.Errorf("load eligibility: %w", err)
		}
		changed, eligibilityInput := merge(*opportunity, eligibility, input)
		if err := checkLiveEdit(*opportunity, changed, now); err != nil {
			return err
		}
		checked, err := check(changed, eligibilityInput)
		if err != nil {
			return err
		}
		if err := checkDepartments(ctx, repositories.Opportunities, checked.Eligibility); err != nil {
			return err
		}
		apply(opportunity, checked)
		opportunity.UpdatedAt = now
		if err := repositories.Opportunities.Update(ctx, opportunity); err != nil {
			return fmt.Errorf("update opportunity: %w", err)
		}
		if input.Eligibility != nil {
			if err := repositories.Opportunities.ReplaceEligibility(ctx, opportunity.ID, checked.Eligibility); err != nil {
				return fmt.Errorf("replace eligibility: %w", err)
			}
		}
		return audit(ctx, repositories, actorID, "opportunity_updated", opportunity.ID, map[string]string{"status": string(opportunity.Status)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.response(ctx, id, actorID)
}

// Managed lists every Opportunity for placement staff, newest first,
// optionally one status only.
func (s *Service) Managed(ctx context.Context, actorID, status, cursor string, limit int) ([]OpportunityResponse, *ListMeta, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, nil, err
	}
	var filter *Status
	if status != "" {
		value := Status(status)
		if !validStatus(value) {
			return nil, nil, apperrors.NewValidation("invalid status", map[string]string{"status": "use draft, published or closed"})
		}
		filter = &value
	}
	limit, err := checkLimit(limit)
	if err != nil {
		return nil, nil, err
	}
	after, err := decodeCursor(cursor)
	if err != nil {
		return nil, nil, err
	}
	views, err := s.repository.Managed(ctx, filter, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list opportunities: %w", err)
	}
	meta := &ListMeta{}
	if len(views) > limit {
		views = views[:limit]
		last := views[len(views)-1]
		meta.NextCursor = encodeCursor(Cursor{At: last.CreatedAt, ID: last.ID})
	}
	responses, err := s.toResponses(ctx, views, actorID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.addApplicantCounts(ctx, responses); err != nil {
		return nil, nil, err
	}
	return responses, meta, nil
}

// merge applies the fields a PATCH sent onto the Opportunity's current
// content. It returns the Eligibility input to validate, or nil to keep the
// current one.
func merge(opportunity Opportunity, eligibility []EligibilityRule, input UpdateOpportunityInput) (content, []EligibilityRuleInput) {
	changed := content{
		OpportunityType: opportunity.OpportunityType,
		Title:           opportunity.Title,
		Company:         opportunity.Company,
		Description:     opportunity.Description,
		Location:        opportunity.Location,
		Compensation:    opportunity.Compensation,
		ApplyBy:         opportunity.ApplyBy,
		ApplicationMode: opportunity.ApplicationMode,
		ExternalURL:     opportunity.ExternalURL,
		Eligibility:     eligibility,
	}
	if input.OpportunityType != nil {
		changed.OpportunityType = Type(*input.OpportunityType)
	}
	if input.Title != nil {
		changed.Title = *input.Title
	}
	if input.Company != nil {
		changed.Company = *input.Company
	}
	if input.Description != nil {
		changed.Description = *input.Description
	}
	if input.Location.Set {
		changed.Location = blankToNil(input.Location.Value)
	}
	if input.Compensation.Set {
		changed.Compensation = blankToNil(input.Compensation.Value)
	}
	if input.ApplyBy != nil {
		changed.ApplyBy = *input.ApplyBy
	}
	if input.ApplicationMode != nil {
		changed.ApplicationMode = Mode(*input.ApplicationMode)
	}
	if input.ExternalURL.Set {
		changed.ExternalURL = blankToNil(input.ExternalURL.Value)
	}
	if input.Eligibility == nil {
		return changed, nil
	}
	return changed, *input.Eligibility
}

// check validates the fields. eligibilityInput replaces the Eligibility when
// not nil.
func check(proposed content, eligibilityInput []EligibilityRuleInput) (content, error) {
	details := map[string]string{}
	proposed.Title = strings.TrimSpace(proposed.Title)
	proposed.Company = strings.TrimSpace(proposed.Company)
	proposed.Description = strings.TrimSpace(proposed.Description)
	if !validType(proposed.OpportunityType) {
		details["opportunity_type"] = "use job, internship or training"
	}
	if length := utf8.RuneCountInString(proposed.Title); length < 3 || length > maxTitleLength {
		details["title"] = fmt.Sprintf("use 3 to %d characters", maxTitleLength)
	}
	if length := utf8.RuneCountInString(proposed.Company); length < 1 || length > maxCompanyLength {
		details["company"] = fmt.Sprintf("use 1 to %d characters", maxCompanyLength)
	}
	if utf8.RuneCountInString(proposed.Description) > maxDescriptionLength {
		details["description"] = fmt.Sprintf("use at most %d characters", maxDescriptionLength)
	}
	if proposed.Location != nil && utf8.RuneCountInString(*proposed.Location) > maxLocationLength {
		details["location"] = fmt.Sprintf("use at most %d characters", maxLocationLength)
	}
	if proposed.Compensation != nil && utf8.RuneCountInString(*proposed.Compensation) > maxCompensationLength {
		details["compensation"] = fmt.Sprintf("use at most %d characters", maxCompensationLength)
	}
	switch {
	case !validMode(proposed.ApplicationMode):
		details["application_mode"] = "use internal or external"
	case proposed.ApplicationMode == ModeExternal && proposed.ExternalURL == nil:
		details["external_url"] = "required when students apply on the company's site"
	case proposed.ApplicationMode == ModeInternal && proposed.ExternalURL != nil:
		details["external_url"] = "only for external applications"
	case proposed.ExternalURL != nil && !validURL(*proposed.ExternalURL):
		details["external_url"] = "must be an http or https link"
	}
	if eligibilityInput != nil {
		eligibility, problem := toEligibility(eligibilityInput)
		if problem != "" {
			details["eligibility"] = problem
		}
		proposed.Eligibility = eligibility
	}
	if len(details) > 0 {
		return proposed, apperrors.NewValidation("invalid opportunity", details)
	}
	return proposed, nil
}

func validURL(value string) bool {
	if len(value) > maxURLLength {
		return false
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

// toEligibility validates Eligibility rules. An empty list means everyone.
func toEligibility(inputs []EligibilityRuleInput) ([]EligibilityRule, string) {
	if len(inputs) > maxEligibilityRules {
		return nil, fmt.Sprintf("use at most %d rules", maxEligibilityRules)
	}
	rules := make([]EligibilityRule, 0, len(inputs))
	for _, input := range inputs {
		if input.DepartmentID == nil && input.BatchYear == nil && input.Role == nil {
			return nil, "each rule needs a department, batch year or role; leave eligibility empty for everyone"
		}
		rule := EligibilityRule{BatchYear: input.BatchYear}
		if input.DepartmentID != nil {
			if _, err := uuid.Parse(*input.DepartmentID); err != nil {
				return nil, "a department in the eligibility doesn't exist"
			}
			id := *input.DepartmentID
			rule.DepartmentID = &id
		}
		if input.BatchYear != nil && (*input.BatchYear < 2000 || *input.BatchYear > 2100) {
			return nil, "batch year must be between 2000 and 2100"
		}
		if input.Role != nil {
			role := auth.Role(*input.Role)
			if !eligibilityRoles[role] {
				return nil, fmt.Sprintf("%q is not a LINKS role", *input.Role)
			}
			rule.Role = &role
		}
		rules = append(rules, rule)
	}
	return rules, ""
}

// checkDepartments makes sure the Eligibility's Departments exist,
// share-locking them for the rest of the transaction.
func checkDepartments(ctx context.Context, repository Repository, eligibility []EligibilityRule) error {
	seen := map[string]bool{}
	ids := []string{}
	for _, rule := range eligibility {
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
		return fmt.Errorf("check eligibility departments: %w", err)
	}
	if found != len(ids) {
		return apperrors.NewValidation("invalid opportunity", map[string]string{"eligibility": "a department in the eligibility doesn't exist"})
	}
	return nil
}

func apply(opportunity *Opportunity, proposed content) {
	opportunity.OpportunityType = proposed.OpportunityType
	opportunity.Title = proposed.Title
	opportunity.Company = proposed.Company
	opportunity.Description = proposed.Description
	opportunity.Location = proposed.Location
	opportunity.Compensation = proposed.Compensation
	opportunity.ApplyBy = proposed.ApplyBy
	opportunity.ApplicationMode = proposed.ApplicationMode
	opportunity.ExternalURL = proposed.ExternalURL
}

// isStaff reports whether the user holds a placement staff role now, read
// from the database rather than the token.
func (s *Service) isStaff(ctx context.Context, userID string) (bool, error) {
	assignments, err := s.roles.GetRoleAssignments(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("load roles: %w", err)
	}
	for _, assignment := range assignments {
		switch assignment.Role {
		case auth.RolePlacementOfficer, auth.RolePrincipal, auth.RoleAdmin:
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) requireStaff(ctx context.Context, userID string) error {
	staff, err := s.isStaff(ctx, userID)
	if err != nil {
		return err
	}
	if !staff {
		return apperrors.NewForbidden("only placement staff can manage opportunities")
	}
	return nil
}

// response loads one Opportunity as the viewer sees it, with their own
// Application if they made one.
func (s *Service) response(ctx context.Context, id, viewerID string) (*OpportunityResponse, error) {
	view, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load opportunity: %w", err)
	}
	if view == nil {
		return nil, apperrors.NewNotFound("opportunity not found")
	}
	responses, err := s.toResponses(ctx, []View{*view}, viewerID)
	if err != nil {
		return nil, err
	}
	return &responses[0], nil
}

func (s *Service) toResponses(ctx context.Context, views []View, viewerID string) ([]OpportunityResponse, error) {
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
	}
	rules, err := s.repository.EligibilityRules(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load eligibility: %w", err)
	}
	applications, err := s.repository.ApplicationsOf(ctx, viewerID, ids)
	if err != nil {
		return nil, fmt.Errorf("load applications: %w", err)
	}
	mine := make(map[string]*ApplicationResponse, len(applications))
	for _, application := range applications {
		mine[application.OpportunityID] = toApplicationResponse(application)
	}
	byOpportunity := map[string][]EligibilityRuleResponse{}
	for _, rule := range rules {
		byOpportunity[rule.TargetID] = append(byOpportunity[rule.TargetID], EligibilityRuleResponse{
			DepartmentID: rule.DepartmentID, DepartmentCode: rule.DepartmentCode, BatchYear: rule.BatchYear, Role: rule.Role,
		})
	}
	now := s.now()
	responses := make([]OpportunityResponse, 0, len(views))
	for _, view := range views {
		response := OpportunityResponse{
			ID:              view.ID,
			OpportunityType: view.OpportunityType,
			Title:           view.Title,
			Company:         view.Company,
			Description:     view.Description,
			Location:        view.Location,
			Compensation:    view.Compensation,
			ApplyBy:         view.ApplyBy,
			ApplicationMode: view.ApplicationMode,
			ExternalURL:     view.ExternalURL,
			Eligibility:     byOpportunity[view.ID],
			Status:          view.Status,
			Open:            view.Status == StatusPublished && view.ApplyBy.After(now),
			MyApplication:   mine[view.ID],
			PostedBy:        PosterRef{UserID: view.PostedBy, FullName: view.PosterName},
			PublishedAt:     view.PublishedAt,
			ClosedAt:        view.ClosedAt,
			CreatedAt:       view.CreatedAt,
			UpdatedAt:       view.UpdatedAt,
		}
		if response.Eligibility == nil {
			response.Eligibility = []EligibilityRuleResponse{}
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func audit(ctx context.Context, repositories Repositories, actorID, action, resourceID string, metadata map[string]string, now time.Time) error {
	id := resourceID
	if err := repositories.AuditLogs.Create(ctx, &auth.AuditLog{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: "opportunity",
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
