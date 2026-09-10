package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// APIError is the error envelope returned to clients.
type APIError struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// Common errors. Domain packages wrap or return these.
var (
	ErrUnauthorized = &APIError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication required"}
	ErrForbidden    = &APIError{Status: http.StatusForbidden, Code: "forbidden", Message: "not allowed"}
	ErrNotFound     = &APIError{Status: http.StatusNotFound, Code: "not_found", Message: "resource not found"}
)

// BadRequest builds a 400 error.
func BadRequest(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

// Conflict builds a 409 error.
func Conflict(code, msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: code, Message: msg}
}

// Validation builds a 422 error with per-field messages.
func Validation(fields map[string]string) *APIError {
	return &APIError{Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: "validation failed", Fields: fields}
}

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Error writes err as a JSON error envelope. Unknown errors become 500 and are logged.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var api *APIError
	if !errors.As(err, &api) {
		slog.ErrorContext(r.Context(), "unhandled error", "err", err, "path", r.URL.Path)
		api = &APIError{Status: http.StatusInternalServerError, Code: "internal", Message: "internal server error"}
	}
	JSON(w, api.Status, map[string]any{"error": api})
}

// Decode reads a JSON body into v, rejecting unknown fields and bodies over 1MB.
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return BadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}
