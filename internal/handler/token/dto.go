package token

import (
	"time"

	"github.com/timohahaa/tracker/internal/model"
)

// createTokenRequest — тело POST /tokens. name обязателен.
type createTokenRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

// createTokenResponse — ответ POST /tokens. token содержит сырой Bearer-токен
// (равный id) и отдаётся ровно один раз — при создании; в списке его нет.
type createTokenResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// tokenResponse — публичное представление токена в списке. Секрет (token) здесь
// не сериализуется никогда; id нужен клиенту для отзыва (DELETE).
type tokenResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// tokenListResponse — конверт ответа списка токенов.
type tokenListResponse struct {
	Data []tokenResponse `json:"data"`
}

func toCreateTokenResponse(t model.APIToken) createTokenResponse {
	return createTokenResponse{
		ID:        t.ID,
		Token:     t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
}

func toTokenResponse(t model.APIToken) tokenResponse {
	return tokenResponse{
		ID:         t.ID,
		Name:       t.Name,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
	}
}

func toTokenListResponse(tokens []model.APIToken) tokenListResponse {
	data := make([]tokenResponse, 0, len(tokens))
	for _, t := range tokens {
		data = append(data, toTokenResponse(t))
	}
	return tokenListResponse{Data: data}
}
