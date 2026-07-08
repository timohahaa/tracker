package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
	"github.com/timohahaa/tracker/internal/util/passwords"
)

// assertAppErrCode проверяет, что err — доменная *errs.AppError с ожидаемым кодом.
func assertAppErrCode(t *testing.T, err error, want errs.Code) {
	t.Helper()
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want *errs.AppError with code %s", err, want)
	}
	if appErr.Code != want {
		t.Fatalf("error code = %s, want %s", appErr.Code, want)
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	store.EXPECT().User().Return(users).AnyTimes()
	store.EXPECT().Session().Return(sessions).AnyTimes()

	const password = "s3cret-pw!"
	hash, err := passwords.Hash(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	users.EXPECT().GetByEmail(gomock.Any(), "ada@example.com").
		Return(model.User{ID: "user-1", Email: "ada@example.com", PasswordHash: hash}, nil)

	var sessUser string
	sessions.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, userID string, _ time.Time) (model.Session, error) {
			sessUser = userID
			return model.Session{ID: "sess-1", UserID: userID}, nil
		})

	user, session, err := newAuthService(store).Login(context.Background(), "ada@example.com", password)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("user.ID = %q, want user-1", user.ID)
	}
	if session.ID != "sess-1" || session.UserID != "user-1" {
		t.Errorf("session = %+v, want id=sess-1 user=user-1", session)
	}
	if sessUser != "user-1" {
		t.Errorf("session opened for %q, want user-1", sessUser)
	}
}

func TestLogin_WrongPasswordUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().User().Return(users).AnyTimes()

	hash, _ := passwords.Hash("correct-password")
	users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(model.User{ID: "user-1", PasswordHash: hash}, nil)
	// Session().Create не ожидается: при неверном пароле сессия не открывается.

	_, _, err := newAuthService(store).Login(context.Background(), "ada@example.com", "wrong-password")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestLogin_UnknownEmailUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().User().Return(users).AnyTimes()

	// Неизвестный email маскируется под те же invalid credentials, что и неверный
	// пароль — не раскрываем, существует ли аккаунт.
	users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(model.User{}, errs.NotFound("resource not found"))

	_, _, err := newAuthService(store).Login(context.Background(), "nobody@example.com", "whatever")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestLogin_LookupErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().User().Return(users).AnyTimes()

	dbErr := errs.Internal("db unavailable")
	users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).Return(model.User{}, dbErr)

	_, _, err := newAuthService(store).Login(context.Background(), "ada@example.com", "pw")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestLogin_SessionCreateErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	store.EXPECT().User().Return(users).AnyTimes()
	store.EXPECT().Session().Return(sessions).AnyTimes()

	const password = "s3cret-pw!"
	hash, _ := passwords.Hash(password)
	users.EXPECT().GetByEmail(gomock.Any(), gomock.Any()).
		Return(model.User{ID: "user-1", PasswordHash: hash}, nil)

	dbErr := errs.Internal("insert session failed")
	sessions.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(model.Session{}, dbErr)

	_, _, err := newAuthService(store).Login(context.Background(), "ada@example.com", password)
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

// --- AuthBySession ---

func TestAuthBySession_EmptyIDUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Пустой id отсекается до обращения к репозиторию.

	_, err := newAuthService(store).AuthBySession(context.Background(), "")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthBySession_NotFoundUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "missing").
		Return(model.Session{}, errs.NotFound("resource not found"))

	_, err := newAuthService(store).AuthBySession(context.Background(), "missing")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthBySession_ExpiredUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "sess-1").
		Return(model.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}, nil)
	// User().GetByID и Extend не должны вызываться для протухшей сессии.

	_, err := newAuthService(store).AuthBySession(context.Background(), "sess-1")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthBySession_UserNotFoundUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "sess-1").
		Return(model.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").
		Return(model.User{}, errs.NotFound("resource not found"))

	_, err := newAuthService(store).AuthBySession(context.Background(), "sess-1")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthBySession_SuccessExtendsWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "sess-1").
		Return(model.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").
		Return(model.User{ID: "user-1", Email: "ada@example.com"}, nil)
	// Sliding window: срок сессии продлевается.
	sessions.EXPECT().Extend(gomock.Any(), "sess-1", gomock.Any()).Return(nil)

	user, err := newAuthService(store).AuthBySession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("AuthBySession: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("user.ID = %q, want user-1", user.ID)
	}
}

