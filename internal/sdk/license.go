package sdk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// LicenseInfo mirrors the /api/v1/license/info response.
type LicenseInfo struct {
	LicenseID      string     `json:"licenseID"`
	LicenseType    string     `json:"licenseType"`
	CustomerName   string     `json:"customerName"`
	ExpirationDate *time.Time `json:"expirationDate"`
	IsExpired      bool       `json:"isExpired"`
}

// FieldValue mirrors the /api/v1/license/fields/:name response.
// Value is interface{} because the SDK may return JSON booleans (true/false)
// or strings — decoding into a string field silently drops non-string values.
type FieldValue struct {
	Value interface{} `json:"value"`
}

// licenseCache holds a cached LicenseInfo with a TTL.
type licenseCache struct {
	mu        sync.RWMutex
	info      *LicenseInfo
	fetchedAt time.Time
	ttl       time.Duration
}

var globalLicenseCache = &licenseCache{ttl: 1 * time.Minute}

// GetLicenseInfo returns license info, using a 1-minute cache.
// Returns nil, nil when SDK is unavailable (fail-open).
func (c *Client) GetLicenseInfo(ctx context.Context) (*LicenseInfo, error) {
	if !c.Available() {
		return nil, nil
	}

	globalLicenseCache.mu.RLock()
	if globalLicenseCache.info != nil && time.Since(globalLicenseCache.fetchedAt) < globalLicenseCache.ttl {
		info := globalLicenseCache.info
		globalLicenseCache.mu.RUnlock()
		return info, nil
	}
	globalLicenseCache.mu.RUnlock()

	var info LicenseInfo
	if err := c.get(ctx, "/api/v1/license/info", &info); err != nil {
		return nil, err
	}

	globalLicenseCache.mu.Lock()
	globalLicenseCache.info = &info
	globalLicenseCache.fetchedAt = time.Now()
	globalLicenseCache.mu.Unlock()

	return &info, nil
}

// GetFieldValue returns the value of a named license field as a string.
// Returns "", nil when SDK is unavailable.
func (c *Client) GetFieldValue(ctx context.Context, fieldName string) (string, error) {
	if !c.Available() {
		return "", nil
	}
	var fv FieldValue
	if err := c.get(ctx, "/api/v1/license/fields/"+fieldName, &fv); err != nil {
		return "", err
	}
	if fv.Value == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", fv.Value), nil
}

// IsFeatureEnabled returns true if the named license field equals "true".
// Fail-closed: SDK unreachable → deny access.
// Exception: SDK URL unset AND LocalDev=true → allow (local development bypass).
func (c *Client) IsFeatureEnabled(ctx context.Context, fieldName string) bool {
	if !c.Available() {
		if c.localDev {
			log.Printf("sdk: LOCAL_DEV mode — bypassing %s check", fieldName)
			return true
		}
		log.Printf("sdk: SDK_SERVICE_URL not set — denying %s (set LOCAL_DEV=true to bypass)", fieldName)
		return false
	}
	val, err := c.GetFieldValue(ctx, fieldName)
	if err != nil {
		log.Printf("sdk: %s check failed (%v) — denying access", fieldName, err)
		return false // fail-closed: SDK configured but unreachable
	}
	return val == "true" // absent/null/any other value → deny (fail-closed)
}

// IsLicenseValid returns true if the license exists and is not expired.
// Returns true when SDK is unavailable (fail-open).
func (c *Client) IsLicenseValid(ctx context.Context) bool {
	info, err := c.GetLicenseInfo(ctx)
	if err != nil || info == nil {
		return true // fail-open
	}
	return !info.IsExpired
}
