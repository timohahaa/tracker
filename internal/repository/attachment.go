package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// AttachmentRepo — доступ к tracker.attachment.
type AttachmentRepo struct {
	db Ext
}

// attachmentSelectColumns — общий список колонок привязки.
const attachmentSelectColumns = `
	id
	, issue_id
	, integration_id
	, source_type
	, url
	, title
	, metadata
	, created_at
	, updated_at
	, deleted_at`

// Upsert создаёт или обновляет привязку внешнего ресурса к задаче по
// (url, issue_id). ON CONFLICT восстанавливает soft-deleted строку и обновляет
// title/metadata/integration_id — идемпотентно при повторной доставке webhook.
// metadata jsonb передаётся строкой с ::jsonb-кастом (lib/pq иначе шлёт []byte
// как bytea).
func (r *AttachmentRepo) Upsert(ctx context.Context, a model.Attachment) (model.Attachment, error) {
	const q = `
		INSERT INTO tracker.attachment (
			issue_id
			, integration_id
			, source_type
			, url
			, title
			, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		ON CONFLICT (url, issue_id) WHERE deleted_at IS NULL DO UPDATE SET
			title = EXCLUDED.title
			, metadata = EXCLUDED.metadata
			, integration_id = EXCLUDED.integration_id
			, deleted_at = NULL
			, updated_at = now()
		RETURNING` + attachmentSelectColumns

	var out model.Attachment
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		a.IssueID, a.IntegrationID, a.SourceType, a.URL, a.Title, string(a.Metadata),
	); err != nil {
		return model.Attachment{}, errs.DB(err)
	}
	return out, nil
}

// ListByIssue возвращает активные привязки задачи. Сортировка стабильна
// (created_at, id).
func (r *AttachmentRepo) ListByIssue(ctx context.Context, issueID string) ([]model.Attachment, error) {
	const q = `
		SELECT` + attachmentSelectColumns + `
		FROM tracker.attachment
		WHERE issue_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, id`

	var out []model.Attachment
	if err := sqlx.SelectContext(ctx, r.db, &out, q, issueID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}
