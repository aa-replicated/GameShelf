package api

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

// PageData is passed to every template render.
type PageData struct {
	SiteName        string
	PrimaryColor    string
	SecondaryColor  string
	BackgroundColor string
	FontFamily      string
	HasLogo         bool
	PageTitle       string
	Token           string // admin token, preserved across form POSTs
	PlayerName      string // pre-filled player name from cookie / identity token
	// SDK-driven banners
	LicenseExpired      bool // true → red "license expired" banner
	LicenseExpiringSoon bool // true → yellow "expiring soon" banner (< 30 days)
	UpdateAvailable     bool // true → blue "update available" banner
	// Page-specific (only one populated per page)
	Games                []db.Game
	Game                 *db.Game
	Scores               []leaderboard.Entry
	DBScores             []db.Score
	AllGames             []db.Game
	Site                 *db.Site
	IdentitySecretMasked string // shown (masked) on admin panel
}

// pageBase fills the branding fields from the DB and SDK banner state.
func (s *Server) pageBase(r *http.Request) PageData {
	site, err := db.GetSite(s.db)
	var data PageData
	if err != nil || site == nil {
		data = PageData{
			SiteName:        s.cfg.SiteName,
			PrimaryColor:    "#3B82F6",
			SecondaryColor:  "#1E40AF",
			BackgroundColor: "#F9FAFB",
			FontFamily:      "system",
		}
	} else {
		data = PageData{
			SiteName:        site.Name,
			PrimaryColor:    site.PrimaryColor,
			SecondaryColor:  site.SecondaryColor,
			BackgroundColor: site.BackgroundColor,
			FontFamily:      site.FontFamily,
			HasLogo:         site.HasLogo,
			Site:            site,
		}
	}

	// Populate SDK banners (fail-open: errors are logged and ignored)
	if s.sdk.Available() {
		if info, err := s.sdk.GetLicenseInfo(r.Context()); err == nil && info != nil {
			if info.IsExpired {
				data.LicenseExpired = true
			} else if info.ExpirationDate != nil && time.Until(*info.ExpirationDate) < 30*24*time.Hour {
				data.LicenseExpiringSoon = true
			}
		}
		data.UpdateAvailable = s.sdk.HasUpdate(r.Context())
	}

	return data
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

	// Resolve player name: identity token takes precedence over existing cookie.
	playerName := getPlayerFromCookie(r)
	if token := r.URL.Query().Get("gs_identity"); token != "" {
		if secret, err := s.getOrCreateIdentitySecret(); err == nil {
			if name, ok := verifyIdentityToken(secret, token); ok {
				setPlayerCookie(w, name)
				playerName = name
			}
		}
	}

	data := s.pageBase(r)
	data.PageTitle = game.Name
	data.Game = game
	data.PlayerName = playerName
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
		log.Printf("leaderboard redis fallback for %s: %v", slug, err)
		// Fallback to DB if Redis unavailable
		dbScores, dbErr := db.GetTopScores(s.db, slug, 50)
		if dbErr != nil {
			log.Printf("leaderboard db fallback also failed for %s: %v", slug, dbErr)
		}
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
