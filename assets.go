// Package gameshelf provides embedded assets (migrations, templates, static files).
package gameshelf

import "embed"

// MigrationsFS contains the SQL migration files.
//
//go:embed migrations
var MigrationsFS embed.FS

// TemplatesFS contains the HTML template files.
//
//go:embed all:templates
var TemplatesFS embed.FS

// StaticFS contains the static web assets (JS games, etc).
//
//go:embed all:static
var StaticFS embed.FS
