package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Redis  string `json:"redis"`
}

// GET /healthz — structured health check for Kubernetes probes.
// Returns 200 when healthy, 503 when either dependency is down.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "connected"
	if err := s.db.PingContext(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	}

	redisStatus := "connected"
	if err := s.lb.Ping(ctx); err != nil {
		redisStatus = "error: " + err.Error()
	}

	resp := healthResponse{
		Status: "ok",
		DB:     dbStatus,
		Redis:  redisStatus,
	}
	statusCode := http.StatusOK
	if dbStatus != "connected" || redisStatus != "connected" {
		resp.Status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
