package service

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты TokenService написаны от контракта слайса 2.1 (PLAN.md, docs/tech/api.md):
// токены строго per-(user, workspace); создание/список скоупятся actor'ом; отзыв
// чужого токена трактуется как отсутствие (NotFound), свой — удаляется.
// Наблюдаемое поведение — входы репозитория; негативные инварианты —
// отсутствием ожиданий (лишний вызов Delete роняет тест).

func newTokenService(store repository.Store) *TokenService {
	return &TokenService{repos: store}
}

func tokenActor() model.WorkspaceMember {
	return model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
}

func TestCreateToken_ScopesToActor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	var created model.APIToken
	tokens.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, tok model.APIToken) (model.APIToken, error) {
			created = tok
			tok.ID = "tok-1"
			return tok, nil
		})

	got, err := newTokenService(store).CreateToken(context.Background(), tokenActor(), "ci")
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if created.UserID != "user-1" || created.WorkspaceID != "ws-1" {
		t.Errorf("created scope = user %q ws %q, want user-1/ws-1", created.UserID, created.WorkspaceID)
	}
	if created.Name != "ci" {
		t.Errorf("created name = %q, want ci", created.Name)
	}
	if got.ID != "tok-1" {
		t.Errorf("returned id = %q, want tok-1 (raw token)", got.ID)
	}
}

func TestListTokens_ScopesToActor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	tokens.EXPECT().ListByUser(gomock.Any(), "ws-1", "user-1").
		Return([]model.APIToken{{ID: "tok-1", UserID: "user-1", WorkspaceID: "ws-1"}}, nil)

	got, err := newTokenService(store).ListTokens(context.Background(), tokenActor())
	if err != nil {
		t.Fatalf("ListTokens: %v", err)
	}
	if len(got) != 1 || got[0].ID != "tok-1" {
		t.Errorf("tokens = %v, want [tok-1]", got)
	}
}

func TestRevokeToken_OwnTokenDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "tok-1").
		Return(model.APIToken{ID: "tok-1", UserID: "user-1", WorkspaceID: "ws-1"}, nil)
	tokens.EXPECT().Delete(gomock.Any(), "tok-1").Return(nil)

	if err := newTokenService(store).RevokeToken(context.Background(), tokenActor(), "tok-1"); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
}

func TestRevokeToken_ForeignUserNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	// Токен принадлежит другому пользователю → трактуем как отсутствие, Delete не зовём.
	tokens.EXPECT().GetByID(gomock.Any(), "tok-2").
		Return(model.APIToken{ID: "tok-2", UserID: "user-2", WorkspaceID: "ws-1"}, nil)

	err := newTokenService(store).RevokeToken(context.Background(), tokenActor(), "tok-2")
	assertAppErrCode(t, err, errs.CodeNotFound)
}

func TestRevokeToken_ForeignWorkspaceNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	// Токен из другого workspace → NotFound, Delete не зовём.
	tokens.EXPECT().GetByID(gomock.Any(), "tok-3").
		Return(model.APIToken{ID: "tok-3", UserID: "user-1", WorkspaceID: "ws-2"}, nil)

	err := newTokenService(store).RevokeToken(context.Background(), tokenActor(), "tok-3")
	assertAppErrCode(t, err, errs.CodeNotFound)
}

func TestRevokeToken_MissingPropagatesNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	tokens := repomocks.NewMockTokenRepository(ctrl)
	store.EXPECT().Token().Return(tokens).AnyTimes()

	tokens.EXPECT().GetByID(gomock.Any(), "missing").
		Return(model.APIToken{}, errs.NotFound("resource not found"))
	// Delete не ожидается.

	err := newTokenService(store).RevokeToken(context.Background(), tokenActor(), "missing")
	assertAppErrCode(t, err, errs.CodeNotFound)
}
