package app

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/gin-gonic/gin"
)

// NewServer builds the API router and its foundation middleware.
func NewServer(cfg config.Config, database *db.Database, logger *slog.Logger) (*gin.Engine, error) {
	if database == nil {
		return nil, errors.New("database is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	gin.SetMode(cfg.GINMode)
	router := gin.New()
	router.Use(requestBodyLimit(cfg.RequestBodyLimit), requestLogger(logger), recovery(logger))
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	api := router.Group("/api")
	api.GET("/health", healthHandler)
	api.GET("/ready", readinessHandler(database))

	userRepo := auth.NewGormUserRepository(database.GORM())
	refreshRepo := auth.NewGormRefreshTokenRepository(database.GORM())
	activationRepo := auth.NewGormActivationTokenRepository(database.GORM())

	authService := auth.NewAuthService(
		userRepo,
		refreshRepo,
		activationRepo,
		auth.TokenConfig{
			AccessSecret:  cfg.Auth.JWTAccessSecret,
			RefreshSecret: cfg.Auth.JWTRefreshSecret,
			AccessTTL:     cfg.Auth.AccessTokenTTL,
			RefreshTTL:    cfg.Auth.RefreshTokenTTL,
		},
		auth.NewArgon2PasswordHasher(),
	)

	authHandler := auth.NewHandler(authService, cfg.Cookie)
	authHandler.RegisterRoutes(api)

	return router, nil
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "links-api",
		"status":  "ok",
	})
}

func readinessHandler(database *db.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := database.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, errorResponse("INTERNAL_ERROR", "database unavailable"))
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"database": "connected",
			"status":   "ok",
		})
	}
}
