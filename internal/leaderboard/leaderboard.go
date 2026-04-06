package leaderboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const maxLeaderboardEntries = 1000

// Client wraps Redis for leaderboard operations.
type Client struct {
	rdb *redis.Client
}

// Entry is one leaderboard row.
type Entry struct {
	Rank       int
	PlayerName string
	Score      int
}

// SeedRow is used to load existing scores from PostgreSQL into Redis on startup.
type SeedRow struct {
	GameSlug   string
	PlayerName string
	Score      int
	PlayedAt   time.Time
}

// New creates a leaderboard Client from a Redis URL.
func New(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parsing redis URL: %w", err)
	}
	return &Client{rdb: redis.NewClient(opts)}, nil
}

// Ping checks Redis connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// AddScore adds a score to the sorted set and returns the 1-indexed rank (best=1).
// Member is "<nanosecond-timestamp>:<playerName>" to allow multiple entries per player.
func (c *Client) AddScore(ctx context.Context, gameSlug, playerName string, score int) (int64, error) {
	member := fmt.Sprintf("%d:%s", time.Now().UnixNano(), playerName)
	key := fmt.Sprintf("leaderboard:%s", gameSlug)

	if err := c.rdb.ZAdd(ctx, key, redis.Z{Score: float64(score), Member: member}).Err(); err != nil {
		return 0, fmt.Errorf("zadd: %w", err)
	}

	rank, err := c.rdb.ZRevRank(ctx, key, member).Result()
	if err != nil {
		return 1, nil // non-fatal: return 1 if rank lookup fails
	}

	// Trim to keep only top N entries
	c.rdb.ZRemRangeByRank(ctx, key, 0, -(maxLeaderboardEntries+1))

	return rank + 1, nil
}

// TopScores returns the top N entries for a game.
func (c *Client) TopScores(ctx context.Context, gameSlug string, limit int) ([]Entry, error) {
	key := fmt.Sprintf("leaderboard:%s", gameSlug)
	results, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange: %w", err)
	}
	entries := make([]Entry, 0, len(results))
	for i, r := range results {
		member, _ := r.Member.(string)
		// member format: "<timestamp>:<playerName>" — extract name after first colon
		name := member
		if idx := strings.Index(member, ":"); idx >= 0 {
			name = member[idx+1:]
		}
		entries = append(entries, Entry{
			Rank:       i + 1,
			PlayerName: name,
			Score:      int(r.Score),
		})
	}
	return entries, nil
}

// SeedGame loads scores for one game into Redis sorted set (used on startup).
func (c *Client) SeedGame(ctx context.Context, gameSlug string, rows []SeedRow) error {
	if len(rows) == 0 {
		return nil
	}
	key := fmt.Sprintf("leaderboard:%s", gameSlug)
	members := make([]redis.Z, len(rows))
	for i, r := range rows {
		members[i] = redis.Z{
			Score:  float64(r.Score),
			Member: fmt.Sprintf("%d:%s", r.PlayedAt.UnixNano(), r.PlayerName),
		}
	}
	return c.rdb.ZAdd(ctx, key, members...).Err()
}
