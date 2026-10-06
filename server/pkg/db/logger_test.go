package db

import (
	"bytes"
	"strings"

	"context"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// recording is a GORM logger that remembers which queries it was asked to trace.
type recording struct {
	logger.Interface
	traced []error
}

func (r *recording) Trace(_ context.Context, _ time.Time, _ func() (string, int64), err error) {
	r.traced = append(r.traced, err)
}

func TestQueriesCancelledByTheClientAreNotLoggedAsErrors(t *testing.T) {
	t.Parallel()
	inner := &recording{Interface: logger.Discard}
	quiet := skipCancelled{Interface: inner}
	sql := func() (string, int64) { return "SELECT 1", 0 }

	quiet.Trace(context.Background(), time.Now(), sql, context.Canceled)
	quiet.Trace(context.Background(), time.Now(), sql, fmt.Errorf("query: %w", context.Canceled))
	failure := errors.New("relation does not exist")
	quiet.Trace(context.Background(), time.Now(), sql, failure)
	quiet.Trace(context.Background(), time.Now(), sql, nil)

	if len(inner.traced) != 2 || inner.traced[0] != failure || inner.traced[1] != nil {
		t.Fatalf("traced = %v, want the real failure and the successful query only", inner.traced)
	}
}

func TestSQLLogsDoNotContainQueryValues(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	// Debug-only: raise the level so every statement is traced, as a slow query would be.
	gormDB, err := gorm.Open(
		postgres.New(postgres.Config{DSN: "postgres://nobody@localhost:1/none"}),
		&gorm.Config{Logger: newLogger(&out).LogMode(logger.Info), DryRun: true, DisableAutomaticPing: true},
	)
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}

	const email = "student.one@example.edu"
	var rows []map[string]any
	gormDB.Table("users").Where("email = ?", email).Find(&rows)

	logged := out.String()
	if !strings.Contains(logged, "users") {
		t.Fatalf("expected the query to be logged, got %q", logged)
	}
	if strings.Contains(logged, email) {
		t.Fatalf("log contains a query value: %q", logged)
	}
}
