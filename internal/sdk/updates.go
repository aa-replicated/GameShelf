package sdk

import (
	"context"
	"sync"
	"time"
)

// UpdateInfo describes a pending application update.
type UpdateInfo struct {
	VersionLabel string `json:"versionLabel"`
	CreatedAt    string `json:"createdAt"`
}

// UpdatesResponse mirrors the /api/v1/app/updates response.
type UpdatesResponse struct {
	Updates []UpdateInfo `json:"updates"`
}

type updatesCache struct {
	mu        sync.RWMutex
	updates   []UpdateInfo
	fetchedAt time.Time
	ttl       time.Duration
}

var globalUpdatesCache = &updatesCache{ttl: 5 * time.Minute}

// CheckForUpdates returns available updates, using a 5-minute cache.
// Returns nil, nil when SDK is unavailable.
func (c *Client) CheckForUpdates(ctx context.Context) ([]UpdateInfo, error) {
	if !c.Available() {
		return nil, nil
	}

	globalUpdatesCache.mu.RLock()
	if time.Since(globalUpdatesCache.fetchedAt) < globalUpdatesCache.ttl {
		updates := globalUpdatesCache.updates
		globalUpdatesCache.mu.RUnlock()
		return updates, nil
	}
	globalUpdatesCache.mu.RUnlock()

	var resp UpdatesResponse
	if err := c.get(ctx, "/api/v1/app/updates", &resp); err != nil {
		return nil, err
	}

	globalUpdatesCache.mu.Lock()
	globalUpdatesCache.updates = resp.Updates
	globalUpdatesCache.fetchedAt = time.Now()
	globalUpdatesCache.mu.Unlock()

	return resp.Updates, nil
}

// HasUpdate returns true if there is at least one pending update.
// Returns false when SDK is unavailable.
func (c *Client) HasUpdate(ctx context.Context) bool {
	updates, err := c.CheckForUpdates(ctx)
	if err != nil || len(updates) == 0 {
		return false
	}
	return true
}
