package queue

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

var testDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, thisFile, _, _ := runtime.Caller(0)
	changesDir := filepath.Join(filepath.Dir(thisFile), "..", "db", "changelog", "changes")

	pgc, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithDatabase("telive"),
		tcpostgres.WithUsername("telive"),
		tcpostgres.WithPassword("telive"),
		tcpostgres.WithInitScripts(
			filepath.Join(changesDir, "001-initial-schema.sql"),
			filepath.Join(changesDir, "002-seed-songs.sql"),
			filepath.Join(changesDir, "003-settings.sql"),
			filepath.Join(changesDir, "004-host-users.sql"),
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := pgc.Terminate(ctx); err != nil {
			log.Printf("terminate postgres container: %v", err)
		}
	}()

	testDSN, err = pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func openTestQueue(t *testing.T) *PgQueue {
	t.Helper()
	q, err := NewPgQueue(testDSN)
	require.NoError(t, err)
	return q
}

func insertHostUser(t *testing.T, q *PgQueue, label, passcode string, active bool) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(passcode), 12)
	require.NoError(t, err)
	var id int
	err = q.db.QueryRow(
		`INSERT INTO host_users (label, passcode_hash, active) VALUES ($1, $2, $3) RETURNING id`,
		label, string(hash), active,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestAuthenticateHost_CorrectPasscode(t *testing.T) {
	q := openTestQueue(t)
	id := insertHostUser(t, q, "dan", "correct-code", true)
	t.Cleanup(func() { q.db.Exec(`DELETE FROM host_users WHERE id = $1`, id) })

	assert.True(t, q.AuthenticateHost("correct-code"))
}

func TestAuthenticateHost_WrongPasscode(t *testing.T) {
	q := openTestQueue(t)
	id := insertHostUser(t, q, "dan", "correct-code", true)
	t.Cleanup(func() { q.db.Exec(`DELETE FROM host_users WHERE id = $1`, id) })

	assert.False(t, q.AuthenticateHost("wrong-code"))
}

func TestAuthenticateHost_InactiveUser(t *testing.T) {
	q := openTestQueue(t)
	id := insertHostUser(t, q, "dan", "correct-code", false)
	t.Cleanup(func() { q.db.Exec(`DELETE FROM host_users WHERE id = $1`, id) })

	assert.False(t, q.AuthenticateHost("correct-code"))
}

func TestToggleSignups_OpenClearsOldSignups(t *testing.T) {
	q := openTestQueue(t)

	_, err := q.db.Exec(`UPDATE settings SET value = 'false' WHERE key = 'signups_open'`)
	require.NoError(t, err)

	var oldID, todayID int
	err = q.db.QueryRow(`
		INSERT INTO signups (name, position, created_at)
		VALUES ('yesterday-singer', 1.0, NOW() - INTERVAL '1 day')
		RETURNING id`).Scan(&oldID)
	require.NoError(t, err)

	err = q.db.QueryRow(`
		INSERT INTO signups (name, position, created_at)
		VALUES ('today-singer', 2.0, NOW())
		RETURNING id`).Scan(&todayID)
	require.NoError(t, err)

	t.Cleanup(func() {
		q.db.Exec(`DELETE FROM signups WHERE id = ANY($1)`, []int{oldID, todayID})
		q.db.Exec(`UPDATE settings SET value = 'false' WHERE key = 'signups_open'`)
	})

	open := q.ToggleSignups()
	assert.True(t, open)

	var count int
	err = q.db.QueryRow(`SELECT COUNT(*) FROM signups WHERE id = $1`, oldID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "yesterday's signup should be deleted")

	err = q.db.QueryRow(`SELECT COUNT(*) FROM signups WHERE id = $1`, todayID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "today's signup should be preserved")
}

func TestToggleSignups_CloseDoesNotClearSignups(t *testing.T) {
	q := openTestQueue(t)

	_, err := q.db.Exec(`UPDATE settings SET value = 'true' WHERE key = 'signups_open'`)
	require.NoError(t, err)

	var oldID int
	err = q.db.QueryRow(`
		INSERT INTO signups (name, position, created_at)
		VALUES ('yesterday-singer', 1.0, NOW() - INTERVAL '1 day')
		RETURNING id`).Scan(&oldID)
	require.NoError(t, err)

	t.Cleanup(func() {
		q.db.Exec(`DELETE FROM signups WHERE id = $1`, oldID)
		q.db.Exec(`UPDATE settings SET value = 'false' WHERE key = 'signups_open'`)
	})

	open := q.ToggleSignups()
	assert.False(t, open)

	var count int
	err = q.db.QueryRow(`SELECT COUNT(*) FROM signups WHERE id = $1`, oldID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "closing signups should not delete old signups")
}
