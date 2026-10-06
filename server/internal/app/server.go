package app

import (
	"context"
	"errors"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"log/slog"
	"net/http"
	"strings"

	"github.com/AbhishekBalija/Links/server/internal/announcements"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/dashboard"
	"github.com/AbhishekBalija/Links/server/internal/departments"
	"github.com/AbhishekBalija/Links/server/internal/directory"
	"github.com/AbhishekBalija/Links/server/internal/events"
	"github.com/AbhishekBalija/Links/server/internal/mailer"
	"github.com/AbhishekBalija/Links/server/internal/opportunities"
	"github.com/AbhishekBalija/Links/server/internal/profiles"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

// Option changes how NewServer wires the API, for tests.
type Option func(*options)

type options struct {
	mailer       mailer.Mailer
	codeSettings auth.CodeSettings
	googleCerts  *http.Client
}

// WithMailer sends email through m instead of the configured mailer.
func WithMailer(m mailer.Mailer) Option {
	return func(o *options) { o.mailer = m }
}

// WithCodeSettings replaces the email code's default limits.
func WithCodeSettings(settings auth.CodeSettings) Option {
	return func(o *options) { o.codeSettings = settings }
}

// WithGoogleCertsClient fetches Google's signing keys through client, so
// tests can sign their own tokens.
func WithGoogleCertsClient(client *http.Client) Option {
	return func(o *options) { o.googleCerts = client }
}

// NewServer builds the API router and its foundation middleware.
func NewServer(cfg config.Config, database *db.Database, logger *slog.Logger, opts ...Option) (*gin.Engine, error) {
	wiring := options{codeSettings: auth.DefaultCodeSettings()}
	for _, opt := range opts {
		opt(&wiring)
	}
	if database == nil {
		return nil, errors.New("database is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}
	for _, origin := range cfg.CORS.AllowedOrigins {
		if origin == "*" {
			return nil, errors.New("CORS_ALLOWED_ORIGINS must list explicit origins when credentials are enabled")
		}
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
		// First, so every route below gets a Sentry hub (middleware added
		// after a route is registered doesn't apply to it).
		sentrygin.New(sentrygin.Options{Repanic: false}),
		securityHeaders(cfg.AppEnv == "production"),
		requestBodyLimit(cfg.RequestBodyLimit),
		requestLogger(logger),
		clientGone(),
		recovery(logger),
	)
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	// Behind a platform that sets the client's address itself, read it from
	// that header; anywhere else the header could be forged.
	router.TrustedPlatform = cfg.ClientIPHeader

	api := router.Group("/api")
	api.GET("/health", healthHandler)
	api.GET("/ready", readinessHandler(database))
	api.POST("/csp-report", cspReportHandler(logger))

	userRepo := auth.NewGormUserRepository(database.GORM())
	refreshRepo := auth.NewGormRefreshTokenRepository(database.GORM())
	// Ending a role also withdraws or hands over the person's announcements
	// and events, in the same transaction (ADR 0028).
	authUnitOfWork := auth.NewGormAuthUnitOfWork(database.GORM(), announcements.HandOverOn, events.HandOverOn)

	tokenCfg := auth.TokenConfig{
		AccessSecret:  cfg.Auth.JWTAccessSecret,
		RefreshSecret: cfg.Auth.JWTRefreshSecret,
		AccessTTL:     cfg.Auth.AccessTokenTTL,
		RefreshTTL:    cfg.Auth.RefreshTokenTTL,
	}

	m := wiring.mailer
	switch {
	case m != nil:
	case cfg.Mailer.Provider == "smtp":
		m = mailer.NewSMTPMailer(mailer.SMTPSettings{
			Host: cfg.Mailer.SMTPHost, Port: cfg.Mailer.SMTPPort,
			Username: cfg.Mailer.SMTPUsername, Password: cfg.Mailer.SMTPPassword,
		}, cfg.Mailer.FromEmail, cfg.Mailer.FrontendURL)
	case cfg.Mailer.ResendAPIKey == "":
		logger.Warn("RESEND_API_KEY not set, using NoopMailer — no emails will be sent")
		m = mailer.NoopMailer{}
	default:
		if strings.HasSuffix(strings.ToLower(cfg.Mailer.FromEmail), "@resend.dev") {
			logger.Warn("FROM_EMAIL is a @resend.dev sender: Resend delivers only to its account owner until a domain is verified")
		}
		m = mailer.NewResendMailer(cfg.Mailer.ResendAPIKey, cfg.Mailer.FromEmail, cfg.Mailer.FrontendURL)
	}

	var googleVerifier auth.GoogleVerifier
	if cfg.Google.ClientID != "" {
		var googleOpts []idtoken.ClientOption
		if wiring.googleCerts != nil {
			googleOpts = append(googleOpts, option.WithHTTPClient(wiring.googleCerts))
		}
		verifier, err := auth.NewIDTokenVerifier(context.Background(), cfg.Google.ClientID, googleOpts...)
		if err != nil {
			return nil, err
		}
		googleVerifier = verifier
	}

	// The e2e suite reads the codes it asks for, so it can sign in the way
	// people do. Only while the test sign-in is on (local only).
	var codeRecorder *mailer.CodeRecorder
	if cfg.EnableTestSignIn {
		codeRecorder = mailer.NewCodeRecorder(m)
		m = codeRecorder
	}

	// A test copy can let the principal and admins use an email code too.
	codeSettings := wiring.codeSettings
	codeSettings.EveryRoleUsesCodes = cfg.EmailCodeForEveryRole
	if cfg.CodesPerNetwork > 0 {
		codeSettings.PerIPLimit = cfg.CodesPerNetwork
	}
	if cfg.NotOnListCodesPerDay > 0 {
		codeSettings.NotOnListDailyLimit = cfg.NotOnListCodesPerDay
	}

	authService := auth.NewAuthService(
		userRepo,
		refreshRepo,
		authUnitOfWork,
		tokenCfg,
		m,
		codeSettings,
		googleVerifier,
	)

	policy := auth.NewPolicy()
	authHandler := auth.NewHandler(authService, policy, cfg.Cookie, tokenCfg)
	// The web app's own origin (FRONTEND_URL) may use the refresh cookie
	// even when it isn't listed for CORS, as on a same-domain deployment.
	cookieOrigins := append([]string{cfg.Mailer.FrontendURL}, cfg.CORS.AllowedOrigins...)
	authHandler.RegisterRoutes(api, requireAllowedOrigin(cookieOrigins))
	if googleVerifier != nil {
		authHandler.RegisterGoogleRoutes(api, requireAllowedOrigin(cookieOrigins))
	} else {
		logger.Warn("GOOGLE_CLIENT_ID not set: Google sign-in is off")
	}
	if cfg.EnableTestSignIn {
		logger.Warn("ENABLE_TEST_SIGN_IN is on: anyone can sign in by email alone (e2e only)")
		authHandler.RegisterTestSignIn(api)
		api.GET("/v1/test/sign-in-code", lastSignInCode(codeRecorder))
	}

	v1 := api.Group("/v1")
	v1.Use(auth.RequireAuth(tokenCfg), auth.RefuseInactive(userRepo))
	v1.GET("/me", authHandler.Me)
	v1.POST("/auth/not-me", authHandler.NotMe)
	v1.POST("/me/roles/:id/welcomed", authHandler.Welcomed)

	adminHandler := auth.NewAdminHandler(authService, policy)
	adminHandler.RegisterAdminRoutes(v1)

	departmentRepo := departments.NewGormRepository(database.GORM())
	departmentUnitOfWork := departments.NewGormUnitOfWork(database.GORM())
	departmentService := departments.NewService(departmentRepo, departmentUnitOfWork)
	departmentHandler := departments.NewHandler(departmentService, policy)
	departmentHandler.RegisterRoutes(v1)
	departmentHandler.RegisterPublicRoutes(api.Group("/v1"))

	announcementRepo := announcements.NewGormRepository(database.GORM())
	announcementUnitOfWork := announcements.NewGormUnitOfWork(database.GORM())
	announcementService := announcements.NewService(announcementRepo, userRepo, announcementUnitOfWork).WithNotifier(m)
	announcements.NewHandler(announcementService, policy).RegisterRoutes(v1)

	eventService := events.NewService(events.NewGormRepository(database.GORM()), userRepo, events.NewGormUnitOfWork(database.GORM())).WithNotifier(m)
	events.NewHandler(eventService, policy).RegisterRoutes(v1)

	opportunityService := opportunities.NewService(opportunities.NewGormRepository(database.GORM()), userRepo, opportunities.NewGormUnitOfWork(database.GORM())).WithNotifier(m)
	opportunities.NewHandler(opportunityService, policy).RegisterRoutes(v1)

	directoryService := directory.NewService(directory.NewGormRepository(database.GORM()))
	directory.NewHandler(directoryService, policy).RegisterRoutes(v1)

	dashboardService := dashboard.NewService(dashboard.NewGormRepository(database.GORM()), announcementService, eventService, opportunityService, directoryService, departmentService, authService)
	dashboard.NewHandler(dashboardService, policy).RegisterRoutes(v1)

	profileRepo := profiles.NewGormProfileRepository(database.GORM())
	profileUnitOfWork := profiles.NewGormUnitOfWork(database.GORM())
	profileService := profiles.NewService(profileRepo, userRepo, directoryService, profileUnitOfWork)
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

// lastSignInCode answers GET /api/v1/test/sign-in-code?email=... with the
// last sign-in code emailed to that address, for the e2e suite.
func lastSignInCode(recorder *mailer.CodeRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, ok := recorder.LastCode(c.Query("email"))
		if !ok {
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "no code was sent to that email"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"code": code}})
	}
}
