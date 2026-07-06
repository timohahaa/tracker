package migrations

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres-драйвер (lib/pq)
	_ "github.com/golang-migrate/migrate/v4/source/file"       // чтение миграций с ФС
)

// Config — параметры запуска миграций.
type Config struct {
	// DSN подключения к PostgreSQL, например:
	// postgres://user:pass@localhost:5432/dbname?sslmode=disable
	DSN string

	// Path — путь до директории с .sql-файлами (относительно рабочей директории
	// процесса). По умолчанию "migrations/sql".
	Path string

	// Attempts — сколько раз пытаться подключиться к БД (0 = 1 попытка).
	Attempts int

	// Timeout — пауза между попытками.
	Timeout time.Duration

	// Logf — опциональный логгер (например log.Printf / logrus.Infof).
	Logf func(format string, args ...any)
}

func (c *Config) withDefaults() {
	if c.Path == "" {
		c.Path = "migrations/sql"
	}
	if c.Attempts <= 0 {
		c.Attempts = 1
	}
	if c.Timeout <= 0 {
		c.Timeout = time.Second
	}
	if c.Logf == nil {
		c.Logf = func(string, ...any) {}
	}
}

// Up применяет все доступные миграции вверх.
func Up(cfg Config) error {
	return run(cfg, func(m *migrate.Migrate) error {
		return m.Up()
	})
}

// Down откатывает все миграции вниз (полный откат схемы).
func Down(cfg Config) error {
	return run(cfg, func(m *migrate.Migrate) error {
		return m.Down()
	})
}

// Steps применяет n шагов: n>0 — вверх, n<0 — вниз.
func Steps(cfg Config, n int) error {
	return run(cfg, func(m *migrate.Migrate) error {
		return m.Steps(n)
	})
}

// Version возвращает текущую версию миграции и флаг dirty.
func Version(cfg Config) (version uint, dirty bool, err error) {
	cfg.withDefaults()
	m, err := newMigrate(cfg)
	if err != nil {
		return 0, false, err
	}
	defer closeMigrate(m, cfg)
	return m.Version()
}

// run — общая обвязка: создаёт migrate-инстанс, выполняет действие,
// корректно обрабатывает ErrNoChange и закрывает ресурсы.
func run(cfg Config, action func(*migrate.Migrate) error) error {
	cfg.withDefaults()

	m, err := newMigrate(cfg)
	if err != nil {
		return err
	}
	defer closeMigrate(m, cfg)

	err = action(m)
	if errors.Is(err, migrate.ErrNoChange) {
		cfg.Logf("[migrate]: no change")
		return nil
	}
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	cfg.Logf("[migrate]: success")
	return nil
}

// newMigrate собирает migrate.Migrate из file-источника и postgres-драйвера,
// с ретраями подключения к БД.
func newMigrate(cfg Config) (*migrate.Migrate, error) {
	sourceURL := "file://" + cfg.Path

	var (
		m   *migrate.Migrate
		err error
	)
	attempts := cfg.Attempts

	for {
		m, err = migrate.New(sourceURL, cfg.DSN)
		if err == nil {
			break
		}
		attempts--
		if attempts <= 0 {
			return nil, fmt.Errorf("migrate: connect: %w", err)
		}
		cfg.Logf("[migrate]: connecting, attempts left: %d", attempts)
		time.Sleep(cfg.Timeout)
	}

	return m, nil
}

func closeMigrate(m *migrate.Migrate, cfg Config) {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		cfg.Logf("[migrate]: close source: %v", srcErr)
	}
	if dbErr != nil {
		cfg.Logf("[migrate]: close db: %v", dbErr)
	}
}
