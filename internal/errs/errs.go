package errs

import "fmt"

// Code — машиночитаемый код доменной ошибки.
type Code string

const (
	CodeNotFound      Code = "not_found"
	CodeForbidden     Code = "forbidden"
	CodeUnauthorized  Code = "unauthorized"
	CodeValidation    Code = "validation"
	CodeAlreadyExists Code = "already_exists"
	CodeInternal      Code = "internal"
)

// AppError — доменная ошибка. Возвращается из service layer, не знает про транспорт.
type AppError struct {
	Code    Code
	Message string
	Detail  string
	Fields  []FieldError
}

// FieldError — ошибка валидации конкретного поля.
type FieldError struct {
	Field   string `json:"field"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg}
}

func Validation(msg string, fields ...FieldError) *AppError {
	return &AppError{Code: CodeValidation, Message: msg, Fields: fields}
}

func AlreadyExists(msg string) *AppError {
	return &AppError{Code: CodeAlreadyExists, Message: msg}
}

func Internal(msg string) *AppError {
	return &AppError{Code: CodeInternal, Message: msg}
}
