package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
	"github.com/timohahaa/tracker/internal/util/passwords"
)

// Тесты Register написаны от контракта, задокументированного в docs (docs/product/
// features/auth.md, docs/tech/auth.md, docs/tech/error-handling.md) и в doc-комментарии
// метода: регистрация создаёт пользователя, workspace и owner-membership в одной
// транзакции, затем открывает сессию; slug workspace и username владельца
// генерируются автоматически; пароль хранится хешем; дубль email → already_exists.
// Проверяется наблюдаемое поведение (входы репозиториев и выходы Register), а не
// внутренняя последовательность вызовов.

func newAuthService(store repository.Store) *AuthService {
	cfg := config.Config{}
	cfg.SetDefaults()
	return &AuthService{repos: store, cfg: cfg}
}

// expectTx настраивает WithTx как «прозрачный» транзакционный раннер: выполняет
// колбэк с тем же Store. Атомарность (откат при ошибке) — деталь реальной БД и
// покрывается integration-тестами; на unit-уровне проверяем контракт сервиса:
// шаги идут внутри одной транзакции, ошибка любого шага пробрасывается наружу.
func expectTx(store *repomocks.MockStore) {
	store.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(repository.Store) error) error {
			return fn(store)
		})
}

func TestRegister_CreatesOwnerWorkspaceAndSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)

	expectTx(store)
	store.EXPECT().User().Return(users).AnyTimes()
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()
	store.EXPECT().Session().Return(sessions).AnyTimes()

	params := RegisterParams{
		Email:         "ada@example.com",
		Password:      "s3cret-pw!",
		DisplayName:   "Ada Lovelace",
		WorkspaceName: "Acme Inc",
	}

	var createdUser model.User
	users.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, u model.User) (model.User, error) {
			createdUser = u
			u.ID = "user-1"
			return u, nil
		})

	// Слаг свободен — генератору достаётся первый кандидат.
	workspaces.EXPECT().SlugTaken(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()

	var createdWS model.Workspace
	workspaces.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ws model.Workspace) (model.Workspace, error) {
			createdWS = ws
			ws.ID = "ws-1"
			return ws, nil
		})

	var member model.WorkspaceMember
	workspaces.EXPECT().CreateMember(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error) {
			member = m
			return m, nil
		})

	var sessionUserID string
	sessions.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, userID string, _ time.Time) (model.Session, error) {
			sessionUserID = userID
			return model.Session{ID: "sess-1", UserID: userID}, nil
		})

	gotUser, gotWS, gotSession, err := newAuthService(store).Register(context.Background(), params)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	// Возвращается созданный пользователь.
	if gotUser.ID != "user-1" || gotUser.Email != params.Email || gotUser.DisplayName != params.DisplayName {
		t.Errorf("returned user = %+v, want id=user-1 email=%s name=%s", gotUser, params.Email, params.DisplayName)
	}

	// Пароль хранится в виде хеша, а не в открытом виде (docs: auth по email+password).
	if createdUser.PasswordHash == "" || createdUser.PasswordHash == params.Password {
		t.Errorf("password must be stored hashed, got %q", createdUser.PasswordHash)
	}
	if !passwords.Verify(createdUser.PasswordHash, params.Password) {
		t.Error("stored password hash does not verify against the original password")
	}

	// Workspace создаётся с автоматически сгенерированным url-safe slug.
	if gotWS.ID != "ws-1" || gotWS.Name != params.WorkspaceName {
		t.Errorf("returned workspace = %+v, want id=ws-1 name=%s", gotWS, params.WorkspaceName)
	}
	if createdWS.Slug == "" {
		t.Error("workspace slug must be auto-generated, got empty")
	}
	if createdWS.Slug != slugify(params.WorkspaceName) {
		t.Errorf("workspace slug = %q, want url-safe %q", createdWS.Slug, slugify(params.WorkspaceName))
	}

	// Создатель становится owner (docs: Owner — создатель, один на workspace) с
	// автоматически сгенерированным username, привязанным к созданным user/workspace.
	if member.Role != model.WorkspaceRoleOwner {
		t.Errorf("member role = %q, want %q", member.Role, model.WorkspaceRoleOwner)
	}
	if member.Username == "" {
		t.Error("member username must be auto-generated, got empty")
	}
	if member.WorkspaceID != "ws-1" || member.UserID != "user-1" {
		t.Errorf("member links = (ws %q, user %q), want (ws-1, user-1)", member.WorkspaceID, member.UserID)
	}

	// Сессия открывается для только что созданного пользователя.
	if sessionUserID != "user-1" {
		t.Errorf("session opened for user %q, want user-1", sessionUserID)
	}
	if gotSession.ID != "sess-1" || gotSession.UserID != "user-1" {
		t.Errorf("returned session = %+v, want id=sess-1 user=user-1", gotSession)
	}
}

func TestRegister_DuplicateEmailReturnsAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)

	expectTx(store)
	store.EXPECT().User().Return(users).AnyTimes()
	store.EXPECT().Session().Return(sessions).AnyTimes()

	// Репозиторий маппит уникальное нарушение email в already_exists (docs/tech/
	// error-handling.md: 23505 → AlreadyExists).
	users.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(model.User{}, errs.AlreadyExists("resource already exists"))
	// Никаких ожиданий на sessions.Create: провал регистрации не открывает сессию.

	gotUser, gotWS, gotSession, err := newAuthService(store).Register(context.Background(), RegisterParams{
		Email:         "dup@example.com",
		Password:      "s3cret-pw!",
		DisplayName:   "Dup",
		WorkspaceName: "Dup WS",
	})

	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeAlreadyExists {
		t.Fatalf("error = %v, want code %s", err, errs.CodeAlreadyExists)
	}
	if gotUser != (model.User{}) || gotWS != (model.Workspace{}) || gotSession != (model.Session{}) {
		t.Errorf("on failure expected zero values, got user=%+v ws=%+v session=%+v", gotUser, gotWS, gotSession)
	}
}

func TestRegister_StepFailureAbortsAndOpensNoSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)

	expectTx(store)
	store.EXPECT().User().Return(users).AnyTimes()
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()
	store.EXPECT().Session().Return(sessions).AnyTimes()

	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(model.User{ID: "user-1"}, nil)
	workspaces.EXPECT().SlugTaken(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	workspaces.EXPECT().Create(gomock.Any(), gomock.Any()).Return(model.Workspace{ID: "ws-1"}, nil)

	// Сбой на шаге внутри транзакции.
	stepErr := errs.Internal("insert member failed")
	workspaces.EXPECT().CreateMember(gomock.Any(), gomock.Any()).Return(model.WorkspaceMember{}, stepErr)
	// sessions.Create не ожидается: незавершённая транзакция не должна открыть сессию.

	gotUser, gotWS, gotSession, err := newAuthService(store).Register(context.Background(), RegisterParams{
		Email:         "ada@example.com",
		Password:      "s3cret-pw!",
		DisplayName:   "Ada",
		WorkspaceName: "Acme",
	})
	if !errors.Is(err, stepErr) {
		t.Fatalf("error = %v, want %v", err, stepErr)
	}
	if gotUser != (model.User{}) || gotWS != (model.Workspace{}) || gotSession != (model.Session{}) {
		t.Errorf("on failure expected zero values, got user=%+v ws=%+v session=%+v", gotUser, gotWS, gotSession)
	}
}

func TestRegister_TransactionErrorIsSurfaced(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)

	// Транзакция не смогла выполниться (begin/commit) — регистрация целиком идёт
	// в транзакции, поэтому ошибка пробрасывается, а частичный результат не возвращается.
	txErr := errs.Internal("could not run transaction")
	store.EXPECT().WithTx(gomock.Any(), gomock.Any()).Return(txErr)

	gotUser, gotWS, gotSession, err := newAuthService(store).Register(context.Background(), RegisterParams{
		Email:         "ada@example.com",
		Password:      "s3cret-pw!",
		DisplayName:   "Ada",
		WorkspaceName: "Acme",
	})
	if !errors.Is(err, txErr) {
		t.Fatalf("error = %v, want %v", err, txErr)
	}
	if gotUser != (model.User{}) || gotWS != (model.Workspace{}) || gotSession != (model.Session{}) {
		t.Errorf("on tx failure expected zero values, got user=%+v ws=%+v session=%+v", gotUser, gotWS, gotSession)
	}
}
