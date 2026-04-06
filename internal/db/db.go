package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection.
func Connect(databaseURL string) (*sql.DB, error) {
	d, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(5)
	return d, nil
}

// Migrate runs the embedded SQL migration against the database.
func Migrate(d *sql.DB, fs embed.FS) error {
	data, err := fs.ReadFile("migrations/001_schema.sql")
	if err != nil {
		return fmt.Errorf("reading migration: %w", err)
	}
	if _, err := d.Exec(string(data)); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	return nil
}
