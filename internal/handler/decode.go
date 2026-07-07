package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/timohahaa/tracker/internal/errs"
)

// validate — общий переиспользуемый инстанс валидатора. Имена полей в ошибках
// берутся из json-тегов, чтобы FieldError соответствовал телу запроса.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

// DecodeJSON декодирует JSON-тело запроса в T и валидирует его тегами validate.
// Ошибки парсинга и валидации возвращаются как errs.AppError с кодом Validation
// (с заполненными FieldError по тегам validate) — маппятся на HTTP через RespondError.
func DecodeJSON[T any](r *http.Request) (T, error) {
	var dst T

	if r.Body == nil {
		return dst, errs.Validation("request body is required")
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&dst); err != nil {
		return dst, decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return dst, errs.Validation("request body must contain a single JSON object")
	}

	if err := validate.Struct(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			return dst, errs.Validation("validation failed", validationFields(verrs)...)
		}
		return dst, errs.Validation("validation failed")
	}

	return dst, nil
}

// decodeError превращает ошибку json.Decode в человекочитаемую Validation-ошибку.
func decodeError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return errs.Validation("invalid request body", errs.FieldError{
			Field:   typeErr.Field,
			Reason:  "type",
			Message: fmt.Sprintf("must be of type %s", typeErr.Type.String()),
		})
	}
	if errors.Is(err, io.EOF) {
		return errs.Validation("request body is required")
	}
	return errs.Validation("invalid request body")
}

// validationFields конвертирует ошибки валидатора в доменные FieldError.
func validationFields(verrs validator.ValidationErrors) []errs.FieldError {
	fields := make([]errs.FieldError, 0, len(verrs))
	for _, fe := range verrs {
		field := fe.Field()
		if field == "" {
			field = fe.StructField()
		}
		fields = append(fields, errs.FieldError{
			Field:   field,
			Reason:  fe.Tag(),
			Message: validationMessage(fe),
		})
	}
	return fields
}

// validationMessage формирует читаемое сообщение по правилу валидации.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s in length", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	default:
		if fe.Param() != "" {
			return fmt.Sprintf("failed %s=%s", fe.Tag(), fe.Param())
		}
		return fmt.Sprintf("failed %s", fe.Tag())
	}
}
