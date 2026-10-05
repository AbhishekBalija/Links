package auth

import (
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/mail"
	"strings"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/gin-gonic/gin"
)

const (
	refreshCookieName = "refresh_token"
	nonceCookieName   = "google_nonce"
	// nonceCookiePath limits the nonce cookie to the Google sign-in routes.
	nonceCookiePath = "/api/v1/auth/google"
	nonceTTL        = 10 * time.Minute
)

type Handler struct {
	service   AuthService
	policy    *Policy
	cookieCfg config.CookieConfig
	tokenCfg  TokenConfig
}

func NewHandler(service AuthService, policy *Policy, cookieCfg config.CookieConfig, tokenCfg TokenConfig) *Handler {
	return &Handler{service: service, policy: policy, cookieCfg: cookieCfg, tokenCfg: tokenCfg}
}

// RegisterRoutes adds the auth routes. cookieGuard runs before the routes
// that act on the refresh cookie, to refuse cross-site requests.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, cookieGuard gin.HandlerFunc) {
	v1 := rg.Group("/v1/auth")
	// Signing in sets the refresh cookie, so a cross-site page mustn't be
	// able to sign a browser into someone else's account (login CSRF).
	v1.POST("/code", cookieGuard, h.RequestCode)
	v1.POST("/code/verify", cookieGuard, h.VerifyCode)
	v1.POST("/access-request", h.RequestAccessWithProof)
	v1.POST("/refresh", cookieGuard, h.Refresh)
	v1.POST("/logout", cookieGuard, h.Logout)
}

func (h *Handler) RequestCode(c *gin.Context) {
	var input RequestCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	email := strings.TrimSpace(input.Email)
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "enter a valid email address", nil)
		return
	}

	device, _ := c.Cookie(DeviceCookieName)
	challengeID, err := h.service.RequestCode(c.Request.Context(), email, c.ClientIP(), device)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, RequestCodeResponse{ChallengeID: challengeID, Message: CodeSentMessage}, nil)
}

func (h *Handler) VerifyCode(c *gin.Context) {
	var input VerifyCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	device, _ := c.Cookie(DeviceCookieName)
	signedIn, err := h.service.VerifyCode(c.Request.Context(), input.ChallengeID, input.Email, input.Code, device)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setRefreshCookie(c, signedIn.Refresh)
	// Kept when the person signs out: it is what lets this browser get codes
	// whatever others on its network do.
	h.setCookie(c, DeviceCookieName, signedIn.Device, DeviceMaxAge, DeviceCookiePath)
	response.Success(c, http.StatusOK, signedIn.Login, nil)
}

// RegisterGoogleRoutes adds Google sign-in. The server registers it only
// when GOOGLE_CLIENT_ID is set.
func (h *Handler) RegisterGoogleRoutes(rg *gin.RouterGroup, cookieGuard gin.HandlerFunc) {
	v1 := rg.Group("/v1/auth/google")
	v1.GET("/nonce", h.GoogleNonce)
	v1.POST("", cookieGuard, h.GoogleSignIn)
}

// GoogleNonce gives this browser a nonce for Google Identity Services and
// keeps it in an httpOnly cookie, so only a token obtained in this browser
// signs it in (login CSRF).
func (h *Handler) GoogleNonce(c *gin.Context) {
	nonce, err := NewGoogleNonce()
	if err != nil {
		writeError(c, err)
		return
	}
	h.setCookie(c, nonceCookieName, nonce, int(nonceTTL.Seconds()), nonceCookiePath)
	c.Header("Cache-Control", "no-store")
	response.Success(c, http.StatusOK, GoogleNonceResponse{Nonce: nonce}, nil)
}

func (h *Handler) GoogleSignIn(c *gin.Context) {
	var input GoogleSignInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	nonce, _ := c.Cookie(nonceCookieName)
	// A nonce is good for one try, whatever its outcome.
	h.setCookie(c, nonceCookieName, "", -1, nonceCookiePath)

	resp, refreshRaw, err := h.service.SignInWithGoogle(c.Request.Context(), input.Credential, nonce)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setRefreshCookie(c, refreshRaw)
	response.Success(c, http.StatusOK, resp, nil)
}

// RequestAccessWithProof sends an Access request for an email proven by a
// request token from NOT_ON_LIST.
func (h *Handler) RequestAccessWithProof(c *gin.Context) {
	var input ProvenAccessRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	resp, err := h.service.RequestAccessWithProof(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, resp, nil)
}

// NotMe is "Not you?" after a first sign-in.
func (h *Handler) NotMe(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}
	if err := h.service.NotMe(c.Request.Context(), actor.UserID); err != nil {
		writeError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.Success(c, http.StatusOK, LogoutResponse{Message: "signed out; the account waits for an admin to fix it"}, nil)
}

// Welcomed records that the signed-in user closed the welcome to a new role.
func (h *Handler) Welcomed(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "role not found", nil)
		return
	}
	if err := h.service.Welcomed(c.Request.Context(), actor.UserID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"welcomed": true}, nil)
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
	h.setCookie(c, refreshCookieName, token, int(h.tokenCfg.RefreshTTL.Seconds()), "/")
}

// setCookie sets an httpOnly cookie with the configured SameSite and Secure.
func (h *Handler) setCookie(c *gin.Context, name, value string, maxAge int, path string) {
	// Config only allows lax or strict (see config.Validate).
	sameSite := http.SameSiteLaxMode
	if h.cookieCfg.SameSite == "strict" {
		sameSite = http.SameSiteStrictMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(name, value, maxAge, path, "", h.cookieCfg.Secure, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetCookie(refreshCookieName, "", -1, "/", "", h.cookieCfg.Secure, true)
}

func (h *Handler) Me(c *gin.Context) {
	actor := GetActor(c)
	if actor == nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "not authenticated", nil)
		return
	}

	resp, err := h.service.GetMe(c.Request.Context(), actor.UserID)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp, nil)
}

func writeError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	response.InternalError(c, err)
}

type testSignInInput struct {
	Email string `json:"email" binding:"required,email"`
}

// RegisterTestSignIn adds POST /api/v1/test/sign-in, which signs a member in
// by email alone. The server registers it only when the config allows it.
func (h *Handler) RegisterTestSignIn(rg *gin.RouterGroup) {
	rg.POST("/v1/test/sign-in", func(c *gin.Context) {
		var input testSignInInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
			return
		}
		resp, refreshRaw, err := h.service.TestSignIn(c.Request.Context(), input.Email)
		if err != nil {
			writeError(c, err)
			return
		}
		h.setRefreshCookie(c, refreshRaw)
		response.Success(c, http.StatusOK, resp, nil)
	})
}
