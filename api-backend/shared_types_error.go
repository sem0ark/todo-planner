package main

import (
	"errors"
	"net/http"
)

// AppError describes an expected application failure and its HTTP response.
// Cause is retained for errors.Is and internal logging.
type AppError struct {
	StatusCode int
	Message    string
	Cause      error
}

func (appError *AppError) Error() string {
	return appError.Message
}

func (appError *AppError) Unwrap() error {
	return appError.Cause
}

func NewAppError(statusCode int, message string) *AppError {
	return &AppError{StatusCode: statusCode, Message: message}
}

func NewAppErrorWithCause(statusCode int, message string, cause error) *AppError {
	return &AppError{StatusCode: statusCode, Message: message, Cause: cause}
}

func NewBadRequestError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message)
}

func NewNotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, message)
}

func NewConflictError(message string) *AppError {
	return NewAppError(http.StatusConflict, message)
}

func writeAppError(responseWriter http.ResponseWriter, err error) bool {
	var appError *AppError
	if !errors.As(err, &appError) {
		return false
	}

	http.Error(responseWriter, appError.Message, appError.StatusCode)
	return true
}
