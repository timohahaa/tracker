package team

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
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

// list godoc
//
//	@Summary		List teams
//	@Description	Возвращает команды workspace (участник workspace).
//	@Tags			team
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	teamsResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	teams, err := h.services.Team.ListTeams(r.Context(), ws.ID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toTeamsResponse(teams))
}

// create godoc
//
//	@Summary		Create a team
//	@Description	Создаёт команду и дефолтные статусы (участник workspace).
//	@Tags			team
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			body		body		createTeamRequest	true	"Team payload"
//	@Success		201			{object}	teamResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createTeamRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	team, err := h.services.Team.CreateTeam(r.Context(), actor, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toTeamResponse(team))
}

// get godoc
//
//	@Summary		Get a team
//	@Description	Возвращает команду (участник workspace).
//	@Tags			team
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			teamID		path		string	true	"Team ID"
//	@Success		200			{object}	teamResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID} [get]
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	team, err := h.services.Team.GetTeam(r.Context(), ws.ID, teamID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toTeamResponse(team))
}

// update godoc
//
//	@Summary		Update a team
//	@Description	Обновляет name/description/key команды (участник команды).
//	@Tags			team
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			teamID		path		string				true	"Team ID"
//	@Param			body		body		updateTeamRequest	true	"Team payload"
//	@Success		200			{object}	teamResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID} [patch]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateTeamRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	team, err := h.services.Team.UpdateTeam(r.Context(), actor, teamID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toTeamResponse(team))
}

// delete godoc
//
//	@Summary		Delete a team
//	@Description	Мягко удаляет команду (owner/admin workspace).
//	@Tags			team
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			teamID		path	string	true	"Team ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID} [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Team.DeleteTeam(r.Context(), actor, teamID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}

// listMembers godoc
//
//	@Summary		List team members
//	@Description	Возвращает участников команды (участник workspace).
//	@Tags			team
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			teamID		path		string	true	"Team ID"
//	@Success		200			{object}	membersResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/members [get]
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	members, err := h.services.Team.ListMembers(r.Context(), ws.ID, teamID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toMembersResponse(members))
}

// addMember godoc
//
//	@Summary		Add a team member
//	@Description	Добавляет участника в команду (участник команды). Кандидат — активный участник workspace.
//	@Tags			team
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			teamID		path		string				true	"Team ID"
//	@Param			body		body		addMemberRequest	true	"Member payload"
//	@Success		201			{object}	memberResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/members [post]
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[addMemberRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	member, err := h.services.Team.AddMember(r.Context(), actor, teamID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toMemberResponse(member))
}

// removeMember godoc
//
//	@Summary		Remove a team member
//	@Description	Мягко снимает участника с команды (участник команды).
//	@Tags			team
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			teamID		path	string	true	"Team ID"
//	@Param			userID		path	string	true	"User ID"
//	@Success		204			"No Content"
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/members/{userID} [delete]
func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	userID, err := httpx.PathUUID(r, "userID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Team.RemoveMember(r.Context(), actor, teamID, userID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}

// listStatuses godoc
//
//	@Summary		List team statuses
//	@Description	Возвращает статусы команды (участник workspace).
//	@Tags			team
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Param			teamID		path		string	true	"Team ID"
//	@Success		200			{object}	statusesResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/statuses [get]
func (h *Handler) listStatuses(w http.ResponseWriter, r *http.Request) {
	ws := middleware.WorkspaceFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	statuses, err := h.services.Team.ListStatuses(r.Context(), ws.ID, teamID)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toStatusesResponse(statuses))
}

// createStatus godoc
//
//	@Summary		Create a status
//	@Description	Создаёт статус в команде (участник команды).
//	@Tags			team
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			teamID		path		string				true	"Team ID"
//	@Param			body		body		createStatusRequest	true	"Status payload"
//	@Success		201			{object}	statusResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/statuses [post]
func (h *Handler) createStatus(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createStatusRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	status, err := h.services.Team.CreateStatus(r.Context(), actor, teamID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toStatusResponse(status))
}

// updateStatus godoc
//
//	@Summary		Update a status
//	@Description	Обновляет name/position статуса (участник команды).
//	@Tags			team
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			teamID		path		string				true	"Team ID"
//	@Param			statusID	path		string				true	"Status ID"
//	@Param			body		body		updateStatusRequest	true	"Status payload"
//	@Success		200			{object}	statusResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		409			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/statuses/{statusID} [patch]
func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateStatusRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	statusID, err := httpx.PathUUID(r, "statusID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	status, err := h.services.Team.UpdateStatus(r.Context(), actor, teamID, statusID, req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toStatusResponse(status))
}

// deleteStatus godoc
//
//	@Summary		Delete a status
//	@Description	Мягко удаляет статус (участник команды). Нельзя удалить последний статус в группе.
//	@Tags			team
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			teamID		path	string	true	"Team ID"
//	@Param			statusID	path	string	true	"Status ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/teams/{teamID}/statuses/{statusID} [delete]
func (h *Handler) deleteStatus(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	teamID, err := httpx.PathUUID(r, "teamID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	statusID, err := httpx.PathUUID(r, "statusID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Team.DeleteStatus(r.Context(), actor, teamID, statusID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}
