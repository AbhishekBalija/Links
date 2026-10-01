package db

import (
	"context"
	"errors"
	"fmt"
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
