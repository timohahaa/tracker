//go:build integration

package dbtest

import (
	"testing"

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
