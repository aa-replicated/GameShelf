package api

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gameshelf/gameshelf/internal/config"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

// Server holds all application dependencies.
type Server struct {
	db       *sql.DB
	lb       *leaderboard.Client
	tmpl     *template.Template
	staticFS fs.FS
	cfg      config.Config
}

// NewServer constructs a Server, parsing templates from the embedded FS.
func NewServer(db *sql.DB, lb *leaderboard.Client, templatesFS embed.FS, staticFS embed.FS, cfg config.Config) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	// Strip the "static" prefix so URL /static/games/snake.js maps to games/snake.js in the FS
	stripped, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static fs: %w", err)
	}
	return &Server{db: db, lb: lb, tmpl: tmpl, staticFS: stripped, cfg: cfg}, nil
}

// Handler builds and returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Static files served at /static/*
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(s.staticFS))))

	// Public pages
	r.Get("/", s.indexHandler)
	r.Get("/games/{slug}", s.gameHandler)
	r.Get("/leaderboard/{slug}", s.leaderboardHandler)

	// Score API
	r.Post("/api/scores", s.submitScoreHandler)
	r.Get("/api/scores/{slug}", s.getScoresHandler)

	// Admin (protected)
	r.Group(func(r chi.Router) {
		r.Use(s.adminAuthMiddleware)
		r.Get("/admin", s.adminHandler)
		r.Post("/admin/games/{slug}/toggle", s.toggleGameHandler)
		r.Post("/admin/branding", s.updateBrandingHandler)
	})

	// Health
	r.Get("/healthz", s.healthHandler)

	return r
}
