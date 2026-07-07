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
	MigrationsPath string `arg:"required,--,env:MIGRATIONS_PATH"`

	// SessionTTL — время жизни web-сессии.
	SessionTTL time.Duration `arg:"--,env:SESSION_TTL"`

	// SessionCookieName — имя cookie web-сессии (см. docs/tech/auth.md).
	SessionCookieName string `arg:"--,env:SESSION_COOKIE_NAME"`

	// MaxOpenConns — макс. число открытых соединений с БД.
	MaxOpenConns int `arg:"--,env:DB_MAX_OPEN_CONNS"`

	// MaxIdleConns — макс. число idle-соединений в пуле.
	MaxIdleConns int `arg:"--,env:DB_MAX_IDLE_CONNS"`

	// ConnMaxLifetime — макс. время жизни соединения.
	ConnMaxLifetime time.Duration `arg:"--,env:DB_CONN_MAX_LIFETIME"`

	// Debug — включает debug-уровень логирования.
	Debug bool `arg:"--,env:DEBUG"`
}

// SetDefaults проставляет значения по умолчанию для необязательных полей.
func (c *Config) SetDefaults() {
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = 30 * 24 * time.Hour
	}
	if c.SessionCookieName == "" {
		c.SessionCookieName = "session_id"
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 25
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
}
