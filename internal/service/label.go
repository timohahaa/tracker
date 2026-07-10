package service

import (
	"context"
	"regexp"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// LabelService — CRUD меток (workspace- и team-level) с правилами уникальности
// и merge.
type LabelService struct {
	repos repository.Store
}

// hexColorRe — формат цвета метки: '#RRGGBB' (6 hex-цифр). Вынесен в чистый
// хелпер isValidHexColor, чтобы покрыть юнит-тестом без похода в БД.
var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// isValidHexColor проверяет, что строка — hex-цвет вида '#RRGGBB'.
func isValidHexColor(s string) bool {
	return hexColorRe.MatchString(s)
}

// CreateLabelParams — входные данные создания метки (граница service).
// TeamID == nil → workspace-level метка, иначе team-level.
type CreateLabelParams struct {
	TeamID *string
	Name   string
	Color  string
}

// UpdateLabelParams — частичное обновление метки (name/color). Nil-поле означает
// «не менять».
type UpdateLabelParams struct {
	Name  *string
	Color *string
}

// CreateLabel создаёт метку. Доступно любому участнику workspace (membership
// гарантирован middleware LoadWorkspace). Правила (docs/tech/data-model.md):
//   - team-level: команда обязана принадлежать workspace; нельзя создать
//     team-метку с именем, которое уже занято на workspace-level → errs.AlreadyExists
//     (409); дубль внутри команды ловит партиал-индекс → errs.AlreadyExists (409).
//   - workspace-level: если существуют одноимённые team-метки — merge (создать
//     ws-метку, перепривязать issue_label со старых меток, старые пометить
//     удалёнными) атомарно в транзакции; иначе обычная вставка (дубль ws-level
//     → 409).
func (s *LabelService) CreateLabel(ctx context.Context, actor model.WorkspaceMember, params CreateLabelParams) (model.Label, error) {
	if err := validateColor(params.Color); err != nil {
		return model.Label{}, err
	}

	if params.TeamID != nil {
		return s.createTeamLabel(ctx, actor.WorkspaceID, *params.TeamID, params)
	}
	return s.createWorkspaceLabel(ctx, actor.WorkspaceID, params)
}

// createTeamLabel создаёт team-level метку после проверки принадлежности команды
// workspace и запрета затенения одноимённой workspace-level метки.
func (s *LabelService) createTeamLabel(ctx context.Context, workspaceID, teamID string, params CreateLabelParams) (model.Label, error) {
	if err := s.validateTeam(ctx, workspaceID, teamID); err != nil {
		return model.Label{}, err
	}
	if err := s.ensureNoWorkspaceShadow(ctx, workspaceID, params.Name); err != nil {
		return model.Label{}, err
	}

	return s.repos.Label().Create(ctx, model.Label{
		WorkspaceID: workspaceID,
		TeamID:      &teamID,
		Name:        params.Name,
		Color:       params.Color,
	})
}

// createWorkspaceLabel создаёт workspace-level метку. При наличии одноимённых
// team-меток выполняет merge в одной транзакции.
func (s *LabelService) createWorkspaceLabel(ctx context.Context, workspaceID string, params CreateLabelParams) (model.Label, error) {
	teamLabels, err := s.repos.Label().ListTeamLabelsByName(ctx, workspaceID, params.Name)
	if err != nil {
		return model.Label{}, err
	}

	if len(teamLabels) == 0 {
		return s.repos.Label().Create(ctx, model.Label{
			WorkspaceID: workspaceID,
			Name:        params.Name,
			Color:       params.Color,
		})
	}

	var created model.Label
	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		var cerr error
		created, cerr = tx.Label().Create(ctx, model.Label{
			WorkspaceID: workspaceID,
			Name:        params.Name,
			Color:       params.Color,
		})
		if cerr != nil {
			return cerr
		}
		return repointAndDeleteTeamLabels(ctx, tx, labelIDs(teamLabels), created.ID)
	})
	if err != nil {
		return model.Label{}, err
	}
	return created, nil
}

// GetLabel возвращает метку workspace по id. Доступно участнику workspace.
// Нет метки → errs.NotFound.
func (s *LabelService) GetLabel(ctx context.Context, workspaceID, labelID string) (model.Label, error) {
	return s.repos.Label().GetByID(ctx, workspaceID, labelID)
}

// ListLabels возвращает все активные метки workspace (ws- и team-level).
// Доступно участнику workspace.
func (s *LabelService) ListLabels(ctx context.Context, workspaceID string) ([]model.Label, error) {
	return s.repos.Label().List(ctx, workspaceID)
}

