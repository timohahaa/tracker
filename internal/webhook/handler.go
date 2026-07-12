package webhook

import (
	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — обработчики входящих вебхуков (/webhooks/*).
type Handler struct {
	services *service.Services
}

// New создаёт роутер вебхуков.
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Post("/gitlab/{integrationID}", h.gitlab)

	return mux
}
