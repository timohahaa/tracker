package service

import (
	"context"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// validateLabelSet валидирует набор меток для replace-семантики POST/PATCH:
// каждая метка существует в workspace и видима задаче — либо workspace-level
// (TeamID == nil), либо принадлежит команде задачи (TeamID == teamID). Чужая
// team-метка или несуществующая → errs.Validation. Набор дедуплицируется с
// сохранением порядка. workspaceID/teamID передаются напрямую (а не через
// issue), т.к. CreateIssue валидирует до появления задачи.
func (s *IssueService) validateLabelSet(ctx context.Context, workspaceID, teamID string, labelIDs []string) ([]string, error) {
	seen := make(map[string]bool, len(labelIDs))
	out := make([]string, 0, len(labelIDs))
	for _, id := range labelIDs {
		if seen[id] {
			continue
		}
		seen[id] = true

		label, err := s.repos.Label().GetByID(ctx, workspaceID, id)
		if err != nil {
			if isNotFound(err) {
				return nil, errs.Validation("invalid label", errs.FieldError{
					Field:   "label_ids",
					Reason:  "invalid",
					Message: "must be workspace-level labels or labels of the issue team",
				})
			}
			return nil, err
		}
		if label.TeamID != nil && *label.TeamID != teamID {
			return nil, errs.Validation("invalid label", errs.FieldError{
				Field:   "label_ids",
				Reason:  "invalid",
				Message: "must be workspace-level labels or labels of the issue team",
			})
		}
		out = append(out, id)
	}
	return out, nil
}

// replaceIssueLabels приводит набор активных меток задачи к целевому (set) в
// рамках переданной транзакции: soft-delete отсутствующих, upsert набора
// (восстановление soft-deleted). Набор считается уже провалидированным
// (validateLabelSet). По образцу replaceProjectTeams/replaceAssignees.
func replaceIssueLabels(ctx context.Context, tx repository.Store, issueID string, set []string) error {
	current, err := tx.Issue().ListLabels(ctx, issueID)
	if err != nil {
		return err
	}

	want := make(map[string]bool, len(set))
	for _, id := range set {
		want[id] = true
	}

	// Отвязать те метки, которых нет в новом наборе.
	for _, cur := range current {
		if !want[cur.ID] {
			if rerr := tx.Issue().RemoveIssueLabel(ctx, issueID, cur.ID); rerr != nil {
				return rerr
			}
		}
	}

	// Привязать новый набор (восстановление soft-deleted через upsert).
	for _, labelID := range set {
		if aerr := tx.Issue().AddIssueLabel(ctx, issueID, labelID); aerr != nil {
			return aerr
		}
	}
	return nil
}

// enrichLabels догружает активные метки задачи (read-модель IssueDetails.Labels
// заполняется на уровне сервиса, репо-мапперы её не трогают — как Assignees).
func (s *IssueService) enrichLabels(ctx context.Context, d *model.IssueDetails) error {
	labels, err := s.repos.Issue().ListLabels(ctx, d.ID)
	if err != nil {
		return err
	}
	d.Labels = labels
	return nil
}

// enrichListLabels догружает метки для набора задач одним запросом (без N+1) и
// раскладывает их по строкам.
func (s *IssueService) enrichListLabels(ctx context.Context, issues []model.IssueDetails) error {
	if len(issues) == 0 {
		return nil
	}
	ids := make([]string, len(issues))
	for i := range issues {
		ids[i] = issues[i].ID
	}
	byIssue, err := s.repos.Issue().ListLabelsByIssues(ctx, ids)
	if err != nil {
		return err
	}
	for i := range issues {
		issues[i].Labels = byIssue[issues[i].ID]
	}
	return nil
}
