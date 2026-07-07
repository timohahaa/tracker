// Package dbtest содержит общий код для integration-тестов на реальной БД:
// фасад подключения (Open) и seed-фабрики. Собственно тестовые хелперы лежат
// в файлах под build-тегом integration; этот файл без тега нужен, чтобы
// `go test ./...` без -tags=integration не падал
package dbtest
