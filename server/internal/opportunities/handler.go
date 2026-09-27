package opportunities

import (
	"context"
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
	v1.PATCH("/opportunity-applications/:id/status", h.UpdateStatus)
	opportunities := v1.Group("/opportunities")
	opportunities.GET("", h.Feed)
	opportunities.POST("", h.Create)
	opportunities.GET("/manage", h.Managed)
	opportunities.GET("/:id", h.Get)
	opportunities.PATCH("/:id", h.Update)
	opportunities.POST("/:id/publish", h.staffAction((*Service).Publish))
	opportunities.POST("/:id/close", h.staffAction((*Service).Close))
	opportunities.POST("/:id/apply", h.applicantAction((*Service).Apply, http.StatusCreated))
	opportunities.POST("/:id/withdraw", h.applicantAction((*Service).Withdraw, http.StatusOK))
	opportunities.GET("/:id/applications", h.Applicants)
}

// Applicants lists an Opportunity's Applications for placement staff.
func (h *Handler) Applicants(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionViewApplicantData)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	items, meta, err := h.service.Applicants(c.Request.Context(), actor.UserID, c.Param("id"), ApplicantQuery{
		Status:     c.Query("status"),
		Department: c.Query("department"),
		Batch:      c.Query("batch"),
		Cursor:     c.Query("cursor"),
		Limit:      limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

// UpdateStatus moves one Application for placement staff.
func (h *Handler) UpdateStatus(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionShortlistApplicants)
	if actor == nil {
		return
	}
	var input StatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	application, err := h.service.UpdateStatus(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, application, nil)
}

// applicantAction serves a Student's action on their own Application. Who
// may apply is decided in the service.
func (h *Handler) applicantAction(action func(*Service, context.Context, string, string) (*ApplicationResponse, error), status int) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := h.signedIn(c)
		if actor == nil {
			return
		}
		application, err := action(h.service, c.Request.Context(), actor.UserID, c.Param("id"))
		if err != nil {
			writeError(c, err)
			return
		}
		response.Success(c, status, application, nil)
	}
}

// Feed lists the Opportunities the caller is eligible for.
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
		State:      c.Query("state"),
		Type:       c.Query("type"),
		Department: c.Query("department"),
		Cursor:     c.Query("cursor"),
		Limit:      limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, items, meta)
}

// staffAction serves a placement staff status change on one Opportunity.
func (h *Handler) staffAction(action func(*Service, context.Context, string, string) (*OpportunityResponse, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := h.authorize(c, auth.PermissionPostOpportunity)
		if actor == nil {
			return
		}
		opportunity, err := action(h.service, c.Request.Context(), actor.UserID, c.Param("id"))
		if err != nil {
			writeError(c, err)
			return
		}
		response.Success(c, http.StatusOK, opportunity, nil)
	}
}

func (h *Handler) Create(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostOpportunity)
	if actor == nil {
		return
	}
	var input CreateOpportunityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	opportunity, err := h.service.Create(c.Request.Context(), actor.UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, opportunity, nil)
}

func (h *Handler) Update(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostOpportunity)
	if actor == nil {
		return
	}
	var input UpdateOpportunityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	opportunity, err := h.service.Update(c.Request.Context(), actor.UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, opportunity, nil)
}

// Get needs only a signed-in member; what they may see is decided in the service.
func (h *Handler) Get(c *gin.Context) {
	actor := h.signedIn(c)
	if actor == nil {
		return
	}
	opportunity, err := h.service.Get(c.Request.Context(), actor.UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, opportunity, nil)
}

// Managed lists every Opportunity for placement staff.
func (h *Handler) Managed(c *gin.Context) {
	actor := h.authorize(c, auth.PermissionPostOpportunity)
	if actor == nil {
		return
	}
	limit, ok := parseLimit(c)
	if !ok {
		return
	}
	items, meta, err := h.service.Managed(c.Request.Context(), actor.UserID, c.Query("status"), c.Query("cursor"), limit)
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
