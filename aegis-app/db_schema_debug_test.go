package main

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureSchemaOnRemoteDB(t *testing.T) {
	dbPath := "remote_b_1.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("database file %s not found, skipping", dbPath)
	}

	src, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read source db: %v", err)
	}
	tmpFile := "test_schema_debug.db"
	defer os.Remove(tmpFile)
	if err := os.WriteFile(tmpFile, src, 0644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}

	db, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err = db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if _, err = db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}

	app := &App{}
	if err := app.ensureSchema(db); err != nil {
		t.Fatalf("ensureSchema failed: %v", err)
	}
	t.Log("ensureSchema succeeded")

	app.db = db
	identity, err := app.getLocalIdentity()
	if err != nil {
		t.Fatalf("getLocalIdentity failed: %v", err)
	}
	t.Logf("Identity loaded: pubkey=%s", identity.PublicKey)

	// Verify notifications table has all required columns
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE is_read = 0`).Scan(&count); err != nil {
		t.Fatalf("query notifications is_read: %v", err)
	}
	t.Logf("Unread notifications: %d", count)
}
