package token

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики API-токенов.
type Handler struct {
	services *service.Services
}

// New создаёт роутер token-эндпоинтов (монтируется на /tokens).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Delete("/{tokenID}", h.revoke)

	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request)   {}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {}
