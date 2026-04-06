package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

// PageData is passed to every template render.
type PageData struct {
	SiteName       string
	PrimaryColor   string
	SecondaryColor string
	PageTitle      string
	Token          string // admin token, preserved across form POSTs
	// Page-specific (only one populated per page)
	Games    []db.Game
	Game     *db.Game
	Scores   []leaderboard.Entry
	DBScores []db.Score
	AllGames []db.Game
	Site     *db.Site
}

// pageBase fills the branding fields from the DB.
func (s *Server) pageBase(r *http.Request) PageData {
	site, err := db.GetSite(s.db)
	if err != nil || site == nil {
		return PageData{
			SiteName:       s.cfg.SiteName,
			PrimaryColor:   "#3B82F6",
			SecondaryColor: "#1E40AF",
		}
	}
	return PageData{
		SiteName:       site.Name,
		PrimaryColor:   site.PrimaryColor,
		SecondaryColor: site.SecondaryColor,
		Site:           site,
	}
}

// render executes a named template with the given data.
func (s *Server) render(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// GET / — game library landing page
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	games, err := db.GetEnabledGames(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	data := s.pageBase(r)
	data.PageTitle = "Game Library"
	data.Games = games
	s.render(w, "index.html", data)
}

// GET /games/:slug — play a specific game
func (s *Server) gameHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	game, err := db.GetGame(s.db, slug)
	if err != nil || game == nil || !game.Enabled {
		http.NotFound(w, r)
		return
	}
	data := s.pageBase(r)
	data.PageTitle = game.Name
	data.Game = game
	s.render(w, "game.html", data)
}

// GET /leaderboard/:slug — leaderboard for a game
func (s *Server) leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	game, err := db.GetGame(s.db, slug)
	if err != nil || game == nil {
		http.NotFound(w, r)
		return
	}
	entries, err := s.lb.TopScores(r.Context(), slug, 50)
	if err != nil {
		// Fallback to DB if Redis unavailable
		dbScores, _ := db.GetTopScores(s.db, slug, 50)
		entries = make([]leaderboard.Entry, len(dbScores))
		for i, sc := range dbScores {
			entries[i] = leaderboard.Entry{Rank: i + 1, PlayerName: sc.PlayerName, Score: sc.Score}
		}
	}
	data := s.pageBase(r)
	data.PageTitle = game.Name + " Leaderboard"
	data.Game = game
	data.Scores = entries
	s.render(w, "leaderboard.html", data)
}
