package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты GitLabService написаны от контракта слайса 2.7: настройка интеграции
// (owner/admin, credentials не протекают), обработка merge_request webhook
// (верификация подписи, парсинг identifier, автостатус, идемпотентный upsert,
// non-fatal linkback). Наблюдаемое поведение — входы репозиториев/клиента и коды
// ошибок; негативные инварианты — отсутствием ожиданий.

// fakeGitLabClient фиксирует аргументы linkback и опционально возвращает ошибку
// (проверка не-фатальности сетевого сбоя). Реальных вызовов сети нет.
type fakeGitLabClient struct {
	calls   int
	baseURL string
	token   string
	project int
	mrIID   int
	body    string
	err     error
}

func (f *fakeGitLabClient) PostMRComment(_ context.Context, baseURL, token string, projectID, mrIID int, body string) error {
	f.calls++
	f.baseURL = baseURL
	f.token = token
	f.project = projectID
	f.mrIID = mrIID
	f.body = body
	return f.err
}

func newGitLabService(store repository.Store, client GitLabClient) *GitLabService {
	return &GitLabService{repos: store, gitlab: client}
}

func gitlabActor(role model.WorkspaceRole) model.WorkspaceMember {
	return model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: role}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// --- extractIdentifiers (чистый хелпер) ---

func TestExtractIdentifiers(t *testing.T) {
	// branch, title, description; регистр нормализуется в upper; дедуп по (key,number).
	got := extractIdentifiers(
		"feature/eng-123-add-login",
		"ENG-123: add login and fix ENG-456",
		"relates to team-7",
	)
	want := []mrIdentifier{
		{key: "ENG", number: 123},
		{key: "ENG", number: 456},
		{key: "TEAM", number: 7},
	}
	if len(got) != len(want) {
		t.Fatalf("extractIdentifiers len = %d (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("identifier[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestExtractIdentifiers_None(t *testing.T) {
	if got := extractIdentifiers("no identifiers here", "", "just text"); len(got) != 0 {
		t.Errorf("extractIdentifiers = %v, want empty", got)
	}
}

// --- SetupGitLab: гейт owner/admin ---

func TestSetupGitLab_MemberForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к БД: member отсекается первым.

	_, err := newGitLabService(store, &fakeGitLabClient{}).SetupGitLab(context.Background(),
		gitlabActor(model.WorkspaceRoleMember), SetupGitLabParams{
			BaseURL: "https://gitlab.example.com", AccessToken: "tok", WebhookSecret: "sec",
		})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestSetupGitLab_OwnerUpsertsWithoutLeakingCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	var saved model.Integration
	integrations.EXPECT().Upsert(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Integration) (model.Integration, error) {
			saved = i
			i.ID = "int-1"
			return i, nil
		})

	gi, err := newGitLabService(store, &fakeGitLabClient{}).SetupGitLab(context.Background(),
		gitlabActor(model.WorkspaceRoleOwner), SetupGitLabParams{
			BaseURL: "https://gitlab.example.com", AccessToken: "glpat-xxx", WebhookSecret: "sec",
		})
	if err != nil {
		t.Fatalf("SetupGitLab: %v", err)
	}
	// Записанные credentials содержат секреты...
	if !contains(saved.Credentials, "glpat-xxx") || !contains(saved.Credentials, "sec") {
		t.Errorf("saved credentials = %s, want secrets persisted", saved.Credentials)
	}
	// ...но в ответ уходит только settings, credentials обнулены.
	if gi.Integration.Credentials != nil {
		t.Errorf("response credentials = %s, want nil", gi.Integration.Credentials)
	}
	if gi.Settings.BaseURL != "https://gitlab.example.com" || !gi.Connected {
		t.Errorf("response = %+v, want connected with base_url", gi)
	}
}

// --- GetGitLab: credentials не протекают + не подключено ---

func TestGetGitLab_NullifiesCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByWorkspaceProvider(gomock.Any(), "ws-1", model.ProviderGitLab).
		Return(model.Integration{
			ID:          "int-1",
			WorkspaceID: "ws-1",
			Provider:    model.ProviderGitLab,
			Settings:    []byte(`{"base_url":"https://gitlab.example.com"}`),
			Credentials: []byte(`{"access_token":"glpat-xxx","webhook_secret":"sec"}`),
		}, nil)

	gi, err := newGitLabService(store, &fakeGitLabClient{}).GetGitLab(context.Background(), gitlabActor(model.WorkspaceRoleMember))
	if err != nil {
		t.Fatalf("GetGitLab: %v", err)
	}
	if gi.Integration.Credentials != nil {
		t.Errorf("credentials = %s, want nil (never leaked)", gi.Integration.Credentials)
	}
	if !gi.Connected || gi.Settings.BaseURL != "https://gitlab.example.com" {
		t.Errorf("gi = %+v, want connected with base_url", gi)
	}
}

func TestGetGitLab_NotConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByWorkspaceProvider(gomock.Any(), "ws-1", model.ProviderGitLab).
		Return(model.Integration{}, errs.NotFound("resource not found"))

	gi, err := newGitLabService(store, &fakeGitLabClient{}).GetGitLab(context.Background(), gitlabActor(model.WorkspaceRoleMember))
	if err != nil {
		t.Fatalf("GetGitLab not connected: %v", err)
	}
	if gi.Connected {
		t.Errorf("connected = true, want false when no integration")
	}
}

