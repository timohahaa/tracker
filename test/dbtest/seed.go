//go:build integration

package dbtest

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SeedUser вставляет строку в core.user с уникальным email (чтобы был
// безопасен t.Parallel()) и возвращает её id. Удаление регистрируется через
// t.Cleanup сразу после INSERT — не течёт даже при панике теста.
func SeedUser(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	email := "u-" + uuid.NewString() + "@test.local"
	var id string
	err := db.Get(&id, `
		INSERT INTO core.user (email, password_hash, display_name)
		VALUES ($1, 'x', $1) RETURNING id`, email)
	if err != nil {
		t.Fatalf("seed user: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, id) })
	return id
}

// SeedWorkspace вставляет org.workspace с уникальным slug и возвращает её id.
func SeedWorkspace(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	slug := "ws-" + uuid.NewString()
	var id string
	err := db.Get(&id, `
		INSERT INTO org.workspace (name, slug)
		VALUES ($1, $2) RETURNING id`, slug, slug)
	if err != nil {
		t.Fatalf("seed workspace: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.workspace WHERE id = $1`, id) })
	return id
}

// SeedWorkspaceMember добавляет участника workspace с указанной ролью и
// уникальным username. Возвращает сгенерированный username.
func SeedWorkspaceMember(t *testing.T, db *sqlx.DB, workspaceID, userID, role string) string {
	t.Helper()
	username := "m-" + uuid.NewString()
	_, err := db.Exec(`
		INSERT INTO org.workspace_member (workspace_id, user_id, role, username)
		VALUES ($1, $2, $3, $4)`, workspaceID, userID, role, username)
	if err != nil {
		t.Fatalf("seed workspace member: %s", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM org.workspace_member WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID)
	})
	return username
}

// SeedTeam вставляет org.team с уникальным key и возвращает её id.
func SeedTeam(t *testing.T, db *sqlx.DB, workspaceID string) string {
	t.Helper()
	key := "T" + uuid.NewString()[:8]
	var id string
	err := db.Get(&id, `
		INSERT INTO org.team (workspace_id, name, key)
		VALUES ($1, $2, $3) RETURNING id`, workspaceID, key, key)
	if err != nil {
		t.Fatalf("seed team: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, id) })
	return id
}

// SeedTeamMember добавляет участника команды с указанной ролью.
func SeedTeamMember(t *testing.T, db *sqlx.DB, teamID, userID, role string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO org.team_member (team_id, user_id, role)
		VALUES ($1, $2, $3)`, teamID, userID, role)
	if err != nil {
		t.Fatalf("seed team member: %s", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM org.team_member WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	})
}

// SeedStatus вставляет tracker.status в команду и возвращает её id. Позиция
// уникальна внутри (team_id, group) — вызывающий отвечает за уникальность.
func SeedStatus(t *testing.T, db *sqlx.DB, teamID, name, group string, position int) string {
	t.Helper()
	var id string
	err := db.Get(&id, `
		INSERT INTO tracker.status (team_id, name, status_group, position)
		VALUES ($1, $2, $3, $4) RETURNING id`, teamID, name, group, position)
	if err != nil {
		t.Fatalf("seed status: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, id) })
	return id
}

// SeedIssue вставляет tracker.issue в команду с указанным статусом и возвращает
// её id и выданный issue_number. Номер вычисляется как max+1 в пределах команды
// (уникальность per team), чтобы фабрику можно было звать несколько раз подряд.
func SeedIssue(t *testing.T, db *sqlx.DB, teamID, statusID string) (string, int) {
	t.Helper()
	var out struct {
		ID     string `db:"id"`
		Number int    `db:"issue_number"`
	}
	err := db.Get(&out, `
		INSERT INTO tracker.issue (team_id, status_id, title, issue_number)
		VALUES (
			$1
			, $2
			, 'seed issue'
			, (SELECT COALESCE(MAX(issue_number), 0) + 1 FROM tracker.issue WHERE team_id = $1)
		)
		RETURNING id, issue_number`, teamID, statusID)
	if err != nil {
		t.Fatalf("seed issue: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, out.ID) })
	return out.ID, out.Number
}

// IssueSeed — параметры расширенного сидера задачи. Пустые поля берут дефолты.
type IssueSeed struct {
	Title       string
	Description string
	Priority    string     // none/urgent/high/medium/low; пусто → none
	DueDate     *time.Time // nil → без дедлайна
}

// SeedIssueDetailed вставляет tracker.issue с управляемыми priority/title/
// description/due_date и возвращает её id и выданный issue_number (max+1 per
// team). Нужен для list-тестов: фильтрация/поиск/сортировка по разным полям.
func SeedIssueDetailed(t *testing.T, db *sqlx.DB, teamID, statusID string, seed IssueSeed) (string, int) {
	t.Helper()
	if seed.Title == "" {
		seed.Title = "seed issue " + uuid.NewString()
	}
	if seed.Priority == "" {
		seed.Priority = "none"
	}
	var out struct {
		ID     string `db:"id"`
		Number int    `db:"issue_number"`
	}
	err := db.Get(&out, `
		INSERT INTO tracker.issue (team_id, status_id, title, description, priority, due_date, issue_number)
		VALUES (
			$1
			, $2
			, $3
			, $4
			, $5
			, $6
			, (SELECT COALESCE(MAX(issue_number), 0) + 1 FROM tracker.issue WHERE team_id = $1)
		)
		RETURNING id, issue_number`, teamID, statusID, seed.Title, seed.Description, seed.Priority, seed.DueDate)
	if err != nil {
		t.Fatalf("seed issue detailed: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, out.ID) })
	return out.ID, out.Number
}

// SeedLabel вставляет tracker.label. teamID == nil → workspace-level метка,
// иначе team-level. Возвращает её id. Имя/цвет уникальны на тест.
func SeedLabel(t *testing.T, db *sqlx.DB, workspaceID string, teamID *string) string {
	t.Helper()
	name := "label-" + uuid.NewString()
	var id string
	err := db.Get(&id, `
		INSERT INTO tracker.label (workspace_id, team_id, name, color)
		VALUES ($1, $2, $3, '#3B82F6') RETURNING id`, workspaceID, teamID, name)
	if err != nil {
		t.Fatalf("seed label: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.label WHERE id = $1`, id) })
	return id
}

// SeedIssueAssignee добавляет участника задачи (is_primary → ответственный).
func SeedIssueAssignee(t *testing.T, db *sqlx.DB, issueID, userID string, isPrimary bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO tracker.issue_assignee (issue_id, user_id, is_primary)
		VALUES ($1, $2, $3)`, issueID, userID, isPrimary)
	if err != nil {
		t.Fatalf("seed issue assignee: %s", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.issue_assignee WHERE issue_id = $1 AND user_id = $2`, issueID, userID)
	})
}

