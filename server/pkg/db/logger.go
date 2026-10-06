package db

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
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

// ParamsFilter passes through to GORM's own logger. GORM looks for this method
// on the logger it is given; embedding the Interface alone would hide it, and
// the parameter values would then be printed again.
func (l skipCancelled) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if filter, ok := l.Interface.(gorm.ParamsFilter); ok {
		return filter.ParamsFilter(ctx, sql, params...)
	}
	return sql, params
}

// newLogger builds the database logger. It keeps GORM's defaults (slow
// queries and errors) but prints "?" instead of query values, so emails,
// names, USNs and tokens never reach the logs.
func newLogger(out io.Writer) logger.Interface {
	return skipCancelled{Interface: logger.New(log.New(out, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false,
		ParameterizedQueries:      true,
		Colorful:                  true,
	})}
}

// defaultLogger writes to standard output, like GORM's default logger.
func defaultLogger() logger.Interface {
	return newLogger(os.Stdout)
}
