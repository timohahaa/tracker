package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// IntegrationRepo — доступ к integration.connection.
type IntegrationRepo struct {
	db Ext
}

// integrationSelectColumns — общий список колонок интеграции.
const integrationSelectColumns = `
	id
	, workspace_id
	, provider
	, settings
	, credentials
	, created_at
	, updated_at
	, deleted_at`

// Upsert создаёт или переподключает интеграцию workspace по (workspace_id,
// provider). ON CONFLICT восстанавливает soft-deleted строку (deleted_at=NULL)
// и обновляет settings/credentials — дубля по партиал-уникальному индексу
// (workspace_id, provider) WHERE deleted_at IS NULL не возникает. settings/
// credentials jsonb передаются как строка с явным ::jsonb-кастом (lib/pq иначе
// шлёт []byte как bytea).
func (r *IntegrationRepo) Upsert(ctx context.Context, i model.Integration) (model.Integration, error) {
	const q = `
		INSERT INTO integration.connection (
			workspace_id
			, provider
			, settings
			, credentials
		)
		VALUES ($1, $2, $3::jsonb, $4::jsonb)
		ON CONFLICT (workspace_id, provider) WHERE deleted_at IS NULL DO UPDATE SET
			settings = EXCLUDED.settings
			, credentials = EXCLUDED.credentials
			, deleted_at = NULL
			, updated_at = now()
		RETURNING` + integrationSelectColumns

	var out model.Integration
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		i.WorkspaceID, i.Provider, string(i.Settings), string(i.Credentials),
	); err != nil {
		return model.Integration{}, errs.DB(err)
	}
	return out, nil
}

// GetByWorkspaceProvider возвращает активную интеграцию workspace по провайдеру.
// Нет строки → errs.NotFound (через errs.DB).
func (r *IntegrationRepo) GetByWorkspaceProvider(ctx context.Context, workspaceID string, provider model.Provider) (model.Integration, error) {
	const q = `
		SELECT` + integrationSelectColumns + `
		FROM integration.connection
		WHERE workspace_id = $1 AND provider = $2 AND deleted_at IS NULL`

	var out model.Integration
	if err := sqlx.GetContext(ctx, r.db, &out, q, workspaceID, provider); err != nil {
		return model.Integration{}, errs.DB(err)
	}
	return out, nil
}

// GetByID возвращает активную интеграцию по id без workspace-скоупа. Нужен
// webhook-пути, где workspace резолвится уже из самой интеграции. Нет строки →
// errs.NotFound (через errs.DB).
func (r *IntegrationRepo) GetByID(ctx context.Context, id string) (model.Integration, error) {
	const q = `
		SELECT` + integrationSelectColumns + `
		FROM integration.connection
		WHERE id = $1 AND deleted_at IS NULL`

	var out model.Integration
	if err := sqlx.GetContext(ctx, r.db, &out, q, id); err != nil {
		return model.Integration{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет интеграцию (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой. Attachments остаются
// как исторические данные.
func (r *IntegrationRepo) Delete(ctx context.Context, id string) error {
	const q = `
		UPDATE integration.connection
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return errs.DB(err)
	}
	return nil
}
