package auth

import (
	"errors"
	"net/http"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/gin-gonic/gin"
)

const refreshCookieName = "refresh_token"

type Handler struct {
	service   AuthService
	cookieCfg config.CookieConfig
}

func NewHandler(service AuthService, cookieCfg config.CookieConfig) *Handler {
	return &Handler{service: service, cookieCfg: cookieCfg}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1/auth")
	v1.POST("/request-access", h.RequestAccess)
	v1.POST("/login", h.Login)
	v1.POST("/refresh", h.Refresh)
	v1.POST("/logout", h.Logout)
}

func (h *Handler) RequestAccess(c *gin.Context) {
	var input RequestAccessInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	resp, err := h.service.RequestAccess(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, resp, nil)
}

func (h *Handler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	resp, refreshRaw, err := h.service.Login(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setRefreshCookie(c, refreshRaw)
	response.Success(c, http.StatusOK, resp, nil)
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshRaw, err := c.Cookie(refreshCookieName)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token not provided", nil)
		return
	}

	resp, newRefreshRaw, err := h.service.Refresh(c.Request.Context(), refreshRaw)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setRefreshCookie(c, newRefreshRaw)
	response.Success(c, http.StatusOK, resp, nil)
}

func (h *Handler) Logout(c *gin.Context) {
	refreshRaw, err := c.Cookie(refreshCookieName)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token not provided", nil)
		return
	}

	if err := h.service.Logout(c.Request.Context(), refreshRaw); err != nil {
		writeError(c, err)
		return
	}

	h.clearRefreshCookie(c)
	response.Success(c, http.StatusOK, LogoutResponse{Message: "logged out successfully"}, nil)
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode
	if h.cookieCfg.SameSite == "strict" {
		sameSite = http.SameSiteStrictMode
	} else if h.cookieCfg.SameSite == "none" {
		sameSite = http.SameSiteNoneMode
	}

	c.SetCookie(refreshCookieName, token,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		h.cookieCfg.Secure,
		true,
	)
	c.SetSameSite(sameSite)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetCookie(refreshCookieName, "", -1, "/", "", h.cookieCfg.Secure, true)
}

func writeError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
}
