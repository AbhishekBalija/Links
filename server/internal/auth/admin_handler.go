package auth

import (
	"errors"
	"io"
	"net/http"

	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service AuthService
	policy  *Policy
}

func NewAdminHandler(service AuthService, policy *Policy) *AdminHandler {
	return &AdminHandler{service: service, policy: policy}
}

func (h *AdminHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/users")
	admin.GET("/review-queue", h.ReviewQueue)
	admin.PATCH("/:id/verify", h.VerifyUser)
	admin.PATCH("/:id/status", h.UpdateUserStatus)
}

func (h *AdminHandler) ReviewQueue(c *gin.Context) {
	if GetActor(c) == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	if err := AuthorizeActor(c, h.policy, PermissionManageUsersAndRoles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	resp, err := h.service.ReviewQueue(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp, nil)
}

func (h *AdminHandler) VerifyUser(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	if err := AuthorizeActor(c, h.policy, PermissionManageUsersAndRoles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	userID := c.Param("id")
	if userID == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "user id is required", nil)
		return
	}

	var input VerifyUserInput
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	if err := h.service.VerifyUser(c.Request.Context(), actor.UserID, userID, input.ScopeType, input.ScopeID, input.Note); err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, VerifyUserResponse{Message: "user verified successfully"}, nil)
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	if err := AuthorizeActor(c, h.policy, PermissionManageUsersAndRoles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	userID := c.Param("id")
	if userID == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "user id is required", nil)
		return
	}

	var input UpdateUserStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	if err := h.service.UpdateUserStatus(c.Request.Context(), actor.UserID, userID, input.Status, input.Note); err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, UpdateUserStatusResponse{Message: "user status updated successfully"}, nil)
}
