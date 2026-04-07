package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	gameshelf "github.com/gameshelf/gameshelf"
	"github.com/gameshelf/gameshelf/internal/api"
	"github.com/gameshelf/gameshelf/internal/config"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	// Connect to PostgreSQL
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer database.Close()

	// Wait for DB with retry
	log.Println("Waiting for database to be ready...")
	if err := waitForDB(database, 30, 2*time.Second); err != nil {
		log.Fatalf("database never became available: %v", err)
	}
	log.Println("Database is ready.")

	// Run migrations
	if err := db.Migrate(database, gameshelf.MigrationsFS); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("Migrations applied.")

	// Connect to Redis
	lb, err := leaderboard.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("connecting to redis: %v", err)
	}
	log.Println("Waiting for Redis to be ready...")
	if err := waitForRedis(lb, 30, 2*time.Second); err != nil {
		log.Fatalf("redis never became available: %v", err)
	}
	log.Println("Redis is ready.")

	// Seed Redis leaderboards from PostgreSQL
	if err := seedLeaderboards(database, lb); err != nil {
		log.Printf("Warning: could not seed leaderboards: %v", err)
	}

	// Build HTTP server
	srv, err := api.NewServer(database, lb, gameshelf.TemplatesFS, gameshelf.StaticFS, cfg)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// waitForRedis retries pinging Redis until available.
func waitForRedis(lb *leaderboard.Client, maxRetries int, interval time.Duration) error {
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := lb.Ping(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("Redis not ready (attempt %d/%d): %v", i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("redis not ready after %d attempts", maxRetries)
}

// waitForDB retries pinging the database until available.
func waitForDB(d *sql.DB, maxRetries int, interval time.Duration) error {
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := d.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("DB not ready (attempt %d/%d): %v", i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("database not ready after %d attempts", maxRetries)
}

// seedLeaderboards loads PostgreSQL scores into Redis sorted sets on startup.
func seedLeaderboards(d *sql.DB, lb *leaderboard.Client) error {
	rows, err := d.Query(`
		SELECT s.game_slug, p.display_name, s.score, s.played_at
		FROM scores s
		JOIN players p ON p.id = s.player_id
		ORDER BY s.game_slug, s.played_at`)
	if err != nil {
		return err
	}
	defer rows.Close()

	byGame := make(map[string][]leaderboard.SeedRow)
	for rows.Next() {
		var r leaderboard.SeedRow
		if err := rows.Scan(&r.GameSlug, &r.PlayerName, &r.Score, &r.PlayedAt); err != nil {
			return err
		}
		byGame[r.GameSlug] = append(byGame[r.GameSlug], r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for gameSlug, seedRows := range byGame {
		if err := lb.SeedGame(ctx, gameSlug, seedRows); err != nil {
			log.Printf("Warning: seeding %s leaderboard: %v", gameSlug, err)
		}
	}
	log.Printf("Leaderboard seeding complete (%d games).", len(byGame))
	return nil
}
