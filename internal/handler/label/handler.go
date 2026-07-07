package label

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики меток.
type Handler struct {
	services *service.Services
}

// New создаёт роутер label-эндпоинтов (монтируется на /labels).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Patch("/{labelID}", h.update)
	mux.Delete("/{labelID}", h.delete)

	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request)   {}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {}