// UpdateLabel частично обновляет name/color метки. Доступно участнику workspace.
// Кросс-уровневые правила симметричны CreateLabel и применяются только при
// реальной смене имени:
//   - team-level rename: нельзя затенять одноимённую ws-метку → errs.AlreadyExists
//     (409); дубль внутри своей команды ловит партиал-индекс → 409.
//   - workspace-level rename в имя, под которым есть team-метки — merge
//     (перепривязать issue_label на эту метку, старые team-метки пометить
//     удалёнными) атомарно в транзакции.
//
// Невалидный цвет → errs.Validation (400); нет метки → errs.NotFound (404).
func (s *LabelService) UpdateLabel(ctx context.Context, actor model.WorkspaceMember, labelID string, params UpdateLabelParams) (model.Label, error) {
	label, err := s.repos.Label().GetByID(ctx, actor.WorkspaceID, labelID)
	if err != nil {
		return model.Label{}, err
	}

	if params.Color != nil {
		if err := validateColor(*params.Color); err != nil {
			return model.Label{}, err
		}
		label.Color = *params.Color
	}

	// Кросс-уровневые правила проверяем только при фактической смене имени —
	// иначе PATCH одного цвета конфликтовал бы сам с собой.
	renaming := params.Name != nil && *params.Name != label.Name
	if !renaming {
		return s.repos.Label().Update(ctx, label)
	}
	label.Name = *params.Name

	if label.TeamID != nil {
		if err := s.ensureNoWorkspaceShadow(ctx, actor.WorkspaceID, label.Name); err != nil {
			return model.Label{}, err
		}
		return s.repos.Label().Update(ctx, label)
	}

	// workspace-level rename: слить одноимённые team-метки в эту метку.
	teamLabels, err := s.repos.Label().ListTeamLabelsByName(ctx, actor.WorkspaceID, label.Name)
	if err != nil {
		return model.Label{}, err
	}
	if len(teamLabels) == 0 {
		return s.repos.Label().Update(ctx, label)
	}

	var updated model.Label
	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		var uerr error
		updated, uerr = tx.Label().Update(ctx, label)
		if uerr != nil {
			return uerr
		}
		return repointAndDeleteTeamLabels(ctx, tx, labelIDs(teamLabels), updated.ID)
	})
	if err != nil {
		return model.Label{}, err
	}
	return updated, nil
}

// DeleteLabel мягко удаляет метку. Доступно участнику workspace.
func (s *LabelService) DeleteLabel(ctx context.Context, actor model.WorkspaceMember, labelID string) error {
	label, err := s.repos.Label().GetByID(ctx, actor.WorkspaceID, labelID)
	if err != nil {
		return err
	}
	return s.repos.Label().Delete(ctx, label.ID)
}

// ensureNoWorkspaceShadow проверяет, что имя не занято активной workspace-level
// меткой. Занято → errs.AlreadyExists (409). Единый код с обычным дублем: раз
// обратное направление (ws поверх team) разрешено через merge, коллизия
// team↔ws — это конфликт уникальности имени, а не ошибка ввода.
//
// TOCTOU: проверка read-then-write без блокировки; БД кросс-уровневые правила не
// гарантирует (партиал-индексы держат уникальность лишь внутри своего уровня).
// При конкурентном создании возможно появление team-метки рядом с одноимённой
// ws-меткой. Для MVP допустимо; надёжное решение — advisory-lock по
// (workspace_id, name) — вынесено в бэклог.
func (s *LabelService) ensureNoWorkspaceShadow(ctx context.Context, workspaceID, name string) error {
	if _, err := s.repos.Label().GetWorkspaceLabelByName(ctx, workspaceID, name); err == nil {
		return errs.AlreadyExists("a workspace-level label with this name already exists")
	} else if !isNotFound(err) {
		return err
	}
	return nil
}

// repointAndDeleteTeamLabels сливает team-метки в целевую: перепривязывает их
// issue_label на targetID и мягко удаляет сами метки. Вызывается в рамках
// merge-транзакции (create/update ws-метки).
//
// TOCTOU: набор team-меток вычислен до транзакции; team-метка, созданная во
// время merge, не будет перепривязана (см. комментарий ensureNoWorkspaceShadow).
func repointAndDeleteTeamLabels(ctx context.Context, tx repository.Store, oldIDs []string, targetID string) error {
	if rerr := tx.Label().RepointIssueLabels(ctx, oldIDs, targetID); rerr != nil {
		return rerr
	}
	for _, id := range oldIDs {
		if derr := tx.Label().Delete(ctx, id); derr != nil {
			return derr
		}
	}
	return nil
}

// labelIDs извлекает id из набора меток.
func labelIDs(labels []model.Label) []string {
	ids := make([]string, len(labels))
	for i, l := range labels {
		ids[i] = l.ID
	}
	return ids
}

// validateTeam проверяет, что команда принадлежит workspace. Иначе
// errs.Validation.
func (s *LabelService) validateTeam(ctx context.Context, workspaceID, teamID string) error {
	if _, err := s.repos.Team().Get(ctx, workspaceID, teamID); err != nil {
		if isNotFound(err) {
			return errs.Validation("invalid team", errs.FieldError{
				Field:   "team_id",
				Reason:  "invalid",
				Message: "must be a team of the workspace",
			})
		}
		return err
	}
	return nil
}

// validateColor проверяет формат цвета метки ('#RRGGBB'). Иначе errs.Validation.
func validateColor(color string) error {
	if !isValidHexColor(color) {
		return errs.Validation("invalid color", errs.FieldError{
			Field:   "color",
			Reason:  "invalid",
			Message: "must be a hex color like '#RRGGBB'",
		})
	}
	return nil
}
