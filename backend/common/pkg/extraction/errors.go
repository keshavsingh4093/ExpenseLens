package extraction

import (
	"errors"
	"fmt"
)

// Error codes describe why extraction failed in terms a user can act on. The
// raw provider error stays in the server log and the admin portal; end users
// only ever see the code.
const (
	CodeRateLimited = "rate_limited" // provider quota or rate limit hit; retry later
	CodeUnavailable = "unavailable"  // provider down, network error, timeout
	CodeBlocked     = "blocked"      // provider refused the content
	CodeBadResponse = "bad_response" // provider answered with something we could not parse
	CodeFailed      = "failed"       // anything else (bad credentials, invalid request, ...)
)

// Error is a classified extraction failure. Message carries provider detail
// for logs and admins; Code is what the user-facing API exposes.
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string { return "extraction: " + e.Message }

// Unwrap exposes the underlying error to errors.Is/As.
func (e *Error) Unwrap() error { return e.Cause }

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// CodeOf returns the classification of err, CodeFailed when it is not an
// *Error and CodeUnavailable for context deadlines/cancellations.
func CodeOf(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	if isTimeout(err) {
		return CodeUnavailable
	}
	return CodeFailed
}

// codeForStatus maps a provider HTTP status to an error code.
func codeForStatus(status int) string {
	switch {
	case status == 429:
		return CodeRateLimited
	case status >= 500, status == 408:
		return CodeUnavailable
	}
	return CodeFailed
}
