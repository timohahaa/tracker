package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gosimple/slug"

	"github.com/timohahaa/tracker/internal/errs"
)

// slugCandidates — сколько числовых суффиксов пробуем перед фолбэком на рандом.
const slugCandidates = 100

// slugify нормализует строку в URL-slug: транслитерирует Unicode в ASCII
// (кириллица/диакритика), приводит к нижнему регистру, оставляет только [a-z0-9-]
// со схлопыванием и обрезкой дефисов. Может вернуть пустую строку (если вход не
// содержит транслитерируемых буквенно-цифровых символов, напр. эмодзи).
func slugify(s string) string {
	return slug.Make(s)
}

// uniqueSlug возвращает свободный slug на основе base. Пробует base, base-2,
// base-3, … пока taken не вернёт false; при исчерпании кандидатов добавляет
// короткий случайный суффикс. taken инкапсулирует проверку в БД (инъекция —
// чтобы функция оставалась тестируемой без БД).
func uniqueSlug(ctx context.Context, base string, taken func(context.Context, string) (bool, error)) (string, error) {
	slug := slugify(base)
	if slug == "" {
		slug = "workspace"
	}

	candidate := slug
	for i := 1; i <= slugCandidates; i++ {
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", slug, i)
		}
		exists, err := taken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	// Крайне маловероятный фолбэк: почти-гарантированно уникальный суффикс.
	candidate = fmt.Sprintf("%s-%s", slug, randomSuffix())
	if exists, err := taken(ctx, candidate); err != nil {
		return "", err
	} else if exists {
		return "", errs.Internal("could not generate unique workspace slug")
	}
	return candidate, nil
}

// defaultUsername генерирует username участника из display name (или, если он
// пустой, из local-part email). Гарантированно непустой.
func defaultUsername(displayName, email string) string {
	if u := slugify(displayName); u != "" {
		return u
	}
	local, _, _ := strings.Cut(email, "@")
	if u := slugify(local); u != "" {
		return u
	}
	return "user"
}

func randomSuffix() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
