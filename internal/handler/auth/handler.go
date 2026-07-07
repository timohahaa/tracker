package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// Handler — HTTP-обработчики аутентификации (/api/auth/*).
type Handler struct {
	services *service.Services
}

// New создаёт роутер auth-эндпоинтов.
func New(services *service.Services) *chi.Mux {
	h := &Handler{services: services}
	mux := chi.NewMux()

	mux.Post("/register", h.register)
	mux.Post("/login", h.login)
	mux.Post("/logout", h.logout)
	mux.Post("/accept-invite", h.acceptInvite)

	return mux
}

// register godoc
//
//	@Summary		Register a new user and workspace
//	@Description	Создаёт пользователя, его workspace и owner-membership, открывает сессию.
//	@Tags			auth
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Registration payload"
//	@Success		201		{object}	registerResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Header			201		{string}	Set-Cookie	"session_id"
//	@Router			/api/auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[registerRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	user, ws, session, err := h.services.Auth.Register(r.Context(), req.toParams())
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	h.setSessionCookie(w, session)
	httpx.RespondJSON(w, http.StatusCreated, registerResponse{
		User:      toUserResponse(user),
		Workspace: toWorkspaceResponse(ws),
	})
}

// login godoc
//
//	@Summary		Log in with email and password
//	@Description	Проверяет учётные данные и открывает сессию (Set-Cookie session_id).
//	@Tags			auth
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login payload"
//	@Success		200		{object}	loginResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Header			200		{string}	Set-Cookie	"session_id"
//	@Router			/api/auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeJSON[loginRequest](r)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	user, session, err := h.services.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	h.setSessionCookie(w, session)
	httpx.RespondJSON(w, http.StatusOK, loginResponse{User: toUserResponse(user)})
}

// logout godoc
//
//	@Summary		Log out the current session
//	@Description	Удаляет сессию и очищает cookie.
//	@Tags			auth
//	@Produce		json
//	@Success		204	"No Content"
//	@Router			/api/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(h.services.Auth.SessionCookieName()); err == nil && c.Value != "" {
		if err := h.services.Auth.Logout(r.Context(), c.Value); err != nil {
			httpx.RespondError(w, err)
			return
		}
	}
	h.clearSessionCookie(w)
	httpx.RespondJSON(w, http.StatusNoContent, nil)
}

// acceptInvite godoc
//
//	@Summary		Accept a workspace invite
//	@Description	Не реализовано (Фаза 2.2).
//	@Tags			auth
//	@Failure		501	"Not Implemented"
//	@Router			/api/auth/accept-invite [post]
func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	// TODO(phase-2.2): реализовать приём инвайта (см. org.invite, docs/tech/auth.md).
	w.WriteHeader(http.StatusNotImplemented)
}

// setSessionCookie выставляет session cookie по контракту auth.md: HttpOnly,
// Secure, SameSite=Lax. Срок жизни берётся из session.ExpiresAt.
func (h *Handler) setSessionCookie(w http.ResponseWriter, session model.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.services.Auth.SessionCookieName(),
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.services.Auth.SessionCookieName(),
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
