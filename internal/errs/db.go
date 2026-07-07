package errs

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// pgUniqueViolation — код ошибки нарушения UNIQUE-ограничения в PostgreSQL.
const pgUniqueViolation = "23505"

// DB превращает ошибки драйвера БД (database/sql + lib/pq) в доменные ошибки.
// Repository вызывает errs.DB(err) при ошибках запросов.
func DB(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NotFound("resource not found")
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
		return AlreadyExists("resource already exists")
	}
	return err
}
