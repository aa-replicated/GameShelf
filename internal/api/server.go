package api

import (
	"bytes"
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
	tmpls    map[string]*template.Template
	staticFS fs.FS
	cfg      config.Config
}

// pageNames lists the templates that can be rendered.
var pageNames = []string{"index.html", "game.html", "leaderboard.html", "admin.html"}

// NewServer constructs a Server, parsing each page template together with base.html.
func NewServer(db *sql.DB, lb *leaderboard.Client, templatesFS embed.FS, staticFS embed.FS, cfg config.Config) (*Server, error) {
	tmpls := make(map[string]*template.Template, len(pageNames))
	for _, page := range pageNames {
		t, err := template.ParseFS(templatesFS, "templates/base.html", "templates/"+page)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}
		tmpls[page] = t
	}
	stripped, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static fs: %w", err)
	}
	return &Server{db: db, lb: lb, tmpls: tmpls, staticFS: stripped, cfg: cfg}, nil
}

// render executes a named template with buffering to prevent partial responses.
func (s *Server) render(w http.ResponseWriter, name string, data PageData) {
	t, ok := s.tmpls[name]
	if !ok {
		http.Error(w, "unknown template: "+name, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w) //nolint:errcheck
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
