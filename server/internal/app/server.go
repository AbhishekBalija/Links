package app

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/mailer"
	"github.com/AbhishekBalija/Links/server/internal/profiles"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/gin-contrib/cors"
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
	if len(cfg.CORS.AllowedOrigins) > 0 {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORS.AllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}
	router.Use(
		requestBodyLimit(cfg.RequestBodyLimit),
		requestLogger(logger),
		recovery(logger),
	)
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	api := router.Group("/api")
	api.GET("/health", healthHandler)
	api.GET("/ready", readinessHandler(database))

	userRepo := auth.NewGormUserRepository(database.GORM())
	refreshRepo := auth.NewGormRefreshTokenRepository(database.GORM())
	activationRepo := auth.NewGormActivationTokenRepository(database.GORM())
	auditLogRepo := auth.NewGormAuditLogRepository(database.GORM())

	tokenCfg := auth.TokenConfig{
		AccessSecret:  cfg.Auth.JWTAccessSecret,
		RefreshSecret: cfg.Auth.JWTRefreshSecret,
		AccessTTL:     cfg.Auth.AccessTokenTTL,
		RefreshTTL:    cfg.Auth.RefreshTokenTTL,
	}

	var m mailer.Mailer
	if cfg.Mailer.ResendAPIKey == "" {
		logger.Warn("RESEND_API_KEY not set, using NoopMailer — no emails will be sent")
		m = mailer.NoopMailer{}
	} else {
		m = mailer.NewResendMailer(cfg.Mailer.ResendAPIKey, cfg.Mailer.FromEmail)
	}

	authService := auth.NewAuthService(
		userRepo,
		refreshRepo,
		activationRepo,
		auditLogRepo,
		tokenCfg,
		auth.NewArgon2PasswordHasher(),
		m,
		cfg.Mailer.FrontendURL,
	)

	policy := auth.NewPolicy()
	authHandler := auth.NewHandler(authService, policy, cfg.Cookie)
	authHandler.RegisterRoutes(api)

	v1 := api.Group("/v1")
	v1.Use(auth.RequireAuth(tokenCfg))
	v1.GET("/me", authHandler.Me)

	adminHandler := auth.NewAdminHandler(authService, policy)
	adminHandler.RegisterAdminRoutes(v1)

	profileRepo := profiles.NewGormProfileRepository(database.GORM())
	profileService := profiles.NewService(profileRepo, userRepo)
	profileHandler := profiles.NewHandler(profileService)
	profileHandler.RegisterRoutes(api, v1, tokenCfg)

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
