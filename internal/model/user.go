package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// User — глобальная идентичность (schema core.user).
type User struct {
	ID           string       `db:"id"`
	Email        string       `db:"email"`
	PasswordHash string       `db:"password_hash"`
	DisplayName  string       `db:"display_name"`
	Settings     UserSettings `db:"settings"`
	CreatedAt    time.Time    `db:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"`
	DeletedAt    *time.Time   `db:"deleted_at"`
}

// UserSettings — глобальные (per-user) настройки, хранятся в core.user.settings
// (jsonb). current_workspace_id — «текущий» workspace для редиректа при входе без
// явного workspace в URL. FK у поля нет (лежит в jsonb), поэтому валидность
// проверяется на чтении: если id не входит в актуальный список workspace
// пользователя — фронт делает фолбэк на первый.
type UserSettings struct {
	CurrentWorkspaceID *string `json:"current_workspace_id,omitempty"`
}

// Scan реализует sql.Scanner: читает jsonb-колонку в UserSettings. NULL/пусто →
// нулевые настройки.
func (s *UserSettings) Scan(src any) error {
	if src == nil {
		*s = UserSettings{}
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("model: unsupported source type for UserSettings.Scan")
	}

	if len(data) == 0 {
		*s = UserSettings{}
		return nil
	}
	return json.Unmarshal(data, s)
}

// Value реализует driver.Valuer: сериализует UserSettings в jsonb.
func (s UserSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}
