package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppErrorUnwrapsCause(t *testing.T) {
	cause := errors.New("database failure")
	appError := NewAppErrorWithCause(http.StatusNotFound, "resource not found", cause)

	if !errors.Is(appError, cause) {
		t.Fatalf("expected AppError to unwrap its cause")
	}
}

func TestWriteAppErrorWritesStatusAndMessage(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	appError := NewConflictError("already exists")

	if !writeAppError(responseRecorder, appError) {
		t.Fatalf("expected AppError to be handled")
	}
	if responseRecorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, responseRecorder.Code)
	}
	if responseRecorder.Body.String() != "already exists\n" {
		t.Fatalf("expected AppError message, got %q", responseRecorder.Body.String())
	}
}

func TestWriteAppErrorRejectsUnexpectedError(t *testing.T) {
	responseRecorder := httptest.NewRecorder()

	if writeAppError(responseRecorder, errors.New("unexpected failure")) {
		t.Fatalf("expected unexpected error to be left for internal handling")
	}
}
