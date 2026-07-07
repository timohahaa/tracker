//go:build integration

package dbtest

import "testing"

// TestSmoke проверяет, что тест-инфраструктура работает end-to-end:
// подключение к БД, seed строки и её реальное существование.
func TestSmoke(t *testing.T) {
	t.Parallel()

	db := Open(t)
	id := SeedUser(t, db)

	var count int
	if err := db.Get(&count, `SELECT count(*) FROM core.user WHERE id = $1`, id); err != nil {
		t.Fatalf("count user: %s", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user with id %s, got %d", id, count)
	}
}
