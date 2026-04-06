package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

type submitScoreRequest struct {
	Game       string `json:"game"`
	PlayerName string `json:"player_name"`
	Score      int    `json:"score"`
}

type submitScoreResponse struct {
	Rank  int64  `json:"rank"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// POST /api/scores — submit a score { game, player_name, score }
func (s *Server) submitScoreHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req submitScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(submitScoreResponse{Error: "invalid JSON"})
		return
	}
	if req.Game == "" || req.PlayerName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(submitScoreResponse{Error: "game and player_name required"})
		return
	}

	// Persist to PostgreSQL
	playerID, err := db.FindOrCreatePlayer(s.db, req.PlayerName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(submitScoreResponse{Error: "db error"})
		return
	}
	if _, err := db.InsertScore(s.db, playerID, req.Game, req.Score); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(submitScoreResponse{Error: "db error"})
		return
	}

	// Update Redis leaderboard
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	rank, _ := s.lb.AddScore(ctx, req.Game, req.PlayerName, req.Score)

	json.NewEncoder(w).Encode(submitScoreResponse{OK: true, Rank: rank})
}

// GET /api/scores/:slug — leaderboard data as JSON
func (s *Server) getScoresHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	entries, err := s.lb.TopScores(ctx, slug, 50)
	if err != nil {
		// Fallback to DB
		dbScores, dbErr := db.GetTopScores(s.db, slug, 50)
		if dbErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "unavailable"})
			return
		}
		entries = make([]leaderboard.Entry, len(dbScores))
		for i, sc := range dbScores {
			entries[i] = leaderboard.Entry{Rank: i + 1, PlayerName: sc.PlayerName, Score: sc.Score}
		}
	}
	json.NewEncoder(w).Encode(entries)
}
