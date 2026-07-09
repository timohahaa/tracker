package service

import (
	"context"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// AddAssignee добавляет участника задачи. Доступно участнику команды задачи
// (requireTeamMember). userID обязан быть активным участником команды задачи,
// иначе errs.Validation. Если primary — назначает ответственным в той же
// транзакции (ClearPrimary + SetPrimary; partial unique нельзя DEFERRABLE).
// Возвращает обновлённую задачу с активными assignees.
func (s *IssueService) AddAssignee(ctx context.Context, actor model.WorkspaceMember, ref, userID string, primary bool) (model.IssueDetails, error) {
	details, err := s.resolveIssue(ctx, actor.WorkspaceID, ref)
	if err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.requireTeamMember(ctx, details.TeamID, actor.UserID); err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.validateTeamMember(ctx, details.TeamID, userID); err != nil {
		return model.IssueDetails{}, err
	}

	if primary {
		err = s.repos.WithTx(ctx, func(tx repository.Store) error {
			if _, aerr := tx.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: details.ID, UserID: userID}); aerr != nil {
				return aerr
			}
			if cerr := tx.Issue().ClearPrimary(ctx, details.ID); cerr != nil {
				return cerr
			}
			return tx.Issue().SetPrimary(ctx, details.ID, userID)
		})
	} else {
		_, err = s.repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: details.ID, UserID: userID})
	}
	if err != nil {
		return model.IssueDetails{}, err
	}

	// Поля задачи не менялись — переиспользуем загруженный details, обновив только
	// набор assignees (без повторного GetByID).
	if err := s.enrichAssignees(ctx, &details); err != nil {
		return model.IssueDetails{}, err
	}
	return details, nil
}

// RemoveAssignee убирает участника задачи (soft delete, идемпотентно). Доступно
// участнику команды задачи. Снятие участника-ответственного оставляет задачу без
// ответственного (без авто-передачи). Обновлённый набор не возвращается —
// эндпоинт отдаёт 204, поэтому финальная догрузка задачи не нужна.
func (s *IssueService) RemoveAssignee(ctx context.Context, actor model.WorkspaceMember, ref, userID string) error {
	details, err := s.resolveIssue(ctx, actor.WorkspaceID, ref)
	if err != nil {
		return err
	}
	if err := s.requireTeamMember(ctx, details.TeamID, actor.UserID); err != nil {
		return err
	}
	return s.repos.Issue().RemoveAssignee(ctx, details.ID, userID)
}

// SetPrimary назначает участника ответственным за задачу. Доступно участнику
// команды задачи. userID обязан быть среди активных участников задачи, иначе
// errs.Validation. Смена идёт в одной транзакции (ClearPrimary + SetPrimary).
// Возвращает обновлённую задачу.
//
// Членство userID в команде здесь НЕ перепроверяется намеренно: ответственный
// выбирается из числа существующих активных assignees (инвариант
// «responsible ⊆ assignees»), а членство проверялось при добавлении участника.
func (s *IssueService) SetPrimary(ctx context.Context, actor model.WorkspaceMember, ref, userID string) (model.IssueDetails, error) {
	details, err := s.resolveIssue(ctx, actor.WorkspaceID, ref)
	if err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.requireTeamMember(ctx, details.TeamID, actor.UserID); err != nil {
		return model.IssueDetails{}, err
	}
	if _, err := s.repos.Issue().GetAssignee(ctx, details.ID, userID); err != nil {
		if isNotFound(err) {
			return model.IssueDetails{}, errs.Validation("responsible must be an assignee", errs.FieldError{
				Field:   "user_id",
				Reason:  "invalid",
				Message: "responsible must be one of the issue assignees",
			})
		}
		return model.IssueDetails{}, err
	}

	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		if cerr := tx.Issue().ClearPrimary(ctx, details.ID); cerr != nil {
			return cerr
		}
		return tx.Issue().SetPrimary(ctx, details.ID, userID)
	})
	if err != nil {
		return model.IssueDetails{}, err
	}

	// Поля задачи не менялись — переиспользуем details, обновив только assignees.
	if err := s.enrichAssignees(ctx, &details); err != nil {
		return model.IssueDetails{}, err
	}
	return details, nil
}

// validateTeamMember проверяет, что пользователь — активный участник команды.
// Не участник → errs.Validation (в отличие от requireTeamMember → Forbidden:
// здесь проверяется валидность назначаемого, а не право actor'а).
func (s *IssueService) validateTeamMember(ctx context.Context, teamID, userID string) error {
	if _, err := s.repos.Team().GetMember(ctx, teamID, userID); err != nil {
		if isNotFound(err) {
			return errs.Validation("invalid assignee", errs.FieldError{
				Field:   "user_id",
				Reason:  "invalid",
				Message: "must be an active member of the issue team",
			})
		}
		return err
	}
	return nil
}

// validateAssigneeSet валидирует набор assignees для replace-семантики PATCH:
// каждый — активный участник команды; максимум один is_primary. Возвращает
// провалидированный набор.
func (s *IssueService) validateAssigneeSet(ctx context.Context, teamID string, set []AssigneeInput) ([]AssigneeInput, error) {
	primaries := 0
	for _, a := range set {
		if err := s.validateTeamMember(ctx, teamID, a.UserID); err != nil {
			return nil, err
		}
		if a.IsPrimary {
			primaries++
		}
	}
	if primaries > 1 {
		return nil, errs.Validation("at most one responsible", errs.FieldError{
			Field:   "assignees",
			Reason:  "invalid",
			Message: "at most one assignee can be primary (responsible)",
		})
	}
	return set, nil
}

// replaceAssignees приводит набор активных участников задачи к целевому (set) в
// рамках переданной транзакции: soft-delete отсутствующих, upsert нового набора
// (is_primary=false), затем ровно один/ноль primary через ClearPrimary +
// SetPrimary. Набор считается уже провалидированным (validateAssigneeSet).
func replaceAssignees(ctx context.Context, tx repository.Store, issueID string, set []AssigneeInput) error {
	current, err := tx.Issue().ListAssignees(ctx, issueID)
	if err != nil {
		return err
	}

	want := make(map[string]bool, len(set))
	var primaryUser string
	for _, a := range set {
		want[a.UserID] = true
		if a.IsPrimary {
			primaryUser = a.UserID
		}
	}

	// Снять тех, кого нет в новом наборе.
	for _, cur := range current {
		if !want[cur.UserID] {
			if rerr := tx.Issue().RemoveAssignee(ctx, issueID, cur.UserID); rerr != nil {
				return rerr
			}
		}
	}

	// Upsert нового набора (восстановление soft-deleted), primary выставляем
	// отдельным шагом, поэтому здесь всегда is_primary=false.
	for _, a := range set {
		if _, aerr := tx.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: a.UserID}); aerr != nil {
			return aerr
		}
	}

	// Привести primary к ровно одному/нулю.
	if cerr := tx.Issue().ClearPrimary(ctx, issueID); cerr != nil {
		return cerr
	}
	if primaryUser != "" {
		return tx.Issue().SetPrimary(ctx, issueID, primaryUser)
	}
	return nil
}
