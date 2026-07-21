package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
	"github.com/AbhishekBalija/Links/server/internal/shared/response"
)

const actorKey = "actor"

type Actor struct {
	UserID string
	Roles  []string
}

func GetActor(c *gin.Context) *Actor {
	a, _ := c.Get(actorKey)
	actor, _ := a.(*Actor)
	return actor
}

func RequireAuth(cfg TokenConfig) gin.HandlerFunc {
	return authMiddleware(cfg, true)
}

func OptionalAuth(cfg TokenConfig) gin.HandlerFunc {
	return authMiddleware(cfg, false)
}

func authMiddleware(cfg TokenConfig, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractBearerToken(c)
		if err != nil {
			if required {
				response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authorization token is required", nil)
				c.Abort()
			}
			return
		}

		claims, err := ValidateAccessToken(token, cfg)
		if err != nil {
			if required {
				response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid or expired token", nil)
				c.Abort()
			}
			return
		}

		c.Set(actorKey, &Actor{
			UserID: claims.UserID,
			Roles:  claims.Roles,
		})
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) (string, error) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return "", apperrors.NewUnauthenticated("authorization header missing")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", apperrors.NewUnauthenticated("invalid authorization header format")
	}
	return parts[1], nil
}
