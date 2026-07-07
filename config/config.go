package config

import "time"

// Config — конфигурация приложения. Парсится из env через go-arg.
type Config struct {
	// PostgresDSN — строка подключения к PostgreSQL, например:
	// postgres://user:pass@localhost:5432/dbname?sslmode=disable
	PostgresDSN string `arg:"required,--,env:POSTGRES_DSN"`

	// HTTPAddr — адрес, на котором слушает HTTP-сервер.
	HTTPAddr string `arg:"--,env:HTTP_ADDR"`

	// MigrationsPath — путь до директории с .sql-миграциями.
	MigrationsPath string `arg:"--,env:MIGRATIONS_PATH"`

	// SessionTTL — время жизни web-сессии.
	SessionTTL time.Duration `arg:"--,env:SESSION_TTL"`

	// Debug — включает debug-уровень логирования.
	Debug bool `arg:"--,env:DEBUG"`
}

// SetDefaults проставляет значения по умолчанию для необязательных полей.
func (c *Config) SetDefaults() {
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	if c.MigrationsPath == "" {
		c.MigrationsPath = "migrations/sql"
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = 30 * 24 * time.Hour
	}
}
