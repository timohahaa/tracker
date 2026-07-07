//go:build integration

package dbtest

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SeedUser вставляет строку в core.user с уникальным email (чтобы был
// безопасен t.Parallel()) и возвращает её id. Удаление регистрируется через
// t.Cleanup сразу после INSERT — не течёт даже при панике теста.
func SeedUser(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	email := "u-" + uuid.NewString() + "@test.local"
	var id string
	err := db.Get(&id, `
		INSERT INTO core.user (email, password_hash, display_name)
		VALUES ($1, 'x', $1) RETURNING id`, email)
	if err != nil {
		t.Fatalf("seed user: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, id) })
	return id
}

// SeedWorkspace вставляет org.workspace с уникальным slug и возвращает её id.
func SeedWorkspace(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	slug := "ws-" + uuid.NewString()
	var id string
	err := db.Get(&id, `
		INSERT INTO org.workspace (name, slug)
		VALUES ($1, $2) RETURNING id`, slug, slug)
	if err != nil {
		t.Fatalf("seed workspace: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.workspace WHERE id = $1`, id) })
	return id
}

// SeedWorkspaceMember добавляет участника workspace с указанной ролью и
// уникальным username. Возвращает сгенерированный username.
func SeedWorkspaceMember(t *testing.T, db *sqlx.DB, workspaceID, userID, role string) string {
	t.Helper()
	username := "m-" + uuid.NewString()
	_, err := db.Exec(`
		INSERT INTO org.workspace_member (workspace_id, user_id, role, username)
		VALUES ($1, $2, $3, $4)`, workspaceID, userID, role, username)
	if err != nil {
		t.Fatalf("seed workspace member: %s", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM org.workspace_member WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID)
	})
	return username
}

// SeedSession создаёт активную сессию (expires_at через час) и возвращает её id.
func SeedSession(t *testing.T, db *sqlx.DB, userID string) string {
	t.Helper()
	var id string
	err := db.Get(&id, `
		INSERT INTO core.session (user_id, expires_at)
		VALUES ($1, $2) RETURNING id`, userID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("seed session: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.session WHERE id = $1`, id) })
	return id
}

// SeedAPIToken создаёт API-токен для пользователя в workspace и возвращает его id
// (он же Bearer-токен).
func SeedAPIToken(t *testing.T, db *sqlx.DB, userID, workspaceID string) string {
	t.Helper()
	var id string
	err := db.Get(&id, `
		INSERT INTO core.api_token (user_id, workspace_id, name)
		VALUES ($1, $2, $3) RETURNING id`, userID, workspaceID, "test-token")
	if err != nil {
		t.Fatalf("seed api token: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.api_token WHERE id = $1`, id) })
	return id
}
