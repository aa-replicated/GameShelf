package api

import (
	"io"
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
	background := r.FormValue("background_color")
	fontFamily := r.FormValue("font_family")

	if name == "" || primary == "" || secondary == "" || background == "" || fontFamily == "" {
		http.Error(w, "all fields required", http.StatusBadRequest)
		return
	}
	if !hexColorRE.MatchString(primary) || !hexColorRE.MatchString(secondary) || !hexColorRE.MatchString(background) {
		http.Error(w, "invalid color format (use #RRGGBB)", http.StatusBadRequest)
		return
	}
	allowedFonts := map[string]bool{"system": true, "serif": true, "mono": true}
	if !allowedFonts[fontFamily] {
		http.Error(w, "invalid font family", http.StatusBadRequest)
		return
	}
	if err := db.UpdateSiteBranding(s.db, name, primary, secondary, background, fontFamily); err != nil {
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

// GET /logo — serve the stored logo image
func (s *Server) logoHandler(w http.ResponseWriter, r *http.Request) {
	data, contentType, err := db.GetLogo(s.db)
	if err != nil || len(data) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data) //nolint:errcheck
}

// POST /admin/logo — upload a new logo image
func (s *Server) uploadLogoHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2MB
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "file too large (max 2MB)", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("logo")
	if err != nil {
		http.Error(w, "logo file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	allowed := map[string]bool{
		"image/png":     true,
		"image/jpeg":    true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
	}
	if !allowed[contentType] {
		http.Error(w, "unsupported image type (use PNG, JPEG, GIF, WebP, or SVG)", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("admin: read logo: %v", err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	if err := db.UpdateLogo(s.db, data, contentType); err != nil {
		log.Printf("admin: update logo: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	redirectURL := "/admin"
	if token := r.URL.Query().Get("token"); token != "" {
		redirectURL = "/admin?token=" + url.QueryEscape(token)
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
