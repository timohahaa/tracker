package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler/auth"
	"github.com/timohahaa/tracker/internal/handler/integration"
	"github.com/timohahaa/tracker/internal/handler/issue"
	"github.com/timohahaa/tracker/internal/handler/label"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/handler/project"
	"github.com/timohahaa/tracker/internal/handler/team"
	"github.com/timohahaa/tracker/internal/handler/token"
	"github.com/timohahaa/tracker/internal/handler/workspace"
	"github.com/timohahaa/tracker/internal/mcp"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/internal/webhook"
)

// New собирает корневой HTTP-роутер: /api/*, /mcp, /webhooks/*.
// При debug=true дополнительно раздаётся OpenAPI-дока на /docs (Scalar UI)
// и сырая спека на /docs/openapi.json.
func New(services *service.Services, cfg config.Config) http.Handler {
	mw := middleware.New(services)

	r := chi.NewMux()
	r.Use(chimw.Recoverer)

	if cfg.Debug {
		mountDocs(r)
	}

	// REST API (/api/*)
	r.Route("/api", func(r chi.Router) {
		// Публичные auth-эндпоинты + аутентифицированный /auth/me
		r.Mount("/auth", auth.New(services))

		// Аутентифицированные (SessionAuth). Список моих workspace живёт вне
		// {workspaceID} (без LoadWorkspace), остальное — скоупнуто на workspace.
		r.Group(func(r chi.Router) {
			r.Use(mw.SessionAuth)

			r.Route("/workspaces", func(r chi.Router) {
				wh := workspace.NewHandler(services)
				r.Get("/", wh.ListMine)

				r.Route("/{workspaceID}", func(r chi.Router) {
					r.Use(mw.LoadWorkspace)

					r.Mount("/teams", team.New(services))
					r.Mount("/issues", issue.New(services))
					r.Mount("/projects", project.New(services))
					r.Mount("/labels", label.New(services))
					r.Mount("/integrations", integration.New(services))
					r.Mount("/tokens", token.New(services))
					// Workspace-level (GET/PATCH корня, /members, /invites) — монтируется последним.
					r.Mount("/", workspace.New(services))
				})
			})
		})
	})

	// MCP endpoint (/mcp) — auth по API-токену.
	r.Group(func(r chi.Router) {
		r.Use(mw.TokenAuth)
		r.Mount("/mcp", mcp.New(services))
	})

	// Входящие вебхуки (/webhooks/*) — верификация подписи внутри.
	r.Mount("/webhooks", webhook.New(services))

	return r
}