// --- DeleteGitLab: гейт owner/admin ---

func TestDeleteGitLab_MemberForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к БД: member отсекается.

	err := newGitLabService(store, &fakeGitLabClient{}).DeleteGitLab(context.Background(), gitlabActor(model.WorkspaceRoleMember))
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestDeleteGitLab_OwnerSoftDeletes(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByWorkspaceProvider(gomock.Any(), "ws-1", model.ProviderGitLab).
		Return(model.Integration{ID: "int-1", WorkspaceID: "ws-1"}, nil)
	integrations.EXPECT().Delete(gomock.Any(), "int-1").Return(nil)

	if err := newGitLabService(store, &fakeGitLabClient{}).DeleteGitLab(context.Background(), gitlabActor(model.WorkspaceRoleAdmin)); err != nil {
		t.Fatalf("DeleteGitLab: %v", err)
	}
}

// --- ProcessGitLabEvent: верификация подписи ---

func gitlabIntegration() model.Integration {
	return model.Integration{
		ID:          "int-1",
		WorkspaceID: "ws-1",
		Provider:    model.ProviderGitLab,
		Settings:    []byte(`{"base_url":"https://gitlab.example.com"}`),
		Credentials: []byte(`{"access_token":"glpat-xxx","webhook_secret":"secret"}`),
	}
}

func mrEventBody(t *testing.T, action, state string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"object_kind": "merge_request",
		"project":     map[string]any{"id": 42},
		"user":        map[string]any{"name": "Dev"},
		"object_attributes": map[string]any{
			"iid":           7,
			"title":         "ENG-123 add login",
			"description":   "",
			"source_branch": "feature/eng-123",
			"state":         state,
			"action":        action,
			"url":           "https://gitlab.example.com/g/p/-/merge_requests/7",
		},
	})
}

func TestProcessGitLabEvent_InvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByID(gomock.Any(), "int-1").Return(gitlabIntegration(), nil)
	// Ни Issue, ни Attachment, ни linkback: подпись невалидна.

	err := newGitLabService(store, &fakeGitLabClient{}).ProcessGitLabEvent(
		context.Background(), "int-1", "wrong-token", mrEventBody(t, "open", "opened"))
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestProcessGitLabEvent_IntegrationNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByID(gomock.Any(), "missing").
		Return(model.Integration{}, errs.NotFound("resource not found"))

	err := newGitLabService(store, &fakeGitLabClient{}).ProcessGitLabEvent(
		context.Background(), "missing", "secret", mrEventBody(t, "open", "opened"))
	assertAppErrCode(t, err, errs.CodeNotFound)
}

// --- ProcessGitLabEvent: маппинг action → группа статуса + upsert + linkback ---

func TestProcessGitLabEvent_ActionMapsToStatusGroup(t *testing.T) {
	cases := []struct {
		action    string
		state     string
		wantGroup model.StatusGroup
		wantStID  string
	}{
		{"open", "opened", model.StatusGroupInProgress, "st-inprogress"},
		{"merge", "merged", model.StatusGroupCompleted, "st-completed"},
		{"close", "closed", model.StatusGroupCanceled, "st-canceled"},
	}
	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := repomocks.NewMockStore(ctrl)
			integrations := repomocks.NewMockIntegrationRepository(ctrl)
			issues := repomocks.NewMockIssueRepository(ctrl)
			statuses := repomocks.NewMockStatusRepository(ctrl)
			attachments := repomocks.NewMockAttachmentRepository(ctrl)
			store.EXPECT().Integration().Return(integrations).AnyTimes()
			store.EXPECT().Issue().Return(issues).AnyTimes()
			store.EXPECT().Status().Return(statuses).AnyTimes()
			store.EXPECT().Attachment().Return(attachments).AnyTimes()
			expectTx(store)

			integrations.EXPECT().GetByID(gomock.Any(), "int-1").Return(gitlabIntegration(), nil)
			issues.EXPECT().GetByIdentifier(gomock.Any(), "ws-1", "ENG", 123).Return(model.IssueDetails{
				Issue:   model.Issue{ID: "issue-1", TeamID: "team-1", StatusID: "st-backlog"},
				TeamKey: "ENG",
			}, nil)
			statuses.EXPECT().List(gomock.Any(), "team-1").Return([]model.Status{
				{ID: "st-backlog", TeamID: "team-1", Group: model.StatusGroupBacklog, Position: 0},
				{ID: "st-inprogress", TeamID: "team-1", Group: model.StatusGroupInProgress, Position: 0},
				{ID: "st-completed", TeamID: "team-1", Group: model.StatusGroupCompleted, Position: 0},
				{ID: "st-canceled", TeamID: "team-1", Group: model.StatusGroupCanceled, Position: 0},
			}, nil)

			var upserted model.Attachment
			attachments.EXPECT().Upsert(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, a model.Attachment) (model.Attachment, error) {
					upserted = a
					a.ID = "att-1"
					return a, nil
				})
			var updated model.Issue
			issues.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) {
					updated = i
					return i, nil
				})

			client := &fakeGitLabClient{}
			if err := newGitLabService(store, client).ProcessGitLabEvent(
				context.Background(), "int-1", "secret", mrEventBody(t, tc.action, tc.state)); err != nil {
				t.Fatalf("ProcessGitLabEvent: %v", err)
			}

			if updated.StatusID != tc.wantStID {
				t.Errorf("issue status = %q, want %q", updated.StatusID, tc.wantStID)
			}
			if upserted.SourceType != model.SourceTypeGitLabMR || upserted.IssueID != "issue-1" {
				t.Errorf("attachment = %+v, want gitlab_mr on issue-1", upserted)
			}
			if upserted.IntegrationID == nil || *upserted.IntegrationID != "int-1" {
				t.Errorf("attachment integration_id = %v, want int-1", upserted.IntegrationID)
			}
			// linkback вызван с данными интеграции/MR.
			if client.calls != 1 {
				t.Fatalf("linkback calls = %d, want 1", client.calls)
			}
			if client.baseURL != "https://gitlab.example.com" || client.token != "glpat-xxx" {
				t.Errorf("linkback base/token = %q/%q, want gitlab.example.com/glpat-xxx", client.baseURL, client.token)
			}
			if client.project != 42 || client.mrIID != 7 {
				t.Errorf("linkback project/mrIID = %d/%d, want 42/7", client.project, client.mrIID)
			}
		})
	}
}

