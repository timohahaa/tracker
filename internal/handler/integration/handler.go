package integration

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики интеграций (GitLab).
type Handler struct {
	services *service.Services
}

// New создаёт роутер integration-эндпоинтов (монтируется на /integrations).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/gitlab", h.getGitLab)
	mux.Post("/gitlab", h.setupGitLab)
	mux.Delete("/gitlab", h.deleteGitLab)

	return mux
}

func (h *Handler) getGitLab(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) setupGitLab(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) deleteGitLab(w http.ResponseWriter, r *http.Request) {}
