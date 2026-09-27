package directory

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
)

type Handler struct {
	service *Service
	policy  *auth.Policy
}

func NewHandler(service *Service, policy *auth.Policy) *Handler {
	return &Handler{service: service, policy: policy}
}

func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/directory", h.List)
	v1.GET("/departments/:code/overview", h.Overview)
}

func (h *Handler) Overview(c *gin.Context) {
	actor := h.authorize(c)
	if actor == nil {
		return
	}
	overview, err := h.service.Overview(c.Request.Context(), actor.UserID, c.Param("code"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, overview, nil)
}

func (h *Handler) List(c *gin.Context) {
	actor := h.authorize(c)
	if actor == nil {
		return
	}
	entries, meta, err := h.service.List(c.Request.Context(), actor.UserID, ListQuery{
		Department: c.Query("department"),
		Role:       c.Query("role"),
		Batch:      c.Query("batch"),
		Q:          c.Query("q"),
		Limit:      c.Query("limit"),
		Cursor:     c.Query("cursor"),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, entries, meta)
}

// authorize lets any member in: everyone may view public profiles.
func (h *Handler) authorize(c *gin.Context) *auth.Actor {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return nil
	}
	if err := auth.AuthorizeActor(c, h.policy, auth.PermissionViewPublicProfiles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return nil
	}
	return actor
}

func writeError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
}
