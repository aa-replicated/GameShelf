package db

import (
	"database/sql"
	"fmt"
	"time"
)

// --- Site ---

func GetSite(d *sql.DB) (*Site, error) {
	row := d.QueryRow(`
		SELECT id, name, primary_color, secondary_color,
		       background_color, font_family,
		       logo_data IS NOT NULL AS has_logo,
		       created_at, updated_at
		FROM sites LIMIT 1`)
	var s Site
	if err := row.Scan(&s.ID, &s.Name, &s.PrimaryColor, &s.SecondaryColor,
		&s.BackgroundColor, &s.FontFamily, &s.HasLogo,
		&s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, fmt.Errorf("get site: %w", err)
	}
	return &s, nil
}

func UpdateSiteBranding(d *sql.DB, name, primaryColor, secondaryColor, backgroundColor, fontFamily string) error {
	_, err := d.Exec(`
		UPDATE sites SET
			name=$1, primary_color=$2, secondary_color=$3,
			background_color=$4, font_family=$5, updated_at=NOW()
		WHERE id=(SELECT id FROM sites LIMIT 1)`,
		name, primaryColor, secondaryColor, backgroundColor, fontFamily)
	return err
}

// UpdateLogo stores a logo image for the site.
func UpdateLogo(d *sql.DB, data []byte, contentType string) error {
	_, err := d.Exec(`
		UPDATE sites SET logo_data=$1, logo_content_type=$2, updated_at=NOW()
		WHERE id=(SELECT id FROM sites LIMIT 1)`,
		data, contentType)
	return err
}

// GetLogo retrieves the stored logo bytes and content type.
// Returns nil data if no logo is set.
func GetLogo(d *sql.DB) ([]byte, string, error) {
	var data []byte
	var contentType string
	err := d.QueryRow(`SELECT logo_data, COALESCE(logo_content_type,'') FROM sites LIMIT 1`).
		Scan(&data, &contentType)
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

// --- Games ---

func GetEnabledGames(d *sql.DB) ([]Game, error) {
	rows, err := d.Query(`SELECT id, slug, name, COALESCE(description,''), enabled, min_players, max_players, created_at FROM games WHERE enabled=true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

func GetAllGames(d *sql.DB) ([]Game, error) {
	rows, err := d.Query(`SELECT id, slug, name, COALESCE(description,''), enabled, min_players, max_players, created_at FROM games ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

func GetGame(d *sql.DB, slug string) (*Game, error) {
	row := d.QueryRow(`SELECT id, slug, name, COALESCE(description,''), enabled, min_players, max_players, created_at FROM games WHERE slug=$1`, slug)
	var g Game
	if err := row.Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.Enabled, &g.MinPlayers, &g.MaxPlayers, &g.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func ToggleGame(d *sql.DB, slug string) error {
	_, err := d.Exec(`UPDATE games SET enabled = NOT enabled WHERE slug=$1`, slug)
	return err
}

func scanGames(rows *sql.Rows) ([]Game, error) {
	var games []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.Enabled, &g.MinPlayers, &g.MaxPlayers, &g.CreatedAt); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

// --- Scores ---

// FindOrCreatePlayer upserts a player by display_name and returns the id.
func FindOrCreatePlayer(d *sql.DB, displayName string) (int, error) {
	var id int
	err := d.QueryRow(`
		INSERT INTO players (display_name)
		VALUES ($1)
		ON CONFLICT (display_name) DO UPDATE SET display_name = EXCLUDED.display_name
		RETURNING id
	`, displayName).Scan(&id)
	return id, err
}

// InsertScore saves a score record and returns the new score's ID.
func InsertScore(d *sql.DB, playerID int, gameSlug string, score int) (int, error) {
	var id int
	err := d.QueryRow(`INSERT INTO scores (player_id, game_slug, score) VALUES ($1,$2,$3) RETURNING id`, playerID, gameSlug, score).Scan(&id)
	return id, err
}

// GetTopScores returns the top N scores for a game, joined with player names.
func GetTopScores(d *sql.DB, gameSlug string, limit int) ([]Score, error) {
	rows, err := d.Query(`
		SELECT s.id, s.player_id, p.display_name, s.game_slug, s.score, s.played_at
		FROM scores s
		JOIN players p ON p.id = s.player_id
		WHERE s.game_slug = $1
		ORDER BY s.score DESC, s.played_at ASC
		LIMIT $2`, gameSlug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScores(rows)
}

// GetAllScores returns all scores for admin view.
func GetAllScores(d *sql.DB) ([]Score, error) {
	rows, err := d.Query(`
		SELECT s.id, s.player_id, p.display_name, s.game_slug, s.score, s.played_at
		FROM scores s
		JOIN players p ON p.id = s.player_id
		ORDER BY s.played_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanScores(rows)
}

func scanScores(rows *sql.Rows) ([]Score, error) {
	var scores []Score
	for rows.Next() {
		var s Score
		var playedAt time.Time
		if err := rows.Scan(&s.ID, &s.PlayerID, &s.PlayerName, &s.GameSlug, &s.Score, &playedAt); err != nil {
			return nil, err
		}
		s.PlayedAt = playedAt
		scores = append(scores, s)
	}
	return scores, rows.Err()
}