func TestAuthBySession_ExtendFailureIsBestEffort(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "sess-1").
		Return(model.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").Return(model.User{ID: "user-1"}, nil)
	// Сбой продления не должен рушить аутентификацию.
	sessions.EXPECT().Extend(gomock.Any(), "sess-1", gomock.Any()).Return(errs.Internal("extend failed"))

	user, err := newAuthService(store).AuthBySession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("AuthBySession must succeed despite extend failure, got %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("user.ID = %q, want user-1", user.ID)
	}
}

func TestAuthBySession_UserLookupErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	sessions.EXPECT().GetByID(gomock.Any(), "sess-1").
		Return(model.Session{ID: "sess-1", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	dbErr := errs.Internal("db unavailable")
	users.EXPECT().GetByID(gomock.Any(), "user-1").Return(model.User{}, dbErr)

	_, err := newAuthService(store).AuthBySession(context.Background(), "sess-1")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

// --- AuthByToken ---

func TestAuthByToken_EmptyTokenUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)

	_, err := newAuthService(store).AuthByToken(context.Background(), "")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthByToken_NotFoundUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "missing").
		Return(model.APIToken{}, errs.NotFound("resource not found"))

	_, err := newAuthService(store).AuthByToken(context.Background(), "missing")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthByToken_UserNotFoundUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "tok-1").
		Return(model.APIToken{ID: "tok-1", UserID: "user-1"}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").
		Return(model.User{}, errs.NotFound("resource not found"))

	_, err := newAuthService(store).AuthByToken(context.Background(), "tok-1")
	assertAppErrCode(t, err, errs.CodeUnauthorized)
}

func TestAuthByToken_SuccessUpdatesLastUsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "tok-1").
		Return(model.APIToken{ID: "tok-1", UserID: "user-1"}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").Return(model.User{ID: "user-1"}, nil)
	tokens.EXPECT().UpdateLastUsed(gomock.Any(), "tok-1").Return(nil)

	user, err := newAuthService(store).AuthByToken(context.Background(), "tok-1")
	if err != nil {
		t.Fatalf("AuthByToken: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("user.ID = %q, want user-1", user.ID)
	}
}

func TestAuthByToken_UpdateLastUsedFailureIsBestEffort(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	users := repomocks.NewMockUserRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()
	store.EXPECT().User().Return(users).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "tok-1").
		Return(model.APIToken{ID: "tok-1", UserID: "user-1"}, nil)
	users.EXPECT().GetByID(gomock.Any(), "user-1").Return(model.User{ID: "user-1"}, nil)
	tokens.EXPECT().UpdateLastUsed(gomock.Any(), "tok-1").Return(errs.Internal("update failed"))

	user, err := newAuthService(store).AuthByToken(context.Background(), "tok-1")
	if err != nil {
		t.Fatalf("AuthByToken must succeed despite last_used update failure, got %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("user.ID = %q, want user-1", user.ID)
	}
}

func TestAuthByToken_LookupErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	dbErr := errs.Internal("db unavailable")
	tokens.EXPECT().GetByID(gomock.Any(), "tok-1").Return(model.APIToken{}, dbErr)

	_, err := newAuthService(store).AuthByToken(context.Background(), "tok-1")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

// --- Logout ---

func TestLogout_DeletesSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	sessions := repomocks.NewMockSessionRepository(ctrl)
	store.EXPECT().Session().Return(sessions).AnyTimes()

	sessions.EXPECT().Delete(gomock.Any(), "sess-1").Return(nil)

	if err := newAuthService(store).Logout(context.Background(), "sess-1"); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}
