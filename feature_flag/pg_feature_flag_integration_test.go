package feature_flag

import (
	"context"
	"database/sql"
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

	_ "github.com/jackc/pgx/v5/stdlib"
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

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", testDSN)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEnabled_TrueFlag(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO telive.feature_flags (key, enabled) VALUES ('test_flag', true)`)
	require.NoError(t, err)
	t.Cleanup(func() { db.Exec(`DELETE FROM telive.feature_flags WHERE key = 'test_flag'`) })

	ff := New(db)
	assert.True(t, ff.Enabled("test_flag"))
}

func TestEnabled_FalseFlag(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO telive.feature_flags (key, enabled) VALUES ('disabled_flag', false)`)
	require.NoError(t, err)
	t.Cleanup(func() { db.Exec(`DELETE FROM telive.feature_flags WHERE key = 'disabled_flag'`) })

	ff := New(db)
	assert.False(t, ff.Enabled("disabled_flag"))
}

func TestEnabled_MissingKey(t *testing.T) {
	ff := New(openTestDB(t))
	assert.False(t, ff.Enabled("nonexistent_flag"))
}