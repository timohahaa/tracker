package service

import (
	"context"
	"net/http"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

// gitlabLinkbackTimeout — таймаут исходящего linkback-запроса в GitLab. Конфига
// под внешние вызовы в проекте пока нет, поэтому таймаут захардкожен.
const gitlabLinkbackTimeout = 10 * time.Second

// GitLabClient — исходящий клиент GitLab для linkback. Абстрагирует сеть, чтобы
// сервис оставался мокаемым (в тестах подменяется фейком, без реальных вызовов).
// baseURL/token берутся из credentials интеграции (свои на каждый workspace),
// поэтому передаются на каждый вызов, а не хранятся в клиенте.
type GitLabClient interface {
	PostMRComment(ctx context.Context, baseURL, token string, projectID, mrIID int, body string) error
}

// gitlabHTTPClient — реальная реализация поверх официальной библиотеки
// gitlab.com/gitlab-org/api/client-go. Клиент строится на каждый вызов, так как
// base_url/token относятся к конкретной интеграции.
type gitlabHTTPClient struct {
	httpClient *http.Client
}

// newGitLabClient создаёт реальный linkback-клиент с фиксированным таймаутом.
func newGitLabClient() GitLabClient {
	return &gitlabHTTPClient{
		httpClient: &http.Client{Timeout: gitlabLinkbackTimeout},
	}
}

// PostMRComment публикует комментарий в merge request через GitLab Notes API.
func (c *gitlabHTTPClient) PostMRComment(ctx context.Context, baseURL, token string, projectID, mrIID int, body string) error {
	client, err := gitlab.NewClient(token,
		gitlab.WithBaseURL(baseURL),
		gitlab.WithHTTPClient(c.httpClient),
	)
	if err != nil {
		return err
	}
	_, _, err = client.Notes.CreateMergeRequestNote(projectID, int64(mrIID), &gitlab.CreateMergeRequestNoteOptions{
		Body: gitlab.Ptr(body),
	}, gitlab.WithContext(ctx))
	return err
}
