package workspace

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики workspace (/api/workspaces/{workspaceID}).
type Handler struct {
	services *service.Services
}

// New создаёт роутер workspace-эндпоинтов.
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.get)
	mux.Patch("/", h.update)
	mux.Get("/members", h.listMembers)
	mux.Patch("/members/{userID}", h.updateMember)
	mux.Delete("/members/{userID}", h.deleteMember)
	mux.Post("/invites", h.createInvite)

	return mux
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request)          {}
func (h *Handler) update(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) deleteMember(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {}
