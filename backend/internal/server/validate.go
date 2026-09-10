package server

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate runs struct tag validation and converts failures to a 422 APIError.
func Validate(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return BadRequest(err.Error())
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
	return Validation(fields)
}