// --- ProcessGitLabEvent: действие без автостатуса — no-op ---

func TestProcessGitLabEvent_UnhandledAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()

	integrations.EXPECT().GetByID(gomock.Any(), "int-1").Return(gitlabIntegration(), nil)
	// Ни Issue, ни Attachment, ни linkback: action=update вне scope.

	client := &fakeGitLabClient{}
	if err := newGitLabService(store, client).ProcessGitLabEvent(
		context.Background(), "int-1", "secret", mrEventBody(t, "update", "opened")); err != nil {
		t.Fatalf("ProcessGitLabEvent unhandled: %v", err)
	}
	if client.calls != 0 {
		t.Errorf("linkback calls = %d, want 0", client.calls)
	}
}

// --- ProcessGitLabEvent: linkback-ошибка не фатальна ---

func TestProcessGitLabEvent_LinkbackErrorNonFatal(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	attachments := repomocks.NewMockAttachmentRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()
	store.EXPECT().Attachment().Return(attachments).AnyTimes()
	expectTx(store)

	integrations.EXPECT().GetByID(gomock.Any(), "int-1").Return(gitlabIntegration(), nil)
	issues.EXPECT().GetByIdentifier(gomock.Any(), "ws-1", "ENG", 123).Return(model.IssueDetails{
		Issue: model.Issue{ID: "issue-1", TeamID: "team-1", StatusID: "st-backlog"},
	}, nil)
	statuses.EXPECT().List(gomock.Any(), "team-1").Return([]model.Status{
		{ID: "st-inprogress", TeamID: "team-1", Group: model.StatusGroupInProgress},
	}, nil)
	attachments.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(model.Attachment{ID: "att-1"}, nil)
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).Return(model.Issue{}, nil)

	client := &fakeGitLabClient{err: errors.New("network down")}
	// Несмотря на сбой linkback, обработка успешна (attachment/статус применены).
	if err := newGitLabService(store, client).ProcessGitLabEvent(
		context.Background(), "int-1", "secret", mrEventBody(t, "open", "opened")); err != nil {
		t.Fatalf("ProcessGitLabEvent should ignore linkback error, got: %v", err)
	}
	if client.calls != 1 {
		t.Errorf("linkback calls = %d, want 1", client.calls)
	}
}

// --- ProcessGitLabEvent: identifier не резолвится (чужой/несуществующий) ---

func TestProcessGitLabEvent_NoMatchingIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	integrations := repomocks.NewMockIntegrationRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Integration().Return(integrations).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	integrations.EXPECT().GetByID(gomock.Any(), "int-1").Return(gitlabIntegration(), nil)
	issues.EXPECT().GetByIdentifier(gomock.Any(), "ws-1", "ENG", 123).
		Return(model.IssueDetails{}, errs.NotFound("issue not found"))
	// Ни WithTx, ни Attachment, ни linkback: нет задачи этого workspace.

	client := &fakeGitLabClient{}
	if err := newGitLabService(store, client).ProcessGitLabEvent(
		context.Background(), "int-1", "secret", mrEventBody(t, "open", "opened")); err != nil {
		t.Fatalf("ProcessGitLabEvent no match: %v", err)
	}
	if client.calls != 0 {
		t.Errorf("linkback calls = %d, want 0 (no attachment applied)", client.calls)
	}
}

func contains(b []byte, sub string) bool {
	return strings.Contains(string(b), sub)
}
