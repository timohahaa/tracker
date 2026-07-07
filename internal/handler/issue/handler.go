package issue

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики задач.
type Handler struct {
	services *service.Services
}

// New создаёт роутер issue-эндпоинтов (монтируется на /issues).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Route("/{issueID}", func(mux chi.Router) {
		mux.Get("/", h.get)
		mux.Patch("/", h.update)
		mux.Delete("/", h.delete)

		mux.Post("/assignees", h.addAssignee)
		mux.Delete("/assignees/{userID}", h.removeAssignee)
		mux.Put("/assignees/{userID}/primary", h.setPrimary)
	})

	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request)           {}
func (h *Handler) create(w http.ResponseWriter, r *http.Request)         {}
func (h *Handler) get(w http.ResponseWriter, r *http.Request)            {}
func (h *Handler) update(w http.ResponseWriter, r *http.Request)         {}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request)         {}
func (h *Handler) addAssignee(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) removeAssignee(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) setPrimary(w http.ResponseWriter, r *http.Request)     {}
