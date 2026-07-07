package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики аутентификации (/api/auth/*).
type Handler struct {
	services *service.Services
}

// New создаёт роутер auth-эндпоинтов.
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Post("/register", h.register)
	mux.Post("/login", h.login)
	mux.Post("/logout", h.logout)
	mux.Post("/accept-invite", h.acceptInvite)

	return mux
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request)     {}
func (h *Handler) login(w http.ResponseWriter, r *http.Request)        {}
func (h *Handler) logout(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {}
