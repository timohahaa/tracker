//go:build integration

package dbtest

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Open открывает уже мигрированную тестовую БД из TEST_POSTGRES_DSN.
// Миграции применяются снаружи — в тестах migrations.Up не зовём.
// Фасад: позже за ним можно спрятать testcontainers без переписывания тестов.
func Open(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %s", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %s", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
