package server

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestMain acquires a PostgreSQL advisory lock (7337741002) for the entire
// server test suite so cross-package DELETE cleanup races with db/agent
// packages are avoided when running `go test ./...`.
func TestMain(m *testing.M) {
	dsn, _, err := db.DSN()
	if err != nil {
		os.Exit(m.Run())
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil || conn.Ping() != nil {
		os.Exit(m.Run())
	}
	defer conn.Close()
	if _, err := conn.Exec(`SELECT pg_advisory_lock(7337741002)`); err != nil {
		os.Exit(m.Run())
	}
	defer conn.Exec(`SELECT pg_advisory_unlock(7337741002)`) //nolint:errcheck
	os.Exit(m.Run())
}

// requireTestPG skips the calling test when no database is configured (neither
// ARTEX_PG_DSN nor a config file), matching db.testDSN, so `go test ./...` passes
// on a machine or CI job without Postgres. A configured but unreachable database
// still fails the test.
func requireTestPG(t *testing.T) {
	t.Helper()
	if _, _, err := db.DSN(); err != nil {
		t.Skipf("no database config (%v) — skipping", err)
	}
}

// testDataDir returns a scratch Manager data dir whose cleanup tolerates files
// written after the test returns. Many tests build a Server with a background
// context and never stop its engine, so worker/archive goroutines can still be
// writing under tasks/ while cleanup runs; with t.TempDir that made RemoveAll
// fail otherwise-passing tests with "directory not empty" (more often under CPU
// load, e.g. on CI runners).
//
// TODO: the real fix is for Server.Close to cancel its context and wait for the
// engine goroutines, and for tests to call it. This only stops the flake.
func testDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "artex-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for attempt := 0; attempt < 10; attempt++ {
			if os.RemoveAll(dir) == nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("leaving test data dir %s: background goroutines still writing", dir)
	})
	return dir
}
