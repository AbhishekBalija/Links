package events

import (
	"errors"
	"net/http"
	"strconv"

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
	events := v1.Group("/events")
	events.GET("", h.Feed)
	events.POST("", h.Create)
	events.GET("/mine", h.Mine)
	events.GET("/reviews", h.Queue)
	events.GET("/:id", h.Get)
	events.POST("/:id/rsvp", h.RSVP)
	events.GET("/:id/rsvps", h.RSVPs)
	events.GET("/:id/export", h.Export)
	events.POST("/:id/cancel", h.Cancel)
	events.PATCH("/:id", h.Update)
	events.DELETE("/:id", h.DeleteDraft)
	events.POST("/:id/submit-for-approval", h.Submit)
	events.PATCH("/:id/hod-review", h.review(StageHOD, auth.PermissionReviewBranchEvent))
	events.PATCH("/:id/final-approval", h.review(StageFinal, auth.PermissionFinalEventApproval))
}

func (h *Handler) Submit(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	event, err := h.service.Submit(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, event, nil)
}

// review serves one review stage; which Events the caller may review is
// decided in the service.
func (h *Handler) review(stage Stage, permission auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := h.authorize(c, permission)
		if actor == nil {
			return
		}
		var input ReviewInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		event, err := h.service.Review(c.Request.Context(), actor.UserID, c.Param("id"), stage, input)
		if err != nil {
			writeError(c, err)
			return
		}
		response.Success(c, http.StatusOK, event, nil)
	}
}

// Queue lists what waits for the caller's review.
func (h *Handler) Queue(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionReviewBranchEvent)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	items, meta, err := h.service.Queue(c.Request.Context(), actor.UserID, c.Query("cursor"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

func (h *Handler) Create(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionProposeEvent)
	if actor == nil {
		return
	}
	var input CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	event, err := h.service.Create(c.Request.Context(), actor.UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, event, nil)
}

// Update needs no permission beyond signing in: the service only finds the
// caller's own Events.
func (h *Handler) Update(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	var input UpdateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	event, err := h.service.Update(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, event, nil)
}

// Mine is open to any signed-in user, so former proposers keep their history.
func (h *Handler) Mine(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	items, meta, err := h.service.Mine(c.Request.Context(), actor.UserID, c.Query("status"), c.Query("cursor"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

func parseLimit(c *gin.Context) (int, bool) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid limit", map[string]string{"limit": "use a whole number"})
		return 0, false
	}
	return limit, true
}

func (h *Handler) signedIn(c *gin.Context) *auth.Actor {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
	}
	return actor
}

func (h *Handler) authorize(c *gin.Context, permission auth.Permission) *auth.Actor {
	actor := h.signedIn(c)
	if actor == nil {
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

// Feed lists published Events for the reader's Audience.
func (h *Handler) Feed(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionViewTargetedNotices)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	items, meta, err := h.service.Feed(c.Request.Context(), actor.UserID, FeedQuery{
		From:       c.Query("from"),
		To:         c.Query("to"),
		Department: c.Query("department"),
		EventType:  c.Query("event_type"),
		Show:       c.Query("show"),
		Cursor:     c.Query("cursor"),
		Limit:      limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

// Get returns one Event; the service decides who may see it.
func (h *Handler) Get(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	event, err := h.service.Get(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, event, nil)
}

func (h *Handler) RSVP(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionViewTargetedNotices)
	if actor == nil {
		return
	}
	var input RSVPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	summary, err := h.service.RSVP(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, summary, nil)
}

func (h *Handler) RSVPs(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	summary, meta, err := h.service.RSVPs(c.Request.Context(), actor.UserID, c.Param("id"), c.Query("cursor"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, summary, meta)
}

// Export sends the participant list as a CSV download.
func (h *Handler) Export(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	file, err := h.service.Export(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="event-participants.csv"`)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", file)
}

// DeleteDraft removes the caller's own draft; anything submitted is cancelled
// instead, so its history stays.
func (h *Handler) DeleteDraft(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	if err := h.service.DeleteDraft(c.Request.Context(), actor.UserID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Cancel(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	var input CancelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	event, err := h.service.Cancel(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, event, nil)
}
