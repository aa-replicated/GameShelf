package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
)

// MigrationsFS is set from cmd/gameshelf/assets.go via go:embed.
var MigrationsFS embed.FS

// Connect opens a PostgreSQL connection and verifies it with a ping.
func Connect(databaseURL string) (*sql.DB, error) {
	d, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(5)
	return d, nil
}

// Migrate runs all embedded SQL migrations idempotently.
func Migrate(d *sql.DB) error {
	data, err := MigrationsFS.ReadFile("migrations/001_schema.sql")
	if err != nil {
		return fmt.Errorf("reading migration: %w", err)
	}
	if _, err := d.Exec(string(data)); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	return nil
}
