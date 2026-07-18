package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatalf("Environment initialization failed: %v", err)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.GetEnv("SENTRY_DSN", ""),
		Environment:      config.GetEnv("APP_ENV", "development"),
		EnableTracing:    true,
		TracesSampleRate: 1.0, // drop to 0.1–0.2 once traffic grows
	}); err != nil {
		log.Printf("sentry.Init failed: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	dsn := config.GetDatabaseDSN()
	if err := db.InitDB(dsn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	r := gin.Default()
	r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("Proxy configuration failed: %v", err)
	}

	api := r.Group("/api")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "links-api",
		})
	})

	api.GET("/health/db", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error":  "database unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	if err := r.Run(":" + config.GetPort()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
