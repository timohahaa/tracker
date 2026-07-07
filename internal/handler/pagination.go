package handler

import (
	"encoding/base64"
	"encoding/json"

	"github.com/timohahaa/tracker/internal/model"
)

// cursorPayload — сериализуемое представление cursor (sort value + id).
type cursorPayload struct {
	V  string `json:"v"`
	ID string `json:"id"`
}

// EncodeCursor кодирует sort value + id в base64-строку для next_cursor.
func EncodeCursor(value, id string) string {
	b, _ := json.Marshal(cursorPayload{V: value, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeCursor разбирает base64-cursor в model.Cursor.
func DecodeCursor(raw string) (*model.Cursor, error) {
	if raw == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var p cursorPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &model.Cursor{Value: p.V, ID: p.ID}, nil
}
