package profiles

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
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api, v1 *gin.RouterGroup, tokenCfg auth.TokenConfig) {
	api.GET("/v1/profiles/:username", auth.OptionalAuth(tokenCfg), h.GetPublicProfile)
	v1.PATCH("/me/profile", h.UpdateMyProfile)
}

func (h *Handler) GetPublicProfile(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "username is required", nil)
		return
	}

	actor := auth.GetActor(c)
	var viewerID *string
	if actor != nil {
		viewerID = &actor.UserID
	}

	resp, err := h.service.GetPublicProfile(c.Request.Context(), username, viewerID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	response.Success(c, http.StatusOK, resp, nil)
}

func (h *Handler) UpdateMyProfile(c *gin.Context) {
	actor := auth.GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	resp, err := h.service.UpdateMyProfile(c.Request.Context(), actor.UserID, input)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
		return
	}

	response.Success(c, http.StatusOK, resp, nil)
}
