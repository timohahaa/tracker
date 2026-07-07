package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	swaggerdocs "github.com/timohahaa/tracker/docs/swagger"
)

// Раздача OpenAPI-доки (только в debug). Спека отдаётся с /docs/openapi.json,
// UI рендерит Scalar. Он клиентский и тянет локальную спеку; наружу уходит только
// загрузка JS-бандла с CDN (jsDelivr).
//
// Конфиг Scalar:
//   - agent.disabled — выключает AI-агента (иначе на localhost он включён по
//     умолчанию и грузит спеку на серверы Scalar).
//   - hideTestRequestButton + hideClientButton — убирают отправку запросов,
//     оставляя только документацию.
//   - layout: classic.
const scalarHTML = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Tracker API</title>
  </head>
  <body>
    <div id="app"></div>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
      Scalar.createApiReference('#app', {
        url: '/docs/openapi.json',
        layout: 'classic',
        agent: { disabled: true },
        hideTestRequestButton: true,
        hideClientButton: true,
      })
    </script>
  </body>
</html>`

// mountDocs монтирует раздачу OpenAPI-доки: сырую спеку и Scalar UI.
// Вызывается только в debug-режиме.
func mountDocs(r chi.Router) {
	r.Get("/docs/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(swaggerdocs.SpecJSON)
	})

	ui := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(scalarHTML))
	}
	r.Get("/docs", ui)
	r.Get("/docs/", ui)
}
