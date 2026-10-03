package departments

import (
	"errors"
	"net/http"

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
	departments := v1.Group("/departments")
	departments.GET("", h.List)
	departments.GET("/:code", h.GetByCode)

	admin := v1.Group("/admin/departments")
	admin.GET("", h.ListForAdmin)
	admin.POST("", h.Create)
	admin.PUT("/:code", h.Update)
	admin.PATCH("/:code", h.Rename)
	admin.DELETE("/:code", h.Delete)
}

// publicListMaxAge lets browsers and the CDN reuse the list for a few minutes;
// a new department shows up on the sign-up form within that time.
const publicListMaxAge = "public, max-age=300"

// RegisterPublicRoutes adds the routes that work without a token.
func (h *Handler) RegisterPublicRoutes(v1 *gin.RouterGroup) {
	v1.GET("/public/departments", h.ListPublic)
}

func (h *Handler) ListPublic(c *gin.Context) {
	result, err := h.service.ListPublic(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", publicListMaxAge)
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) ListForAdmin(c *gin.Context) {
	if h.authorizeAdmin(c) == nil {
		return
	}
	result, err := h.service.ListForAdmin(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) GetByCode(c *gin.Context) {
	result, err := h.service.GetByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) Create(c *gin.Context) {
	actor := h.authorizeAdmin(c)
	if actor == nil {
		return
	}
	var input CreateDepartmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.service.Create(c.Request.Context(), actor.UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, result, nil)
}

func (h *Handler) Update(c *gin.Context) {
	actor := h.authorizeAdmin(c)
	if actor == nil {
		return
	}
	var input UpdateDepartmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.service.Update(c.Request.Context(), actor.UserID, c.Param("code"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) Rename(c *gin.Context) {
	actor := h.authorizeAdmin(c)
	if actor == nil {
		return
	}
	var input RenameDepartmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.service.Rename(c.Request.Context(), actor.UserID, c.Param("code"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, result, nil)
}

func (h *Handler) Delete(c *gin.Context) {
	actor := h.authorizeAdmin(c)
	if actor == nil {
		return
	}
	if err := h.service.Delete(c.Request.Context(), actor.UserID, c.Param("code")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) authorizeAdmin(c *gin.Context) *auth.Actor {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return nil
	}
	if err := auth.AuthorizeActor(c, h.policy, auth.PermissionManageDepartments); err != nil {
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
	response.InternalError(c, err)
}
