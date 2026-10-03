package auth

import (
	"errors"
	"io"
	"net/http"
	"strconv"

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
	admin.GET("/not-signed-in", h.NotSignedIn)
	admin.GET("/not-signed-in/emails", h.NotSignedInEmails)
	admin.POST("", h.InviteStaff)
	admin.PATCH("/:id/verify", h.VerifyUser)
	admin.PATCH("/:id/status", h.UpdateUserStatus)
	admin.POST("/import", h.ImportStudents)
	admin.GET("/:id/roles", h.ListRoles)
	admin.POST("/:id/roles", h.GrantRole)
	admin.DELETE("/:id/roles/:roleAssignmentId", h.EndRole)
	admin.GET("/:id/roles/:roleAssignmentId/ending", h.PreviewEndRole)
}

func (h *AdminHandler) ReviewQueue(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	if err := AuthorizeActor(c, h.policy, PermissionApproveAccess); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	resp, err := h.service.ReviewQueue(c.Request.Context(), actor.UserID)
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

	if err := AuthorizeActor(c, h.policy, PermissionApproveAccess); err != nil {
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

	// Rejecting an Access request is part of deciding it, which HODs do for
	// their Department; suspending or reactivating is for the principal and
	// admins.
	permission := PermissionManageUsersAndRoles
	if UserStatus(input.Status) == UserStatusRejected {
		permission = PermissionApproveAccess
	}
	if err := AuthorizeActor(c, h.policy, permission); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	if err := h.service.UpdateUserStatus(c.Request.Context(), actor.UserID, userID, input.Status, input.Note); err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, UpdateUserStatusResponse{Message: "user status updated successfully"}, nil)
}

func (h *AdminHandler) ListRoles(c *gin.Context) {
	if !h.authorizeRoleManager(c) {
		return
	}
	resp, err := h.service.ListUserRoles(c.Request.Context(), GetActor(c).UserID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, nil)
}

func (h *AdminHandler) GrantRole(c *gin.Context) {
	if !h.authorizeRoleManager(c) {
		return
	}
	var input GrantRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	resp, err := h.service.GrantRole(c.Request.Context(), GetActor(c).UserID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, resp, nil)
}

func (h *AdminHandler) EndRole(c *gin.Context) {
	if !h.authorizeRoleManager(c) {
		return
	}
	resp, err := h.service.EndRole(c.Request.Context(), GetActor(c).UserID, c.Param("id"), c.Param("roleAssignmentId"), c.Query("organiser_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, nil)
}

// PreviewEndRole says what ending the role would do, for the confirmation,
// without changing anything.
func (h *AdminHandler) PreviewEndRole(c *gin.Context) {
	if !h.authorizeRoleManager(c) {
		return
	}
	resp, err := h.service.PreviewEndRole(c.Request.Context(), GetActor(c).UserID, c.Param("id"), c.Param("roleAssignmentId"), c.Query("organiser_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, nil)
}

// authorizeManager writes the error response and returns false unless the
// caller may manage users and roles.
func (h *AdminHandler) authorizeManager(c *gin.Context) bool {
	if GetActor(c) == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return false
	}
	if err := AuthorizeActor(c, h.policy, PermissionManageUsersAndRoles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return false
	}
	return true
}

// authorizeRoleManager lets in everyone who manages some roles; the service
// checks which roles and which users (ADR 0027).
func (h *AdminHandler) authorizeRoleManager(c *gin.Context) bool {
	if GetActor(c) == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return false
	}
	if err := AuthorizeActor(c, h.policy, PermissionManageRoles); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return false
	}
	return true
}

// InviteStaff adds a staff member by email and role (admins, and HODs for
// faculty of their own Department).
func (h *AdminHandler) InviteStaff(c *gin.Context) {
	if GetActor(c) == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}
	if err := AuthorizeActor(c, h.policy, PermissionInviteStaff); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}
	var input InviteStaffInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	resp, err := h.service.InviteStaff(c.Request.Context(), GetActor(c).UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, resp, nil)
}

// maxDepartmentCodeLength bounds the import's department field; codes are
// 2 to 10 letters.
const maxDepartmentCodeLength = 10

func (h *AdminHandler) ImportStudents(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}
	if err := AuthorizeActor(c, h.policy, PermissionImportStudents); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}
	upload, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "upload the CSV as the multipart field \"file\"", map[string]string{"file": "required"})
		return
	}
	if upload.Size > MaxImportBytes {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "the file is too large", map[string]string{"file": "at most 1 MB"})
		return
	}
	file, err := upload.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "the file could not be read", nil)
		return
	}
	defer file.Close()

	dryRun := false
	if value := c.PostForm("dry_run"); value != "" {
		if dryRun, err = strconv.ParseBool(value); err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "dry_run must be true or false", map[string]string{"dry_run": "must be true or false"})
			return
		}
	}
	department := c.PostForm("department")
	if len(department) > maxDepartmentCodeLength {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "unknown department", map[string]string{"department": "not a department code"})
		return
	}

	input := ImportInput{File: file, DepartmentCode: department, DryRun: dryRun}
	resp, err := h.service.ImportStudents(c.Request.Context(), actor.UserID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, nil)
}

// NotSignedIn lists who class lists and staff invites let in and who hasn't
// signed in yet. Whoever decides Access requests may see it; the service
// keeps an HOD to their own Department.
func (h *AdminHandler) NotSignedIn(c *gin.Context) {
	if !h.authorizeAccessDecider(c) {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := h.service.NotSignedIn(c.Request.Context(), GetActor(c).UserID, NotSignedInQuery{
		Department: c.Query("department"), Kind: c.Query("kind"), Cursor: c.Query("cursor"), Limit: limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	meta := gin.H{"total": page.Total}
	if page.NextCursor != "" {
		meta["next_cursor"] = page.NextCursor
	}
	response.Success(c, http.StatusOK, page.People, meta)
}

// NotSignedInEmails gives every matching email, for a reminder sent from
// the college's own mail.
func (h *AdminHandler) NotSignedInEmails(c *gin.Context) {
	if !h.authorizeAccessDecider(c) {
		return
	}
	emails, err := h.service.NotSignedInEmails(c.Request.Context(), GetActor(c).UserID, NotSignedInQuery{
		Department: c.Query("department"), Kind: c.Query("kind"),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"emails": emails}, nil)
}

func (h *AdminHandler) authorizeAccessDecider(c *gin.Context) bool {
	if GetActor(c) == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return false
	}
	if err := AuthorizeActor(c, h.policy, PermissionApproveAccess); err != nil {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return false
	}
	return true
}
