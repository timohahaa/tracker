package service

import (
	"context"
	"errors"
	"time"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/util/passwords"
)

// AuthService — регистрация, логин, сессии, аутентификация по сессии/токену.
type AuthService struct {
	repos repository.Store
	cfg   config.Config
}

// RegisterParams — входные данные регистрации (граница service, не HTTP).
type RegisterParams struct {
	Email         string
	Password      string
	DisplayName   string
	WorkspaceName string
}

// Register создаёт пользователя, его workspace и owner-membership в одной
// транзакции, затем открывает сессию. slug workspace и username владельца
// генерируются автоматически. Дубль email → errs.AlreadyExists.
func (s *AuthService) Register(ctx context.Context, p RegisterParams) (model.User, model.Workspace, model.Session, error) {
	hash, err := passwords.Hash(p.Password)
	if err != nil {
		return model.User{}, model.Workspace{}, model.Session{}, errs.Internal("could not hash password")
	}

	var (
		user    model.User
		ws      model.Workspace
		session model.Session
	)

	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		user, err = tx.User().Create(ctx, model.User{
			Email:        p.Email,
			PasswordHash: hash,
			DisplayName:  p.DisplayName,
		})
		if err != nil {
			return err
		}

		slug, err := uniqueSlug(ctx, p.WorkspaceName, tx.Workspace().SlugTaken)
		if err != nil {
			return err
		}
		ws, err = tx.Workspace().Create(ctx, model.Workspace{
			Name: p.WorkspaceName,
			Slug: slug,
		})
		if err != nil {
			return err
		}

		if _, err = tx.Workspace().CreateMember(ctx, model.WorkspaceMember{
			WorkspaceID: ws.ID,
			UserID:      user.ID,
			Role:        model.WorkspaceRoleOwner,
			Username:    defaultUsername(p.DisplayName, p.Email),
		}); err != nil {
			return err
		}

		session, err = tx.Session().Create(ctx, user.ID, s.sessionExpiry())
		return err
	})
	if err != nil {
		return model.User{}, model.Workspace{}, model.Session{}, err
	}

	return user, ws, session, nil
}

// Login проверяет учётные данные и открывает сессию. Неверные креды (нет такого
// email либо не совпал пароль) → errs.Unauthorized без раскрытия, что именно не так.
func (s *AuthService) Login(ctx context.Context, email, password string) (model.User, model.Session, error) {
	user, err := s.repos.User().GetByEmail(ctx, email)
	if err != nil {
		if isNotFound(err) {
			return model.User{}, model.Session{}, errs.Unauthorized("invalid credentials")
		}
		return model.User{}, model.Session{}, err
	}

	if !passwords.Verify(user.PasswordHash, password) {
		return model.User{}, model.Session{}, errs.Unauthorized("invalid credentials")
	}

	session, err := s.repos.Session().Create(ctx, user.ID, s.sessionExpiry())
	if err != nil {
		return model.User{}, model.Session{}, err
	}
	return user, session, nil
}

// Logout удаляет сессию по id (idempotent).
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.repos.Session().Delete(ctx, sessionID)
}

// AuthBySession резолвит сессию из cookie в пользователя: проверяет срок,
// продлевает окно (sliding window) и возвращает владельца. Нет/протухла →
// errs.Unauthorized.
func (s *AuthService) AuthBySession(ctx context.Context, sessionID string) (model.User, error) {
	if sessionID == "" {
		return model.User{}, errs.Unauthorized("authentication required")
	}

	session, err := s.repos.Session().GetByID(ctx, sessionID)
	if err != nil {
		if isNotFound(err) {
			return model.User{}, errs.Unauthorized("invalid session")
		}
		return model.User{}, err
	}
	if !time.Now().Before(session.ExpiresAt) {
		return model.User{}, errs.Unauthorized("session expired")
	}

	user, err := s.repos.User().GetByID(ctx, session.UserID)
	if err != nil {
		if isNotFound(err) {
			return model.User{}, errs.Unauthorized("invalid session")
		}
		return model.User{}, err
	}

	// Продление окна — best-effort: сбой продления не должен рушить аутентификацию.
	_ = s.repos.Session().Extend(ctx, session.ID, s.sessionExpiry())

	return user, nil
}

// AuthByToken резолвит API-токен из Bearer-заголовка в пользователя и обновляет
// last_used_at. Нет/невалиден → errs.Unauthorized.
func (s *AuthService) AuthByToken(ctx context.Context, tokenID string) (model.User, error) {
	if tokenID == "" {
		return model.User{}, errs.Unauthorized("authentication required")
	}

	token, err := s.repos.Token().GetByID(ctx, tokenID)
	if err != nil {
		if isNotFound(err) {
			return model.User{}, errs.Unauthorized("invalid token")
		}
		return model.User{}, err
	}

	user, err := s.repos.User().GetByID(ctx, token.UserID)
	if err != nil {
		if isNotFound(err) {
			return model.User{}, errs.Unauthorized("invalid token")
		}
		return model.User{}, err
	}

	// best-effort: сбой обновления last_used_at не должен рушить аутентификацию.
	_ = s.repos.Token().UpdateLastUsed(ctx, token.ID)

	return user, nil
}

func (s *AuthService) sessionExpiry() time.Time {
	return time.Now().Add(s.cfg.SessionTTL)
}

// SessionCookieName возвращает имя cookie web-сессии из конфигурации.
func (s *AuthService) SessionCookieName() string {
	return s.cfg.SessionCookieName
}

// isNotFound сообщает, является ли err доменной ошибкой NotFound (из errs.DB).
func isNotFound(err error) bool {
	var appErr *errs.AppError
	return errors.As(err, &appErr) && appErr.Code == errs.CodeNotFound
}
