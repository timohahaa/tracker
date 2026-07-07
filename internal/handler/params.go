package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// QueryLimit парсит ?limit=, применяя дефолт (при отсутствии/невалидном значении)
// и кэп сверху. Значения <= 0 трактуются как «не задано» → def.
func QueryLimit(r *http.Request, def, max int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// QueryCursor читает ?cursor= и декодирует его в model.Cursor. Пустой cursor → (nil, nil).
// Битый cursor возвращается как errs.Validation.
func QueryCursor(r *http.Request) (*model.Cursor, error) {
	cur, err := DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		return nil, errs.Validation("invalid cursor", errs.FieldError{
			Field:   "cursor",
			Reason:  "invalid",
			Message: "must be a valid cursor",
		})
	}
	return cur, nil
}

// QueryString возвращает необязательный query-параметр (пустая строка при отсутствии).
func QueryString(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// RequireQuery возвращает обязательный query-параметр или errs.Validation, если он пуст.
func RequireQuery(r *http.Request, key string) (string, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return "", errs.Validation("missing required query parameter", errs.FieldError{
			Field:   key,
			Reason:  "required",
			Message: "is required",
		})
	}
	return v, nil
}

// PathParam возвращает path-параметр chi по имени.
func PathParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