// SeedIssueLabel привязывает метку к задаче (junction tracker.issue_label).
func SeedIssueLabel(t *testing.T, db *sqlx.DB, issueID, labelID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO tracker.issue_label (issue_id, label_id)
		VALUES ($1, $2)`, issueID, labelID)
	if err != nil {
		t.Fatalf("seed issue label: %s", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.issue_label WHERE issue_id = $1 AND label_id = $2`, issueID, labelID)
	})
}

// SeedSession создаёт активную сессию (expires_at через час) и возвращает её id.
func SeedSession(t *testing.T, db *sqlx.DB, userID string) string {
	t.Helper()
	var id string
	err := db.Get(&id, `
		INSERT INTO core.session (user_id, expires_at)
		VALUES ($1, $2) RETURNING id`, userID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("seed session: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.session WHERE id = $1`, id) })
	return id
}

// SeedAPIToken создаёт API-токен для пользователя в workspace и возвращает его id
// (он же Bearer-токен).
func SeedAPIToken(t *testing.T, db *sqlx.DB, userID, workspaceID string) string {
	t.Helper()
	var id string
	err := db.Get(&id, `
		INSERT INTO core.api_token (user_id, workspace_id, name)
		VALUES ($1, $2, $3) RETURNING id`, userID, workspaceID, "test-token")
	if err != nil {
		t.Fatalf("seed api token: %s", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.api_token WHERE id = $1`, id) })
	return id
}
