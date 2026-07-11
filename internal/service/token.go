package service

import (
	"context"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// TokenService — управление API-токенами.
type TokenService struct {
	repos repository.Store
}

// CreateToken создаёт API-токен для actor в текущем workspace. Доступно любому
// участнику workspace (membership гарантирован middleware LoadWorkspace).
// Возвращает созданную запись — её id одновременно является сырым Bearer-токеном
// и отдаётся хендлером ровно один раз. Валидация имени — на границе HTTP.
func (s *TokenService) CreateToken(ctx context.Context, actor model.WorkspaceMember, name string) (model.APIToken, error) {
	return s.repos.Token().Create(ctx, model.APIToken{
		UserID:      actor.UserID,
		WorkspaceID: actor.WorkspaceID,
		Name:        name,
	})
}

// ListTokens возвращает токены текущего пользователя в текущем workspace.
func (s *TokenService) ListTokens(ctx context.Context, actor model.WorkspaceMember) ([]model.APIToken, error) {
	return s.repos.Token().ListByUser(ctx, actor.WorkspaceID, actor.UserID)
}

// RevokeToken мягко отзывает токен. Проверка владения: чужой токен (другой
// user_id/workspace_id) трактуется как отсутствие (errs.NotFound), чтобы не
// раскрывать существование чужих токенов. Идемпотентно на уровне Delete.
func (s *TokenService) RevokeToken(ctx context.Context, actor model.WorkspaceMember, tokenID string) error {
	token, err := s.repos.Token().GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if token.UserID != actor.UserID || token.WorkspaceID != actor.WorkspaceID {
		return errs.NotFound("resource not found")
	}
	return s.repos.Token().Delete(ctx, token.ID)
}
