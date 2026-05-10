package queue

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

var testDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..")

	net, err := tcnetwork.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer net.Remove(ctx)

	pgc, err := tcpostgres.Run(ctx,
		"postgres:16",
		tcpostgres.WithDatabase("telive"),
		tcpostgres.WithUsername("telive"),
		tcpostgres.WithPassword("telive"),
		tcnetwork.WithNetwork([]string{"postgres"}, net),
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

	lbc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    projectRoot,
				Dockerfile: "Dockerfile.liquibase",
			},
			Networks: []string{net.Name},
			Cmd: []string{
				"--url=jdbc:postgresql://postgres:5432/telive?sslmode=disable",
				"--username=telive",
				"--password=telive",
				"--defaultSchemaName=telive",
				"--liquibaseSchemaName=public",
				"--search-path=/liquibase/changelog",
				"--changeLogFile=root.yaml",
				"update",
			},
			WaitingFor: wait.ForExit(),
		},
		Started: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer lbc.Terminate(ctx)

	state, err := lbc.State(ctx)
	if err != nil || state.ExitCode != 0 {
		if logs, lerr := lbc.Logs(ctx); lerr == nil {
			io.Copy(os.Stderr, logs)
		}
		log.Fatal("liquibase migration failed")
	}

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

func insertSong(t *testing.T, q *PgQueue, title, artist string) int {
	t.Helper()
	var id int
	err := q.db.QueryRow(
		`INSERT INTO songs (title, artist) VALUES ($1, $2) RETURNING id`,
		title, artist,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestAdd_InvalidSongID(t *testing.T) {
	q := openTestQueue(t)

	err := q.Add("Dan", []int{999999})
	assert.ErrorIs(t, err, ErrInvalidSongID)
}

func TestAdd_ValidSongID(t *testing.T) {
	q := openTestQueue(t)
	songID := insertSong(t, q, "Test Song", "Test Artist")
	t.Cleanup(func() {
		q.db.Exec(`DELETE FROM songs WHERE id = $1`, songID)
		q.db.Exec(`DELETE FROM signups WHERE name = 'Dan'`)
	})

	err := q.Add("Dan", []int{songID})
	assert.NoError(t, err)
}

func TestAdd_TwoValidSongIDs(t *testing.T) {
	q := openTestQueue(t)
	song1 := insertSong(t, q, "Test Song 1", "Test Artist")
	song2 := insertSong(t, q, "Test Song 2", "Test Artist")
	t.Cleanup(func() {
		q.db.Exec(`DELETE FROM songs WHERE id = ANY($1)`, []int{song1, song2})
		q.db.Exec(`DELETE FROM signups WHERE name = 'Dan'`)
	})

	err := q.Add("Dan", []int{song1, song2})
	assert.NoError(t, err)
}

func TestAdd_ThreeValidSongIDs(t *testing.T) {
	q := openTestQueue(t)
	song1 := insertSong(t, q, "Test Song 1", "Test Artist")
	song2 := insertSong(t, q, "Test Song 2", "Test Artist")
	song3 := insertSong(t, q, "Test Song 3", "Test Artist")
	t.Cleanup(func() {
		q.db.Exec(`DELETE FROM songs WHERE id = ANY($1)`, []int{song1, song2, song3})
		q.db.Exec(`DELETE FROM signups WHERE name = 'Dan'`)
	})

	err := q.Add("Dan", []int{song1, song2, song3})
	assert.NoError(t, err)
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
