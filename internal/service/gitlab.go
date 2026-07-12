package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// GitLabService — настройка интеграции, обработка webhook-событий, linkback.
type GitLabService struct {
	repos  repository.Store
	gitlab GitLabClient
}

// SetClient подменяет linkback-клиент. Используется в тестах для инъекции фейка
// (реальные сетевые вызовы не выполняются). Сигнатуру service.New не трогаем.
func (s *GitLabService) SetClient(c GitLabClient) {
	s.gitlab = c
}

// gitlabSettings — non-secret конфиг интеграции (отдаётся в API).
type gitlabSettings struct {
	BaseURL string `json:"base_url"`
}

// gitlabCredentials — секреты интеграции (никогда не отдаются в API).
type gitlabCredentials struct {
	AccessToken   string `json:"access_token"`
	WebhookSecret string `json:"webhook_secret"`
}

// attachmentMetadata — provider-specific данные MR, сохраняемые в
// attachment.metadata (jsonb).
type attachmentMetadata struct {
	Branch string `json:"branch"`
	Author string `json:"author"`
	State  string `json:"state"`
}

// SetupGitLabParams — входные данные настройки GitLab (граница service).
type SetupGitLabParams struct {
	BaseURL       string
	AccessToken   string
	WebhookSecret string
}

// GitLabIntegration — read-модель статуса интеграции для API. Credentials
// намеренно отсутствуют: в ответ уходят только provider, признак «подключено» и
// non-secret settings.
type GitLabIntegration struct {
	Integration model.Integration
	Connected   bool
	Settings    gitlabSettings
}

