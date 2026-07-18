package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/app"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
)

func main() {
	logger := newLogger()
	if err := run(logger); err != nil {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.GetEnv("SENTRY_DSN", ""),
		Environment:      config.GetEnv("APP_ENV", "development"),
		EnableTracing:    true,
		TracesSampleRate: 1.0,
	}); err != nil {
		logger.Warn("sentry.Init failed", "error", err)
	}
	defer sentry.Flush(2 * time.Second)

	database, err := db.New(cfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer database.Close()

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelStartup()
	if err := database.Migrate(startupContext, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	handler, err := app.NewServer(cfg, database, logger)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	handler.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("API started", "port", cfg.Port, "environment", cfg.AppEnv)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-shutdownContext.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}