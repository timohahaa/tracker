package issue

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
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

// create godoc
//
//	@Summary		Create an issue
//	@Description	Создаёт задачу и выдаёт identifier (участник команды). status_id по умолчанию — дефолтный статус backlog.
//	@Tags			issue
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			body		body		createIssueRequest	true	"Issue payload"
//	@Success		201			{object}	issueResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/issues [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createIssueRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	params, err := req.toParams()
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	issue, err := h.services.Issue.CreateIssue(r.Context(), actor, params)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toIssueResponse(issue))
}

// get godoc
//
//	@Summary		Get an issue
//	@Description	Возвращает задачу по UUID или identifier (ENG-123) (участник workspace).
//	@Tags			issue
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			issueID		path		string	true	"Issue UUID or identifier"
//	@Success		200			{object}	issueResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/issues/{issueID} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	ref := httpx.PathParam(r, "issueID")
	issue, err := h.services.Issue.GetIssue(r.Context(), ws.ID, ref)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toIssueResponse(issue))
}

// update godoc
//
//	@Summary		Update an issue
//	@Description	Частично обновляет задачу (участник команды).
//	@Tags			issue
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			issueID		path		string				true	"Issue UUID or identifier"
//	@Param			body		body		updateIssueRequest	true	"Issue payload"
//	@Success		200			{object}	issueResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/issues/{issueID} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateIssueRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	params, err := req.toParams()
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	ref := httpx.PathParam(r, "issueID")
	issue, err := h.services.Issue.UpdateIssue(r.Context(), actor, ref, params)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toIssueResponse(issue))
}

// delete godoc
//
//	@Summary		Delete an issue
//	@Description	Мягко удаляет задачу (участник команды).
//	@Tags			issue
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			issueID		path	string	true	"Issue UUID or identifier"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/issues/{issueID} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	ref := httpx.PathParam(r, "issueID")
	if err := h.services.Issue.DeleteIssue(r.Context(), actor, ref); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}

// list, addAssignee, removeAssignee, setPrimary — заглушки следующих заходов
// (List/фильтры и assignees).
func (h *Handler) list(w http.ResponseWriter, r *http.Request)           {}
func (h *Handler) addAssignee(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) removeAssignee(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) setPrimary(w http.ResponseWriter, r *http.Request)     {}
