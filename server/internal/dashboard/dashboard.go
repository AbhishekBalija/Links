// Package dashboard builds the signed-in user's Home summary from other
// modules. It owns no data: each section comes from the module behind it, and
// later features (placements, events) add sections without changing these.
package dashboard

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AbhishekBalija/Links/server/internal/announcements"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const noticesOnHome = 5

// Announcements is what the dashboard needs from the announcements module.
type Announcements interface {
	Feed(ctx context.Context, actorID, category, cursor string, limit int) ([]announcements.AnnouncementResponse, *announcements.FeedMeta, error)
	ApprovalSummary(ctx context.Context, actorID string) (*announcements.ApprovalSummary, error)
	AuthorSummary(ctx context.Context, actorID string) (*announcements.AuthorSummary, error)
	Grants(ctx context.Context, userID string) ([]announcements.Grant, error)
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

// Response is the Home summary. Sections a user doesn't need are left out.
type Response struct {
	User            UserSection                    `json:"user"`
	Notices         NoticesSection                 `json:"notices"`
	Approvals       *announcements.ApprovalSummary `json:"approvals,omitempty"`
	MyAnnouncements *announcements.AuthorSummary   `json:"my_announcements,omitempty"`
}

// Repository reads the profile details Home shows.
type Repository interface {
	Profile(ctx context.Context, userID string) (fullName string, studentDepartmentID *string, err error)
	Department(ctx context.Context, id string) (*Department, error)
}

type Service struct {
	repository    Repository
	announcements Announcements
}

func NewService(repository Repository, announcements Announcements) *Service {
	return &Service{repository: repository, announcements: announcements}
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
	approvals, err := s.announcements.ApprovalSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	mine, err := s.announcements.AuthorSummary(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Response{
		User:            *user,
		Notices:         NoticesSection{Items: notices, HasMore: meta.NextCursor != ""},
		Approvals:       approvals,
		MyAnnouncements: mine,
	}, nil
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
