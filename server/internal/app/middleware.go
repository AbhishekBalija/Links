package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

func requestBodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Header(requestIDHeader, requestID)
		c.Set("request_id", requestID)

		started := time.Now()
		c.Next()

		status := c.Writer.Status()
		logger.Info("HTTP request completed",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status_code", status,
			"latency_ms", time.Since(started).Milliseconds(),
		)
		// A server error's cause is attached by response.InternalError; the
		// client only saw "internal server error", so log it and report it.
		if status >= http.StatusInternalServerError {
			for _, failure := range c.Errors {
				logger.Error("request failed",
					"request_id", requestID,
					"method", c.Request.Method,
					"path", c.FullPath(),
					"status_code", status,
					"error", failure.Err.Error(),
				)
				if hub := sentrygin.GetHubFromContext(c); hub != nil {
					hub.Scope().SetTag("request_id", requestID)
					hub.CaptureException(failure.Err)
				}
			}
		}
	}
}

// statusClientClosedRequest is nginx's code for a request the client gave up
// on before the answer was ready. It is not a standard HTTP status; nobody
// receives it, but it keeps logs from counting these as server errors.
const statusClientClosedRequest = 499

// clientGone answers 499 instead of a 5xx when the client cancelled the
// request (a browser navigating away), because the failure is the cancelled
// database call, not the server. Real server errors on open requests keep
// their status.
func clientGone() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer = &goneWriter{ResponseWriter: c.Writer, request: c.Request}
		c.Next()
	}
}

type goneWriter struct {
	gin.ResponseWriter
	request *http.Request
	dropped bool
}

func (w *goneWriter) WriteHeader(code int) {
	if code >= http.StatusInternalServerError && errors.Is(w.request.Context().Err(), context.Canceled) {
		w.dropped = true
		code = statusClientClosedRequest
	}
	w.ResponseWriter.WriteHeader(code)
}

// The error body would go to a client that has left, so it is skipped.
func (w *goneWriter) Write(data []byte) (int, error) {
	if w.dropped {
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}

func (w *goneWriter) WriteString(data string) (int, error) {
	if w.dropped {
		return len(data), nil
	}
	return w.ResponseWriter.WriteString(data)
}

func recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic while handling request", "request_id", c.GetString("request_id"), "error", recovered)
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			hub.RecoverWithContext(c.Request.Context(), recovered)
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "internal server error"))
	})
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "fallback-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(bytes)
}
