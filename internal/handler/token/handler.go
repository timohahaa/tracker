package token

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики API-токенов.
type Handler struct {
	services *service.Services
}

// New создаёт роутер token-эндпоинтов (монтируется на /tokens).
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Get("/", h.list)
	mux.Post("/", h.create)
	mux.Delete("/{tokenID}", h.revoke)

	return mux
}

// list godoc
//
//	@Summary		List API tokens
//	@Description	Возвращает API-токены текущего пользователя в workspace (без секрета).
//	@Tags			token
//	@Produce		json
//	@Param			workspaceID	path		string	true	"Workspace ID"
//	@Success		200			{object}	tokenListResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/tokens [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	tokens, err := h.services.Token.ListTokens(r.Context(), actor)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, toTokenListResponse(tokens))
}

// create godoc
//
//	@Summary		Create an API token
//	@Description	Создаёт API-токен. Сырой токен (token) возвращается ровно один раз — в этом ответе.
//	@Tags			token
//	@Accept			json
//	@Produce		json
//	@Param			workspaceID	path		string				true	"Workspace ID"
//	@Param			body		body		createTokenRequest	true	"Token payload"
//	@Success		201			{object}	createTokenResponse
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/tokens [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[createTokenRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	actor := middleware.MemberFromContext(r.Context())
	token, err := h.services.Token.CreateToken(r.Context(), actor, req.Name)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusCreated, toCreateTokenResponse(token))
}

// revoke godoc
//
//	@Summary		Revoke an API token
//	@Description	Отзывает токен текущего пользователя. Чужой/несуществующий → 404.
//	@Tags			token
//	@Param			workspaceID	path	string	true	"Workspace ID"
//	@Param			tokenID		path	string	true	"Token ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		401			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Router			/api/workspaces/{workspaceID}/tokens/{tokenID} [delete]
func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	actor := middleware.MemberFromContext(r.Context())
	tokenID, err := httpx.PathUUID(r, "tokenID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}
	if err := h.services.Token.RevokeToken(r.Context(), actor, tokenID); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}
