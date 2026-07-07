package httpx

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/timohahaa/tracker/internal/errs"
)

type sampleBody struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required"`
}

func TestDecodeJSONSuccess(t *testing.T) {
	body := `{"email":"user@test.local","name":"Alice"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))

	got, err := DecodeJSON[sampleBody](r)
	if err != nil {
		t.Fatalf("DecodeJSON: unexpected error: %v", err)
	}
	if got.Email != "user@test.local" || got.Name != "Alice" {
		t.Errorf("decoded = %+v, want email=user@test.local name=Alice", got)
	}
}

func TestDecodeJSONValidationFailure(t *testing.T) {
	body := `{"email":"not-an-email","name":""}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))

	_, err := DecodeJSON[sampleBody](r)
	if err == nil {
		t.Fatal("DecodeJSON: expected validation error, got nil")
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is not *errs.AppError: %T", err)
	}
	if appErr.Code != errs.CodeValidation {
		t.Errorf("code = %q, want %q", appErr.Code, errs.CodeValidation)
	}
	if len(appErr.Fields) == 0 {
		t.Fatal("expected populated Fields, got none")
	}

	seen := map[string]bool{}
	for _, f := range appErr.Fields {
		seen[f.Field] = true
	}
	if !seen["email"] {
		t.Errorf("expected FieldError for %q; got fields %+v", "email", appErr.Fields)
	}
	if !seen["name"] {
		t.Errorf("expected FieldError for %q; got fields %+v", "name", appErr.Fields)
	}
}

func TestDecodeJSONInvalidBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("{bad json"))

	_, err := DecodeJSON[sampleBody](r)
	if err == nil {
		t.Fatal("DecodeJSON: expected error for malformed body, got nil")
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeValidation {
		t.Fatalf("want Validation AppError, got %v", err)
	}
}
