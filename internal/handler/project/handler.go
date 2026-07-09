package project

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики проектов.
type Handler struct {
	services *service.Services
}

// New создаёт роутер project-эндпоинтов (монтируется на /projects).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Route("/{projectID}", func(mux chi.Router) {
		mux.Get("/", h.get)
		mux.Patch("/", h.update)
		mux.Delete("/", h.delete)
	})

	return mux
}

// list godoc
//
//	@Summary		List projects
//	@Description	Возвращает проекты workspace с keyset-пагинацией (участник workspace).
//	@Tags			project
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			sort		query		string	false	"Sort field (created_at, updated_at)"	default(updated_at)
//	@Param			order		query		string	false	"Sort order (asc, desc)"	default(desc)
//	@Param			limit		query		int		false	"Page size"	default(50)
//	@Param			cursor		query		string	false	"Pagination cursor"
//	@Success		200			{object}	projectListResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/projects [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := parseListQuery(r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	ws := middleware.WorkspaceFromContext(r.Context())
	res, err := h.services.Project.ListProjects(r.Context(), ws.ID, page)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toProjectListResponse(res))
}

// create godoc
//
//	@Summary		Create a project
//	@Description	Создаёт проект и (опционально) привязывает команды (участник workspace).
//	@Tags			project
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string					true	"Workspace ID"
//	@Param			body		body		createProjectRequest	true	"Project payload"
//	@Success		201			{object}	projectResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/projects [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createProjectRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	project, err := h.services.Project.CreateProject(r.Context(), actor, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toProjectResponse(project))
}

// get godoc
//
//	@Summary		Get a project
//	@Description	Возвращает проект с привязанными командами (участник workspace).
//	@Tags			project
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			projectID	path		string	true	"Project ID"
//	@Success		200			{object}	projectResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/projects/{projectID} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	projectID, err := httpx.PathUUID(r, "projectID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	project, err := h.services.Project.GetProject(r.Context(), ws.ID, projectID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toProjectResponse(project))
}

// update godoc
//
//	@Summary		Update a project
//	@Description	Частично обновляет проект (участник workspace). team_ids — replace-семантика (null = не трогать, [] = отвязать все).
//	@Tags			project
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string					true	"Workspace ID"
//	@Param			projectID	path		string					true	"Project ID"
//	@Param			body		body		updateProjectRequest	true	"Project payload"
//	@Success		200			{object}	projectResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/projects/{projectID} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateProjectRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	projectID, err := httpx.PathUUID(r, "projectID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	project, err := h.services.Project.UpdateProject(r.Context(), actor, projectID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toProjectResponse(project))
}

// delete godoc
//
//	@Summary		Delete a project
//	@Description	Мягко удаляет проект (owner/admin workspace).
//	@Tags			project
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			projectID	path	string	true	"Project ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/projects/{projectID} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	projectID, err := httpx.PathUUID(r, "projectID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Project.DeleteProject(r.Context(), actor, projectID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}
