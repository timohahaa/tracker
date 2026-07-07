package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики команд, участников и статусов.
type Handler struct {
	services *service.Services
}

// New создаёт роутер team-эндпоинтов (монтируется на /teams).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Route("/{teamID}", func(mux chi.Router) {
		mux.Get("/", h.get)
		mux.Patch("/", h.update)
		mux.Delete("/", h.delete)

		mux.Get("/members", h.listMembers)
		mux.Post("/members", h.addMember)
		mux.Delete("/members/{userID}", h.removeMember)

		mux.Get("/statuses", h.listStatuses)
		mux.Post("/statuses", h.createStatus)
		mux.Patch("/statuses/{statusID}", h.updateStatus)
		mux.Delete("/statuses/{statusID}", h.deleteStatus)
	})

	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request)         {}
func (h *Handler) create(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) get(w http.ResponseWriter, r *http.Request)          {}
func (h *Handler) update(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request)       {}
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request)  {}
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) listStatuses(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) createStatus(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) deleteStatus(w http.ResponseWriter, r *http.Request) {}
