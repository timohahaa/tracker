package auth

import (
	"net/http"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
)

// me godoc
//
//	@Summary		Get current user
//	@Description	Возвращает текущего пользователя по session-cookie (bootstrap SPA).
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	userResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Router			/api/auth/me [get]
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	httpx.RespondJSON(w, http.StatusOK, toUserResponse(user))
}

// updateMe godoc
//
//	@Summary		Update current user settings
//	@Description	Частично обновляет глобальные настройки пользователя (current_workspace_id). Отсутствие поля — не менять.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		updateMeRequest	true	"Settings payload"
//	@Success		200		{object}	userResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Router			/api/auth/me [patch]
func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[updateMeRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	user := middleware.UserFromContext(r.Context())
	settings := user.Settings
	if req.CurrentWorkspaceID != nil {
		settings.CurrentWorkspaceID = req.CurrentWorkspaceID
	}

	updated, err := h.services.Auth.UpdateSettings(r.Context(), user.ID, settings)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toUserResponse(updated))
}
