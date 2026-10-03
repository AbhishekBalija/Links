package response

import (
	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Data interface{} `json:"data,omitempty"`
	Meta interface{} `json:"meta,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func Success(c *gin.Context, status int, data interface{}, meta interface{}) {
	c.JSON(status, Envelope{Data: data, Meta: meta})
}

func Error(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, ErrorEnvelope{Error: ErrorDetail{Code: code, Message: message, Details: details}})
}

// InternalError answers a plain 500 that hides the cause from the client,
// and attaches the cause to the request so the request logger logs it and
// reports it to Sentry.
func InternalError(c *gin.Context, err error) {
	_ = c.Error(err)
	Error(c, 500, "INTERNAL_ERROR", "internal server error", nil)
}
