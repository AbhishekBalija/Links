package announcements

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	policy  *auth.Policy
}

func NewHandler(service *Service, policy *auth.Policy) *Handler {
	return &Handler{service: service, policy: policy}
}

func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	announcements := v1.Group("/announcements")
	announcements.GET("", h.Feed)
	announcements.POST("", h.Create)
	announcements.GET("/mine", h.Mine)
	announcements.GET("/approvals", h.Queue)
	announcements.PATCH("/:id", h.Update)
	announcements.POST("/:id/submit-for-approval", h.Submit)
	announcements.PATCH("/:id/approval", h.Review)
	announcements.POST("/:id/withdraw", h.Withdraw)
}

func (h *Handler) Feed(c *gin.Context) {
	h.list(c, auth.PermissionViewTargetedNotices, h.service.Feed)
}

func (h *Handler) Create(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostAnnouncement)
	if actor == nil {
		return
	}
	var input CreateAnnouncementInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	result, err := h.service.Create(c.Request.Context(), actor.UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, result, nil)
}

// Mine only ever lists the caller's own Announcements, so anyone signed in may
// see it, including authors whose posting role has since ended.
func (h *Handler) Mine(c *gin.Context) {
	h.list(c, auth.PermissionViewTargetedNotices, h.service.Mine)
}

func (h *Handler) Queue(c *gin.Context) {
	h.list(c, auth.PermissionApproveAnnouncement, h.service.Queue)
}

func (h *Handler) Update(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostAnnouncement)
	if actor == nil {
		return
	}
	var input UpdateAnnouncementInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	result, err := h.service.Update(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) Submit(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostAnnouncement)
	if actor == nil {
		return
	}
	result, err := h.service.Submit(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) Review(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionApproveAnnouncement)
	if actor == nil {
		return
	}
	var input ReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	result, err := h.service.Review(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

// Withdraw is open to any signed-in user; the service decides who may withdraw.
func (h *Handler) Withdraw(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionViewTargetedNotices)
	if actor == nil {
		return
	}
	result, err := h.service.Withdraw(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

type listFunc func(ctx context.Context, actorID, cursor string, limit int) ([]AnnouncementResponse, *FeedMeta, error)

// list serves a cursor-paginated list after checking the permission.
func (h *Handler) list(c *gin.Context, permission auth.Permission, load listFunc) {
	actor := h.authorize(c, permission)
	if actor == nil {
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid limit", map[string]string{"limit": "use a whole number"})
		return
	}
	items, meta, err := load(c.Request.Context(), actor.UserID, c.Query("cursor"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

func (h *Handler) authorize(c *gin.Context, permission auth.Permission) *auth.Actor {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return nil
	}
	if err := auth.AuthorizeActor(c, h.policy, permission); err != nil {
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
