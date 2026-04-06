package db

import (
	"database/sql"
	"fmt"
	"time"
)

// --- Site ---

func GetSite(d *sql.DB) (*Site, error) {
	row := d.QueryRow(`SELECT id, name, COALESCE(logo_url,''), primary_color, secondary_color, created_at, updated_at FROM sites LIMIT 1`)
	var s Site
	if err := row.Scan(&s.ID, &s.Name, &s.LogoURL, &s.PrimaryColor, &s.SecondaryColor, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, fmt.Errorf("get site: %w", err)
	}
	return &s, nil
}

func UpdateSiteBranding(d *sql.DB, name, primaryColor, secondaryColor string) error {
	_, err := d.Exec(`UPDATE sites SET name=$1, primary_color=$2, secondary_color=$3, updated_at=NOW() WHERE id=(SELECT id FROM sites LIMIT 1)`,
		name, primaryColor, secondaryColor)
	return err
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

// FindOrCreatePlayer looks up a player by display_name, creating one if not found.
func FindOrCreatePlayer(d *sql.DB, displayName string) (int, error) {
	var id int
	err := d.QueryRow(`SELECT id FROM players WHERE display_name=$1 ORDER BY created_at DESC LIMIT 1`, displayName).Scan(&id)
	if err == sql.ErrNoRows {
		err = d.QueryRow(`INSERT INTO players (display_name) VALUES ($1) RETURNING id`, displayName).Scan(&id)
	}
	if err != nil {
		return 0, fmt.Errorf("find/create player: %w", err)
	}
	return id, nil
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
