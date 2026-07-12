package webhook

import (
	"io"
	"net/http"

	"github.com/timohahaa/tracker/internal/handler/httpx"
)

// gitlab обрабатывает входящий GitLab webhook: POST /webhooks/gitlab/{integrationID}.
// Смонтирован вне auth-группы (без session/workspace middleware) — подпись
// проверяется внутри сервиса по X-Gitlab-Token против credentials.webhook_secret.
// Верификация подписи, парсинг события, upsert attachment, автостатус и linkback
// живут в service.GitLab.ProcessGitLabEvent; здесь — только транспорт.
func (h *Handler) gitlab(w http.ResponseWriter, r *http.Request) {
	integrationID, err := httpx.PathUUID(r, "integrationID")
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.RespondError(w, err)
		return
	}

	token := r.Header.Get("X-Gitlab-Token")
	if err := h.services.GitLab.ProcessGitLabEvent(r.Context(), integrationID, token, body); err != nil {
		httpx.RespondError(w, err)
		return
	}
	httpx.RespondJSON(w, http.StatusOK, nil)
}
