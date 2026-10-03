// Package dashboard builds the signed-in user's Home summary from other
// modules. It owns no data: each section comes from the module behind it, and
// later features add sections without changing these.
package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/announcements"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/events"
	"github.com/AbhishekBalija/Links/server/internal/opportunities"
	"github.com/AbhishekBalija/Links/server/internal/shared/authorwork"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const (
	// workOnHome is how many sent back and how many waiting items Home lists.
	workOnHome          = 10
	noticesOnHome       = 5
	opportunitiesOnHome = 3
)

// Announcements is what the dashboard needs from the announcements module.
type Announcements interface {
	Feed(ctx context.Context, actorID, category, cursor string, limit int) ([]announcements.AnnouncementResponse, *announcements.FeedMeta, error)
	ApprovalSummary(ctx context.Context, actorID string) (*announcements.ApprovalSummary, error)
	AuthorSummary(ctx context.Context, actorID string) (*announcements.AuthorSummary, error)
	AuthorWork(ctx context.Context, actorID string, limit int) ([]authorwork.SentBack, []authorwork.Waiting, error)
	Grants(ctx context.Context, userID string) ([]announcements.Grant, error)
}

// Events is what the dashboard needs from the events module.
type Events interface {
	ReviewSummary(ctx context.Context, actorID string) (*events.ReviewSummary, error)
	AuthorWork(ctx context.Context, actorID string, limit int) ([]authorwork.SentBack, []authorwork.Waiting, error)
	UpcomingInDepartment(ctx context.Context, departmentID string, limit int) ([]events.DepartmentEvent, error)
}

// Opportunities is what the dashboard needs from the opportunities module.
type Opportunities interface {
	Feed(ctx context.Context, actorID string, query opportunities.FeedQuery) ([]opportunities.OpportunityResponse, *opportunities.ListMeta, error)
	PlacementSummary(ctx context.Context, actorID string) (*opportunities.PlacementSummary, error)
}

// Department is the user's Department as shown on Home.
type Department struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type UserSection struct {
	FullName   string      `json:"full_name"`
	Roles      []string    `json:"roles"`
	Department *Department `json:"department"`
}

type NoticesSection struct {
	Items   []announcements.AnnouncementResponse `json:"items"`
	HasMore bool                                 `json:"has_more"`
}

// ApprovalsSection is what waits for a reviewer: Announcements (and edits)
// and Event proposals, counted apart.
type ApprovalsSection struct {
	PendingCount           int        `json:"pending_count"`
	OldestSubmittedAt      *time.Time `json:"oldest_submitted_at"`
	EventsPendingCount     int        `json:"events_pending_count"`
	OldestEventSubmittedAt *time.Time `json:"oldest_event_submitted_at"`
}

// OpportunitiesSection is the next open Opportunities the user is eligible
// for, soonest deadline first.
type OpportunitiesSection struct {
	Items   []opportunities.OpportunityResponse `json:"items"`
	HasMore bool                                `json:"has_more"`
}

// Response is the Home summary. Sections a user doesn't need are left out.
type Response struct {
	User            UserSection                     `json:"user"`
	Notices         NoticesSection                  `json:"notices"`
	Approvals       *ApprovalsSection               `json:"approvals,omitempty"`
	MyAnnouncements *announcements.AuthorSummary    `json:"my_announcements,omitempty"`
	MyWork          *MyWorkSection                  `json:"my_work,omitempty"`
	Opportunities   *OpportunitiesSection           `json:"opportunities,omitempty"`
	Placement       *opportunities.PlacementSummary `json:"placement,omitempty"`
	Department      *DepartmentSection              `json:"department,omitempty"`
	College         *CollegeSection                 `json:"college,omitempty"`
	AccessRequests  *auth.AccessSummary             `json:"access_requests,omitempty"`
	Lists           *auth.ListsSummary              `json:"lists,omitempty"`
}

// Repository reads the profile details Home shows.
type Repository interface {
	Profile(ctx context.Context, userID string) (fullName string, studentDepartmentID *string, err error)
	Department(ctx context.Context, id string) (*Department, error)
}

type Service struct {
	repository    Repository
	announcements Announcements
	events        Events
	opportunities Opportunities
	directory     Directory
	departments   Departments
	access        Access
}

