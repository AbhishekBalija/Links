package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm/logger"
)

// skipCancelled wraps GORM's logger so a query cut short because the client
// cancelled the request (a browser navigating away) is not printed as a
// database error. Every other query is logged as before.
type skipCancelled struct {
	logger.Interface
}

func (l skipCancelled) LogMode(level logger.LogLevel) logger.Interface {
	return skipCancelled{Interface: l.Interface.LogMode(level)}
}

func (l skipCancelled) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	l.Interface.Trace(ctx, begin, fc, err)
}
