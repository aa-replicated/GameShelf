package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client is a thin wrapper around the Replicated SDK sidecar HTTP API.
// All methods are fail-open: if the SDK is unreachable they return zero
// values and a nil error so the application continues normally.
type Client struct {
	base       string
	httpClient *http.Client
}

// New returns a Client pointed at baseURL (e.g. "http://localhost:3000").
// Pass an empty string to get a no-op client that always returns zero values.
func New(baseURL string) *Client {
	return &Client{
		base: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Available reports whether the SDK sidecar is configured.
func (c *Client) Available() bool {
	return c.base != ""
}

// get performs a GET request and JSON-decodes the response body into dst.
// Returns an error only for non-2xx responses; connection failures return nil
// so callers stay fail-open (they check Available() first).
func (c *Client) get(ctx context.Context, path string, dst any) error {
	if !c.Available() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return fmt.Errorf("sdk: build request: %w", err)
	}
	log.Printf("sdk: GET %s", path)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Connection failure — stay fail-open, return nil
		log.Printf("sdk: GET %s unreachable: %v", path, err)
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sdk: %s returned %d: %s", path, resp.StatusCode, body)
	}
	log.Printf("sdk: GET %s -> %d OK", path, resp.StatusCode)
	if dst != nil {
		if err := json.Unmarshal(body, dst); err != nil {
			return fmt.Errorf("sdk: decode %s: %w", path, err)
		}
	}
	return nil
}

// patch performs a PATCH request with a JSON body.
func (c *Client) patch(ctx context.Context, path string, body io.Reader) error {
	if !c.Available() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.base+path, body)
	if err != nil {
		return fmt.Errorf("sdk: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	log.Printf("sdk: PATCH %s", path)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("sdk: PATCH %s unreachable: %v", path, err)
		return nil // fail-open
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sdk: PATCH %s returned %d: %s", path, resp.StatusCode, b)
	}
	log.Printf("sdk: PATCH %s -> %d OK", path, resp.StatusCode)
	return nil
}
