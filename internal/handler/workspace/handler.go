package workspace

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики workspace (/api/workspaces/{workspaceID}).
type Handler struct {
	services *service.Services
}

// NewHandler создаёт обработчик workspace-эндпоинтов. Экспортируется, чтобы
// роутер мог зарегистрировать GET /api/workspaces (ListMine) вне {workspaceID}.
func NewHandler(services *service.Services) *Handler {
	return &Handler{services: services}
}

// New создаёт роутер workspace-эндпоинтов, скоупнутых на {workspaceID}.
func New(services *service.Services) *chi.Mux {
	h := NewHandler(services)
	mux := chi.NewMux()

	mux.Get("/", h.get)
	mux.Patch("/", h.update)
	mux.Get("/members", h.listMembers)
	mux.Patch("/members/{userID}", h.updateMember)
	mux.Delete("/members/{userID}", h.deleteMember)
	mux.Post("/invites", h.createInvite)

	return mux
}

// ListMine godoc
//
//	@Summary		List my workspaces
//	@Description	Возвращает workspace'ы текущего пользователя вместе с его ролью (bootstrap SPA).
//	@Tags			workspace
//	@Produce		json
//	@Success		200	{object}	workspacesResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Router			/api/workspaces [get]
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	list, err := h.services.Workspace.ListForUser(r.Context(), user.ID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toWorkspacesResponse(list))
}

// get godoc
//
//	@Summary		Get current workspace
//	@Description	Возвращает текущий workspace (участник workspace).
//	@Tags			workspace
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	workspaceResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	httpx.RespondJSON(w, http.StatusOK, toWorkspaceResponse(ws))
}

// update godoc
//
//	@Summary		Update workspace settings
//	@Description	Обновляет настройки workspace (name/slug). Доступно owner/admin.
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string					true	"Workspace ID"
//	@Param			body		body		updateWorkspaceRequest	true	"Settings payload"
//	@Success		200			{object}	workspaceResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateWorkspaceRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	ws, err := h.services.Workspace.UpdateSettings(r.Context(), actor, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toWorkspaceResponse(ws))
}

// listMembers godoc
//
//	@Summary		List workspace members
//	@Description	Возвращает участников workspace (участник workspace).
//	@Tags			workspace
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	membersResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/members [get]
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	members, err := h.services.Workspace.ListMembers(r.Context(), ws.ID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toMembersResponse(members))
}

// updateMember godoc
//
//	@Summary		Update a workspace member
//	@Description	Обновляет role/username/title участника. Доступно owner/admin. Назначить роль owner нельзя.
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			userID		path		string				true	"User ID"
//	@Param			body		body		updateMemberRequest	true	"Member payload"
//	@Success		200			{object}	memberResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/members/{userID} [patch]
func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateMemberRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	userID, err := httpx.PathUUID(r, "userID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	member, err := h.services.Workspace.UpdateMember(r.Context(), actor, userID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toMemberResponse(member, "", ""))
}

// deleteMember godoc
//
//	@Summary		Remove a workspace member
//	@Description	Мягко удаляет участника workspace. Доступно owner/admin. Последнего owner удалить нельзя.
//	@Tags			workspace
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			userID		path	string	true	"User ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/members/{userID} [delete]
func (h *Handler) deleteMember(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	userID, err := httpx.PathUUID(r, "userID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Workspace.RemoveMember(r.Context(), actor, userID); err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusNoContent, nil)
}

// createInvite godoc
//
//	@Summary		Invite a member to the workspace
//	@Description	Не реализовано (отложено вместе с accept-invite).
//	@Tags			workspace
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Failure		501			"Not Implemented"
//	@Router			/api/workspaces/{workspaceID}/invites [post]
func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {
	// TODO(later): реализовать создание инвайта (см. org.invite, docs/tech/auth.md);
	// отложено вместе с POST /api/auth/accept-invite.
	w.WriteHeader(http.StatusNotImplemented)
}