// identifierPattern — извлечение issue identifier вида TEAM-123 из branch/title/
// description. Ключ команды — буквенно-цифровой (нормализуется в upper при
// резолве), номер — положительное целое.
var identifierPattern = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9]*)-(\d+)\b`)

// mrActionToStatusGroup — маппинг действия merge_request на целевую группу
// статуса. Берётся первый статус группы по position у команды задачи. Прочие
// действия (reopen/update/approved/...) статус не меняют.
var mrActionToStatusGroup = map[string]model.StatusGroup{
	"open":  model.StatusGroupInProgress,
	"merge": model.StatusGroupCompleted,
	"close": model.StatusGroupCanceled,
}

// SetupGitLab настраивает (или переподключает) интеграцию GitLab. Доступно
// только owner/admin workspace. Возвращает статус без credentials.
func (s *GitLabService) SetupGitLab(ctx context.Context, actor model.WorkspaceMember, params SetupGitLabParams) (GitLabIntegration, error) {
	if !canManageWorkspace(actor.Role) {
		return GitLabIntegration{}, errs.Forbidden("insufficient permissions")
	}

	settings := gitlabSettings{BaseURL: params.BaseURL}
	creds := gitlabCredentials{AccessToken: params.AccessToken, WebhookSecret: params.WebhookSecret}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return GitLabIntegration{}, errs.Internal("marshal settings")
	}
	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return GitLabIntegration{}, errs.Internal("marshal credentials")
	}

	saved, err := s.repos.Integration().Upsert(ctx, model.Integration{
		WorkspaceID: actor.WorkspaceID,
		Provider:    model.ProviderGitLab,
		Settings:    settingsJSON,
		Credentials: credsJSON,
	})
	if err != nil {
		return GitLabIntegration{}, err
	}
	return toGitLabIntegration(saved)
}

// GetGitLab возвращает статус интеграции GitLab workspace. Доступно любому
// участнику workspace. credentials в ответ не попадают. Нет интеграции →
// «не подключено» (без ошибки).
func (s *GitLabService) GetGitLab(ctx context.Context, actor model.WorkspaceMember) (GitLabIntegration, error) {
	integration, err := s.repos.Integration().GetByWorkspaceProvider(ctx, actor.WorkspaceID, model.ProviderGitLab)
	if err != nil {
		if isNotFound(err) {
			return GitLabIntegration{Connected: false}, nil
		}
		return GitLabIntegration{}, err
	}
	return toGitLabIntegration(integration)
}

// DeleteGitLab мягко удаляет интеграцию GitLab. Доступно только owner/admin.
// Attachments остаются как исторические данные.
func (s *GitLabService) DeleteGitLab(ctx context.Context, actor model.WorkspaceMember) error {
	if !canManageWorkspace(actor.Role) {
		return errs.Forbidden("insufficient permissions")
	}
	integration, err := s.repos.Integration().GetByWorkspaceProvider(ctx, actor.WorkspaceID, model.ProviderGitLab)
	if err != nil {
		return err
	}
	return s.repos.Integration().Delete(ctx, integration.ID)
}

// gitlabMREvent — интересующие поля payload GitLab merge_request webhook.
type gitlabMREvent struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		ID int `json:"id"`
	} `json:"project"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		SourceBranch string `json:"source_branch"`
		State        string `json:"state"`
		Action       string `json:"action"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
}

// ProcessGitLabEvent обрабатывает входящее событие GitLab webhook: верифицирует
// подпись, парсит merge_request, привязывает MR к найденным задачам workspace
// (attachment upsert) и переводит их статус по действию MR. linkback-комментарий
// в MR не фатален. Возвращает доменные ошибки для маппинга в HTTP.
func (s *GitLabService) ProcessGitLabEvent(ctx context.Context, integrationID, gitlabToken string, body []byte) error {
	integration, err := s.repos.Integration().GetByID(ctx, integrationID)
	if err != nil {
		return err
	}

	var creds gitlabCredentials
	if err := json.Unmarshal(integration.Credentials, &creds); err != nil {
		return errs.Internal("unmarshal credentials")
	}
	// Постоянное по времени сравнение токена (защита от timing-атак).
	if subtle.ConstantTimeCompare([]byte(gitlabToken), []byte(creds.WebhookSecret)) != 1 {
		return errs.Unauthorized("invalid webhook token")
	}

	var event gitlabMREvent
	if err := json.Unmarshal(body, &event); err != nil {
		return errs.Validation("invalid webhook payload")
	}

	// Вне scope: не merge_request или действие без автостатуса — тихо игнорируем.
	if event.ObjectKind != "merge_request" {
		return nil
	}
	targetGroup, handled := mrActionToStatusGroup[event.ObjectAttributes.Action]
	if !handled {
		return nil
	}

	identifiers := extractIdentifiers(
		event.ObjectAttributes.SourceBranch,
		event.ObjectAttributes.Title,
		event.ObjectAttributes.Description,
	)
	if len(identifiers) == 0 {
		return nil
	}

	metadataJSON, err := json.Marshal(attachmentMetadata{
		Branch: event.ObjectAttributes.SourceBranch,
		Author: event.User.Name,
		State:  event.ObjectAttributes.State,
	})
	if err != nil {
		return errs.Internal("marshal metadata")
	}

	linkback := false
	for _, id := range identifiers {
		issue, err := s.repos.Issue().GetByIdentifier(ctx, integration.WorkspaceID, id.key, id.number)
		if err != nil {
			if isNotFound(err) {
				continue // MR-идентификатор не относится к задаче этого workspace.
			}
			return err
		}

		statusID, err := s.resolveGroupStatus(ctx, issue.TeamID, targetGroup)
		if err != nil {
			return err
		}

		// «Смена статуса + upsert attachment» — атомарно на задачу.
		if err := s.repos.WithTx(ctx, func(tx repository.Store) error {
			if _, uerr := tx.Attachment().Upsert(ctx, model.Attachment{
				IssueID:       issue.ID,
				IntegrationID: &integration.ID,
				SourceType:    model.SourceTypeGitLabMR,
				URL:           event.ObjectAttributes.URL,
				Title:         event.ObjectAttributes.Title,
				Metadata:      metadataJSON,
			}); uerr != nil {
				return uerr
			}
			if statusID != "" && statusID != issue.StatusID {
				updated := issue.Issue
				updated.StatusID = statusID
				if _, uerr := tx.Issue().Update(ctx, updated); uerr != nil {
					return uerr
				}
			}
			return nil
		}); err != nil {
			return err
		}
		linkback = true
	}

	// linkback — best-effort: attachment/статус уже применены, сетевую ошибку
	// не считаем фатальной.
	if linkback {
		var settings gitlabSettings
		if err := json.Unmarshal(integration.Settings, &settings); err == nil {
			body := fmt.Sprintf("Linked to tracker issue(s): %s", event.ObjectAttributes.URL)
			if lerr := s.gitlab.PostMRComment(ctx, settings.BaseURL, creds.AccessToken,
				event.Project.ID, event.ObjectAttributes.IID, body); lerr != nil {
				log.Warnf("gitlab linkback: %v", lerr)
			}
		}
	}

	return nil
}

// resolveGroupStatus возвращает id первого статуса целевой группы команды (по
// position). Пустой id, если у команды нет статуса этой группы (тогда автостатус
// пропускается — не ошибка). List упорядочен по (group, position).
func (s *GitLabService) resolveGroupStatus(ctx context.Context, teamID string, group model.StatusGroup) (string, error) {
	statuses, err := s.repos.Status().List(ctx, teamID)
	if err != nil {
		return "", err
	}
	for _, st := range statuses {
		if st.Group == group {
			return st.ID, nil
		}
	}
	return "", nil
}

// mrIdentifier — распарсенный issue identifier (нормализованный ключ + номер).
type mrIdentifier struct {
	key    string
	number int
}

// extractIdentifiers извлекает уникальные issue identifiers (TEAM-123) из
// переданных строк. Ключ нормализуется в upper (как ключ команды при создании),
// поэтому матчинг регистронезависим. Дедуп по (key, number) с сохранением
// порядка первого появления.
func extractIdentifiers(sources ...string) []mrIdentifier {
	seen := make(map[string]bool)
	var out []mrIdentifier
	for _, src := range sources {
		for _, m := range identifierPattern.FindAllStringSubmatch(src, -1) {
			key := strings.ToUpper(m[1])
			number, err := atoiPositive(m[2])
			if err != nil {
				continue
			}
			dedup := fmt.Sprintf("%s-%d", key, number)
			if seen[dedup] {
				continue
			}
			seen[dedup] = true
			out = append(out, mrIdentifier{key: key, number: number})
		}
	}
	return out
}

// atoiPositive парсит строку в положительное целое. Ошибка, если не число или
// значение <= 0 (issue_number всегда >= 1).
func atoiPositive(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("non-positive number: %s", s)
	}
	return n, nil
}

// toGitLabIntegration собирает read-модель статуса интеграции, распаковывая
// non-secret settings. credentials в результат не попадают.
func toGitLabIntegration(i model.Integration) (GitLabIntegration, error) {
	var settings gitlabSettings
	if len(i.Settings) > 0 {
		if err := json.Unmarshal(i.Settings, &settings); err != nil {
			return GitLabIntegration{}, errs.Internal("unmarshal settings")
		}
	}
	// credentials обнуляем перед отдачей — в API уходят только settings.
	i.Credentials = nil
	return GitLabIntegration{Integration: i, Connected: true, Settings: settings}, nil
}
