// Package http holds the response envelope, route loading and the HTTP
// server used by every backend service. Import it as gohttp.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// APIResponse is the wire envelope for every JSON response:
// {success, data, error, code, fields}.
type APIResponse struct {
	Success bool              `json:"success"`
	Data    any               `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
	Code    string            `json:"code,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// APIError is an error value that carries the HTTP status and stable code to
// report. Services return these; handlers pass them to WriteError.
type APIError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// HTTPStatus lets packages that cannot import this one (middleware) map the
// error to a response.
func (e *APIError) HTTPStatus() int { return e.Status }

// ErrorCode returns the stable machine-readable code.
func (e *APIError) ErrorCode() string { return e.Code }

// Common error values.
var (
	ErrUnauthorized = &APIError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication required"}
	ErrForbidden    = &APIError{Status: http.StatusForbidden, Code: "forbidden", Message: "not allowed"}
	ErrNotFound     = &APIError{Status: http.StatusNotFound, Code: "not_found", Message: "resource not found"}
)

// ErrBadRequest builds a 400 error value.
func ErrBadRequest(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

// ErrConflict builds a 409 error value with a stable code.
func ErrConflict(code, msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: code, Message: msg}
}

// ErrValidation builds a 422 error value with per-field messages.
func ErrValidation(fields map[string]string) *APIError {
	return &APIError{Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: "validation failed", Fields: fields}
}

// ErrWithCode builds an error value with an explicit status and code.
func ErrWithCode(status int, code, msg string) *APIError {
	return &APIError{Status: status, Code: code, Message: msg}
}

// JSON writes v with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Success sends 200 with data.
func Success(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

// Created sends 201 with data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, APIResponse{Success: true, Data: data})
}

// Accepted sends 202 with data (async work queued).
func Accepted(w http.ResponseWriter, data any) {
	JSON(w, http.StatusAccepted, APIResponse{Success: true, Data: data})
}

// NoContent sends 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Error sends an error envelope with the given status and message.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIResponse{Success: false, Error: message})
}

// ErrorWithCode sends an error envelope carrying a stable machine-readable code.
func ErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, APIResponse{Success: false, Error: message, Code: code})
}

// ValidationError sends 422 with per-field messages.
func ValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, APIResponse{Success: false, Error: "validation failed", Code: "validation_failed", Fields: fields})
}

// BadRequest sends 400.
func BadRequest(w http.ResponseWriter, message string) {
	ErrorWithCode(w, http.StatusBadRequest, "bad_request", message)
}

// Unauthorized sends 401.
func Unauthorized(w http.ResponseWriter, message string) {
	ErrorWithCode(w, http.StatusUnauthorized, "unauthorized", message)
}

// Forbidden sends 403.
func Forbidden(w http.ResponseWriter, message string) {
	ErrorWithCode(w, http.StatusForbidden, "forbidden", message)
}

// NotFound sends 404.
func NotFound(w http.ResponseWriter, message string) {
	ErrorWithCode(w, http.StatusNotFound, "not_found", message)
}

// Conflict sends 409.
func Conflict(w http.ResponseWriter, code, message string) {
	ErrorWithCode(w, http.StatusConflict, code, message)
}

// InternalError sends 500.
func InternalError(w http.ResponseWriter, message string) {
	ErrorWithCode(w, http.StatusInternalServerError, "internal", message)
}

// WriteError maps err to the envelope. *APIError values keep their status
// and code; anything else is logged and reported as a 500.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var api *APIError
	if !errors.As(err, &api) {
		slog.ErrorContext(r.Context(), "unhandled error", "err", err, "method", r.Method, "path", r.URL.Path)
		InternalError(w, "internal server error")
		return
	}
	JSON(w, api.Status, APIResponse{Success: false, Error: api.Message, Code: api.Code, Fields: api.Fields})
}

// Decode reads a JSON body into v, rejecting unknown fields and bodies over 1MB.
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return ErrBadRequest("invalid JSON body: " + err.Error())
	}
	return nil
}
