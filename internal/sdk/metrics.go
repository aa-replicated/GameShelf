package sdk

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// Metric is a single key/value pair for the custom metrics API.
type Metric struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// metricsPayload is the request body for PATCH /api/v1/app/custom-metrics.
type metricsPayload struct {
	Data []Metric `json:"data"`
}

// ReportMetrics sends the provided metrics to the SDK sidecar.
func (c *Client) ReportMetrics(ctx context.Context, metrics []Metric) error {
	payload := metricsPayload{Data: metrics}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.patch(ctx, "/api/v1/app/custom-metrics", bytes.NewReader(b))
}

// RunMetricsLoop starts a background goroutine that queries the database every
// interval and reports custom metrics to the SDK. It exits when ctx is cancelled.
func (c *Client) RunMetricsLoop(ctx context.Context, db *sql.DB, interval time.Duration) {
	if !c.Available() {
		return
	}
	log.Printf("sdk: metrics loop started (interval: %v)", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.collectAndReport(ctx, db); err != nil {
					log.Printf("sdk: metrics report error: %v", err)
				}
			}
		}
	}()
}

// collectAndReport queries DB counters and sends them to the SDK.
func (c *Client) collectAndReport(ctx context.Context, db *sql.DB) error {
	var scoresSubmitted int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scores`).Scan(&scoresSubmitted); err != nil {
		return err
	}
	var gamesPlayed int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT game_slug) FROM scores`).Scan(&gamesPlayed); err != nil {
		return err
	}
	var activePlayers int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT player_id) FROM scores
		WHERE played_at > NOW() - INTERVAL '24 hours'`).Scan(&activePlayers); err != nil {
		return err
	}
	var activeGames int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT game_slug) FROM scores
		WHERE played_at > NOW() - INTERVAL '24 hours'`).Scan(&activeGames); err != nil {
		return err
	}

	metrics := []Metric{
		{Key: "games_played_total", Value: gamesPlayed},
		{Key: "scores_submitted_total", Value: scoresSubmitted},
		{Key: "active_players_24h", Value: activePlayers},
		{Key: "active_games", Value: activeGames},
	}
	if err := c.ReportMetrics(ctx, metrics); err != nil {
		return err
	}
	log.Printf("sdk: metrics reported: games_played_total=%d scores_submitted_total=%d active_players_24h=%d active_games=%d",
		gamesPlayed, scoresSubmitted, activePlayers, activeGames)
	return nil
}
