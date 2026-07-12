package integration

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
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

// getGitLab godoc
//
//	@Summary		Get GitLab integration status
//	@Description	Возвращает статус GitLab-интеграции workspace (участник workspace). credentials не возвращаются.
//	@Tags			integration
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	gitlabResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/integrations/gitlab [get]
func (h *Handler) getGitLab(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	gi, err := h.services.GitLab.GetGitLab(r.Context(), actor)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toGitLabResponse(gi))
}

// setupGitLab godoc
//
//	@Summary		Set up GitLab integration
//	@Description	Настраивает или переподключает GitLab-интеграцию (owner/admin workspace). Идемпотентно по (workspace, provider).
//	@Tags			integration
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			body		body		setupGitLabRequest	true	"GitLab integration payload"
//	@Success		201			{object}	gitlabResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/integrations/gitlab [post]
func (h *Handler) setupGitLab(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[setupGitLabRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	gi, err := h.services.GitLab.SetupGitLab(r.Context(), actor, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toGitLabResponse(gi))
}

// deleteGitLab godoc
//
//	@Summary		Delete GitLab integration
//	@Description	Мягко удаляет GitLab-интеграцию (owner/admin workspace).
//	@Tags			integration
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/integrations/gitlab [delete]
func (h *Handler) deleteGitLab(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	if err := h.services.GitLab.DeleteGitLab(r.Context(), actor); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}
