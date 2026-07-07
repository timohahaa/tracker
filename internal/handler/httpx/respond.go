package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/timohahaa/tracker/internal/errs"
)

// ErrorResponse — конверт ошибки API: {"error": {...}}.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody — тело доменной ошибки в HTTP-ответе.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Fields  []errs.FieldError `json:"fields,omitempty"`
}

// RespondJSON пишет успешный JSON-ответ с указанным статусом.
func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("respond: encode: %v", err)
	}
}

// RespondError маппит доменную ошибку в HTTP-ответ.
func RespondError(w http.ResponseWriter, err error) {
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		log.Errorf("respond: unrecognized error: %v", err)
		RespondJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{Code: string(errs.CodeInternal), Message: "internal error"},
		})
		return
	}
	RespondJSON(w, mapHTTPStatus(appErr.Code), ErrorResponse{
		Error: ErrorBody{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Detail:  appErr.Detail,
			Fields:  appErr.Fields,
		},
	})
}

func mapHTTPStatus(code errs.Code) int {
	switch code {
	case errs.CodeNotFound:
		return http.StatusNotFound
	case errs.CodeValidation:
		return http.StatusBadRequest
	case errs.CodeAlreadyExists:
		return http.StatusConflict
	case errs.CodeForbidden:
		return http.StatusForbidden
	case errs.CodeUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
