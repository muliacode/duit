package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type DB struct {
	*sql.DB
}

// Open creates/opens the SQLite file at path and applies the schema. Schema
// statements all use CREATE TABLE IF NOT EXISTS / INSERT OR IGNORE, so Open
// is safe to call on every app launch — there is no separate migration
// runner yet. If you need real migrations later (renaming/dropping columns),
// introduce a schema_version table and a versioned migrations/ folder; don't
// grow this file with ad-hoc ALTER TABLE statements.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite allows only one writer at a time; capping the pool avoids
	// "database is locked" errors under concurrent Wails calls.
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &DB{sqlDB}, nil
}

// OpenInMemory is used by tests: a fresh, isolated database per test.
func OpenInMemory() (*DB, error) {
	return Open(":memory:")
}