func NewService(repository Repository, announcements Announcements, events Events, opportunities Opportunities, directory Directory, departments Departments, access Access) *Service {
	return &Service{
		repository:    repository,
		announcements: announcements,
		events:        events,
		opportunities: opportunities,
		directory:     directory,
		departments:   departments,
		access:        access,
	}
}

func (s *Service) Get(ctx context.Context, userID string) (*Response, error) {
	grants, err := s.announcements.Grants(ctx, userID)
	if err != nil {
		return nil, err
	}
	user, err := s.user(ctx, userID, grants)
	if err != nil {
		return nil, err
	}
	notices, meta, err := s.announcements.Feed(ctx, userID, "", "", noticesOnHome)
	if err != nil {
		return nil, err
	}
	approvals, err := s.approvals(ctx, userID)
	if err != nil {
		return nil, err
	}
	mine, err := s.announcements.AuthorSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	myWork, err := s.myWork(ctx, userID, mine != nil)
	if err != nil {
		return nil, err
	}
	openings, err := s.openOpportunities(ctx, userID)
	if err != nil {
		return nil, err
	}
	placement, err := s.opportunities.PlacementSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	department, err := s.department(ctx, userID, grants)
	if err != nil {
		return nil, err
	}
	college, err := s.college(ctx, userID, user.Roles)
	if err != nil {
		return nil, err
	}
	access, err := s.access.AccessSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	lists, err := s.access.ListsSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Response{
		Lists:           lists,
		Department:      department,
		College:         college,
		AccessRequests:  access,
		User:            *user,
		Notices:         NoticesSection{Items: notices, HasMore: meta.NextCursor != ""},
		Approvals:       approvals,
		MyAnnouncements: mine,
		MyWork:          myWork,
		Opportunities:   openings,
		Placement:       placement,
	}, nil
}

// openOpportunities returns the first open Opportunities the user is
// eligible for, or nil when there are none.
func (s *Service) openOpportunities(ctx context.Context, userID string) (*OpportunitiesSection, error) {
	items, meta, err := s.opportunities.Feed(ctx, userID, opportunities.FeedQuery{Limit: opportunitiesOnHome})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &OpportunitiesSection{Items: items, HasMore: meta.NextCursor != ""}, nil
}

// approvals combines both review queues, or returns nil for a user who
// reviews neither.
func (s *Service) approvals(ctx context.Context, userID string) (*ApprovalsSection, error) {
	notices, err := s.announcements.ApprovalSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	proposals, err := s.events.ReviewSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	if notices == nil && proposals == nil {
		return nil, nil
	}
	section := &ApprovalsSection{}
	if notices != nil {
		section.PendingCount = notices.PendingCount
		section.OldestSubmittedAt = notices.OldestSubmittedAt
	}
	if proposals != nil {
		section.EventsPendingCount = proposals.PendingCount
		section.OldestEventSubmittedAt = proposals.OldestSubmittedAt
	}
	return section, nil
}

// user loads the name, current roles and Department: the Student identity's
// Department, otherwise the first Department-scoped role.
func (s *Service) user(ctx context.Context, userID string, grants []announcements.Grant) (*UserSection, error) {
	fullName, departmentID, err := s.repository.Profile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	section := &UserSection{FullName: fullName, Roles: []string{}}
	seen := map[string]bool{}
	for _, grant := range grants {
		if !seen[string(grant.Role)] {
			seen[string(grant.Role)] = true
			section.Roles = append(section.Roles, string(grant.Role))
		}
		if departmentID == nil && grant.DepartmentID != "" {
			id := grant.DepartmentID
			departmentID = &id
		}
	}
	if departmentID != nil {
		department, departmentErr := s.repository.Department(ctx, *departmentID)
		if departmentErr != nil {
			return nil, fmt.Errorf("load department: %w", departmentErr)
		}
		section.Department = department
	}
	return section, nil
}

type Handler struct {
	service *Service
	policy  *auth.Policy
}

func NewHandler(service *Service, policy *auth.Policy) *Handler {
	return &Handler{service: service, policy: policy}
}

func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/dashboard", h.Get)
}

func (h *Handler) Get(c *gin.Context) {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}
	if err := auth.AuthorizeActor(c, h.policy, auth.PermissionViewTargetedNotices); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}
	result, err := h.service.Get(c.Request.Context(), actor.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}
