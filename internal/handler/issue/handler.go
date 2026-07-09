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

// list godoc
//
//	@Summary		List issues
//	@Description	Возвращает задачи workspace с фильтрами, поиском и keyset-пагинацией (участник workspace).
//	@Tags			issue
//	@Produce		json
//	@Param			workspaceID		path		string	true	"Workspace ID"
//	@Param			team			query		string	false	"Team ID"
//	@Param			status			query		string	false	"Status ID"
//	@Param			status_group	query		string	false	"Status group (backlog, planned, in_progress, completed, canceled)"
//	@Param			assignee		query		string	false	"Assignee user ID (any participant)"
//	@Param			responsible		query		string	false	"Responsible user ID"
//	@Param			priority		query		string	false	"Priority (none, urgent, high, medium, low)"
//	@Param			project			query		string	false	"Project ID"
//	@Param			label			query		[]string	false	"Label ID (repeatable, matches all)"	collectionFormat(multi)
//	@Param			due_date_before	query		string	false	"Due date on or before (YYYY-MM-DD)"
//	@Param			due_date_after	query		string	false	"Due date on or after (YYYY-MM-DD)"
//	@Param			q				query		string	false	"Full-text search (title + description)"
//	@Param			sort			query		string	false	"Sort field (created_at, updated_at, priority)"	default(updated_at)
//	@Param			order			query		string	false	"Sort order (asc, desc)"	default(desc)
//	@Param			limit			query		int		false	"Page size"	default(50)
//	@Param			cursor			query		string	false	"Pagination cursor"
//	@Success		200				{object}	issueListResponse
//	@Failure		400				{object}	httpx.ErrorResponse
//	@Failure		401				{object}	httpx.ErrorResponse
//	@Failure		404				{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/issues [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	filter, page, err := parseListQuery(r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	ws := middleware.WorkspaceFromContext(r.Context())
	res, err := h.services.Issue.ListIssues(r.Context(), ws.ID, filter, page)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toIssueListResponse(res))
}

// addAssignee, removeAssignee, setPrimary — заглушки следующего захода (assignees).
func (h *Handler) addAssignee(w http.ResponseWriter, r *http.Request)    {}
func (h *Handler) removeAssignee(w http.ResponseWriter, r *http.Request) {}
func (h *Handler) setPrimary(w http.ResponseWriter, r *http.Request)     {}
