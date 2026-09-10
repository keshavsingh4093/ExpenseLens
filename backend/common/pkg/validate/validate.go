// Package validate turns input checks into 422 API errors with a fields map.
package validate

import (
	"errors"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
)

var v = validator.New(validator.WithRequiredStructEnabled())

// Struct runs struct-tag validation and returns a 422 *gohttp.APIError on
// failure, with lowerCamelCase field names.
func Struct(in any) error {
	err := v.Struct(in)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return gohttp.ErrBadRequest(err.Error())
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		name := strings.ToLower(fe.Field()[:1]) + fe.Field()[1:]
		switch fe.Tag() {
		case "required":
			fields[name] = "is required"
		case "email":
			fields[name] = "must be a valid email"
		case "min":
			fields[name] = "must be at least " + fe.Param() + " characters"
		case "max":
			fields[name] = "must be at most " + fe.Param() + " characters"
		case "oneof":
			fields[name] = "must be one of: " + fe.Param()
		default:
			fields[name] = "is invalid (" + fe.Tag() + ")"
		}
	}
	return gohttp.ErrValidation(fields)
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

// Email reports whether s looks like an email address.
func Email(s string) bool { return len(s) <= 254 && emailRe.MatchString(strings.TrimSpace(s)) }

// Password enforces the minimum policy (8..128 chars).
func Password(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(pw) > 128 {
		return errors.New("password is too long")
	}
	return nil
}

// RequiredString returns a 422 error naming field when s is blank.
func RequiredString(field, s string) error {
	if strings.TrimSpace(s) == "" {
		return gohttp.ErrValidation(map[string]string{field: "is required"})
	}
	return nil
}

// ObjectID parses a hex id, returning gohttp.ErrNotFound when malformed
// (a bad id can never match a resource).
func ObjectID(s string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return bson.ObjectID{}, gohttp.ErrNotFound
	}
	return id, nil
}
