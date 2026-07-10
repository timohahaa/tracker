package label

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики меток.
type Handler struct {
	services *service.Services
}

// New создаёт роутер label-эндпоинтов (монтируется на /labels).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Get("/{labelID}", h.get)
	mux.Patch("/{labelID}", h.update)
	mux.Delete("/{labelID}", h.delete)

	return mux
}

// list godoc
//
//	@Summary		List labels
//	@Description	Возвращает все метки workspace (workspace- и team-level) (участник workspace).
//	@Tags			label
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	labelsResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/labels [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	labels, err := h.services.Label.ListLabels(r.Context(), ws.ID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toLabelsResponse(labels))
}

// create godoc
//
//	@Summary		Create a label
//	@Description	Создаёт метку (участник workspace). team_id=null → workspace-level, иначе team-level. При создании ws-метки поверх одноимённых team-меток выполняется merge.
//	@Tags			label
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			body		body		createLabelRequest	true	"Label payload"
//	@Success		201			{object}	labelResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/labels [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createLabelRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	label, err := h.services.Label.CreateLabel(r.Context(), actor, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toLabelResponse(label))
}

// get godoc
//
//	@Summary		Get a label
//	@Description	Возвращает метку workspace по id (участник workspace).
//	@Tags			label
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			labelID		path		string	true	"Label ID"
//	@Success		200			{object}	labelResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/labels/{labelID} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	labelID, err := httpx.PathUUID(r, "labelID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	label, err := h.services.Label.GetLabel(r.Context(), ws.ID, labelID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toLabelResponse(label))
}

// update godoc
//
//	@Summary		Update a label
//	@Description	Частично обновляет name/color метки (участник workspace).
//	@Tags			label
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			labelID		path		string				true	"Label ID"
//	@Param			body		body		updateLabelRequest	true	"Label payload"
//	@Success		200			{object}	labelResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/labels/{labelID} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateLabelRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	labelID, err := httpx.PathUUID(r, "labelID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	label, err := h.services.Label.UpdateLabel(r.Context(), actor, labelID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toLabelResponse(label))
}

// delete godoc
//
//	@Summary		Delete a label
//	@Description	Мягко удаляет метку (участник workspace).
//	@Tags			label
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			labelID		path	string	true	"Label ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/labels/{labelID} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	labelID, err := httpx.PathUUID(r, "labelID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Label.DeleteLabel(r.Context(), actor, labelID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}
