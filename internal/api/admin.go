package api

import (
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/go-chi/chi/v5"
)

var hexColorRE = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// GET /admin — admin panel
func (s *Server) adminHandler(w http.ResponseWriter, r *http.Request) {
	games, err := db.GetAllGames(s.db)
	if err != nil {
		log.Printf("admin: get all games: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	allScores, err := db.GetAllScores(s.db)
	if err != nil {
		log.Printf("admin: get all scores: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	data := s.pageBase(r)
	data.PageTitle = "Admin Panel"
	data.AllGames = games
	data.DBScores = allScores
	data.Token = r.URL.Query().Get("token") // preserve token for form actions
	s.render(w, "admin.html", data)
}

// POST /admin/games/:slug/toggle — enable or disable a game
func (s *Server) toggleGameHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := db.ToggleGame(s.db, slug); err != nil {
		log.Printf("admin: toggle game %s: %v", slug, err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	redirectURL := "/admin"
	if token := r.URL.Query().Get("token"); token != "" {
		redirectURL = "/admin?token=" + url.QueryEscape(token)
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// POST /admin/branding — update site branding
func (s *Server) updateBrandingHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	primary := r.FormValue("primary_color")
	secondary := r.FormValue("secondary_color")

	if name == "" || primary == "" || secondary == "" {
		http.Error(w, "all fields required", http.StatusBadRequest)
		return
	}
	if !hexColorRE.MatchString(primary) || !hexColorRE.MatchString(secondary) {
		http.Error(w, "invalid color format (use #RRGGBB)", http.StatusBadRequest)
		return
	}
	if err := db.UpdateSiteBranding(s.db, name, primary, secondary); err != nil {
		log.Printf("admin: update branding: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	redirectURL := "/admin"
	if token := r.URL.Query().Get("token"); token != "" {
		redirectURL = "/admin?token=" + url.QueryEscape(token)
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
