package db

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

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

// Migrate runs all embedded SQL migrations in lexicographic order.
func Migrate(d *sql.DB, fs embed.FS) error {
	entries, err := fs.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := fs.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}
		if _, err := d.Exec(string(data)); err != nil {
			return fmt.Errorf("running migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}
