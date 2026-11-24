package app

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "migration.db"))
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	return db
}

func databaseVersion(t *testing.T, db *sql.DB) int {
	t.Helper()

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func TestMigrationsFreshDatabase(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := migrateDatabase(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if got := databaseVersion(t, db); got != len(schemaMigrations) {
		t.Fatalf("version = %d, want %d", got, len(schemaMigrations))
	}

	if err := migrateDatabase(db); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	if got := databaseVersion(t, db); got != len(schemaMigrations) {
		t.Fatalf("version after second run = %d", got)
	}
}

func TestMigrationVersionOneToTwo(t *testing.T) {
	db := openMigrationTestDB(t)

	migrations := []string{
		`CREATE TABLE items (id INTEGER PRIMARY KEY);`,
		`ALTER TABLE items ADD COLUMN name TEXT NOT NULL DEFAULT '';`,
	}

	if err := migrateDatabaseWithMigrations(db, migrations[:1]); err != nil {
		t.Fatal(err)
	}

	if got := databaseVersion(t, db); got != 1 {
		t.Fatalf("initial version = %d, want 1", got)
	}

	if err := migrateDatabaseWithMigrations(db, migrations); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if got := databaseVersion(t, db); got != 2 {
		t.Fatalf("upgraded version = %d, want 2", got)
	}

	var columnCount int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('items') WHERE name = 'name'",
	).Scan(&columnCount)
	if err != nil {
		t.Fatal(err)
	}

	if columnCount != 1 {
		t.Fatal("migration 2 did not add name column")
	}
}

func TestMigrationRollback(t *testing.T) {
	db := openMigrationTestDB(t)

	migrations := []string{
		`CREATE TABLE first_table (id INTEGER);`,
		`CREATE TABLE second_table (id INTEGER); INVALID SQL;`,
	}

	err := migrateDatabaseWithMigrations(db, migrations)
	if err == nil {
		t.Fatal("expected migration failure")
	}

	if got := databaseVersion(t, db); got != 1 {
		t.Fatalf("version after rollback = %d, want 1", got)
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='second_table'",
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatal("failed migration was not rolled back")
	}
}

func TestMigrationRejectsFutureVersion(t *testing.T) {
	db := openMigrationTestDB(t)

	if _, err := db.Exec("PRAGMA user_version = 999"); err != nil {
		t.Fatal(err)
	}

	err := migrateDatabase(db)
	if err == nil {
		t.Fatal("expected future database version to be rejected")
	}

	if !strings.Contains(err.Error(), "newer than supported version") {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := databaseVersion(t, db); got != 999 {
		t.Fatalf("future version changed to %d", got)
	}
}

func TestOpenStoreRejectsFutureSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")

	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("PRAGMA user_version = 999"); err != nil {
		db.Close()
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := OpenStore(path)
	if err == nil {
		if store != nil {
			store.Close()
		}
		t.Fatal("expected OpenStore to reject a future schema version")
	}

	if !strings.Contains(err.Error(), "newer than supported version") {
		t.Fatalf("unexpected error: %v", err)
	}
}
