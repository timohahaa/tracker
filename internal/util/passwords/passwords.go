// Package passwords содержит утилиты хеширования и проверки паролей (bcrypt).
// Без зависимостей от БД; используется слайсом auth (Фаза 2.1).
package passwords

import "golang.org/x/crypto/bcrypt"

// Hash возвращает bcrypt-хеш пароля (cost по умолчанию).
func Hash(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify проверяет, соответствует ли пароль хешу.
func Verify(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
