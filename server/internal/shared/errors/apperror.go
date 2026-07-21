package errors

import "net/http"

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Details    interface{}
}

func (e *AppError) Error() string {
	return e.Message
}

func NewValidation(message string, details interface{}) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

func NewUnauthenticated(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHENTICATED",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

func NewForbidden(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

func NewNotFound(message string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    message,
		HTTPStatus: http.StatusNotFound,
	}
}

func NewConflict(message string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}
}

func NewInternal(message string) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

func NewRateLimited(message string) *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    message,
		HTTPStatus: http.StatusTooManyRequests,
	}
}