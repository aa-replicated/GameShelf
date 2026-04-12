package sdk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SupportBundleResult holds the response from the SDK after a successful upload.
type SupportBundleResult struct {
	BundleID string `json:"bundleId"`
	Slug     string `json:"slug"`
}

// TriggerSupportBundleUpload collects lightweight app diagnostics, packs them
// into a tar.gz archive, and uploads it to the Vendor Portal via the SDK sidecar.
// Returns a zero SupportBundleResult and nil error when the SDK is unavailable.
//
// NOTE: In production, generate a real bundle with kubectl support-bundle and
// POST the resulting archive to POST /api/v1/supportbundle instead.
func (c *Client) TriggerSupportBundleUpload(ctx context.Context, licenseInfo *LicenseInfo) (SupportBundleResult, error) {
	if !c.Available() {
		return SupportBundleResult{}, nil
	}

	buf, err := buildBundle(licenseInfo)
	if err != nil {
		return SupportBundleResult{}, fmt.Errorf("sdk: build support bundle: %w", err)
	}

	contentLength := int64(buf.Len())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/supportbundle", buf)
	if err != nil {
		return SupportBundleResult{}, fmt.Errorf("sdk: build support bundle request: %w", err)
	}
	req.Header.Set("Content-Type", "application/gzip")
	req.ContentLength = contentLength

	// The SDK upload flow makes multiple outbound calls (get presigned URL → S3 PUT →
	// mark uploaded), so we use a longer timeout than the default 3s client.
	uploadClient := &http.Client{Timeout: 30 * time.Second}
	log.Printf("sdk: POST /api/v1/supportbundle (%d bytes)", contentLength)
	resp, err := uploadClient.Do(req)
	if err != nil {
		return SupportBundleResult{}, fmt.Errorf("sdk: support bundle upload: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return SupportBundleResult{}, fmt.Errorf("sdk: support bundle upload returned %d: %s", resp.StatusCode, body)
	}

	var result SupportBundleResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SupportBundleResult{}, fmt.Errorf("sdk: decode support bundle response: %w", err)
	}
	log.Printf("sdk: support bundle uploaded: id=%s slug=%s", result.BundleID, result.Slug)
	return result, nil
}

// buildBundle creates an in-memory tar.gz with lightweight app diagnostics.
func buildBundle(licenseInfo *LicenseInfo) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	appInfo := map[string]interface{}{
		"collectedAt": time.Now().UTC().Format(time.RFC3339),
		"source":      "gameshelf-app",
		"note":        "Lightweight diagnostic bundle collected by the GameShelf application. In production, use kubectl support-bundle for a full cluster bundle.",
	}
	if err := addJSONFile(tw, "bundle/app-info.json", appInfo); err != nil {
		return nil, err
	}

	if licenseInfo != nil {
		licenseData := map[string]interface{}{
			"licenseID":    licenseInfo.LicenseID,
			"licenseType":  licenseInfo.LicenseType,
			"customerName": licenseInfo.CustomerName,
			"isExpired":    licenseInfo.IsExpired,
		}
		if licenseInfo.ExpirationDate != nil {
			licenseData["expirationDate"] = licenseInfo.ExpirationDate.Format(time.RFC3339)
		}
		if err := addJSONFile(tw, "bundle/license.json", licenseData); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("close gzip: %w", err)
	}
	return &buf, nil
}

// addJSONFile writes v as an indented JSON file into the tar archive.
func addJSONFile(tw *tar.Writer, name string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	hdr := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header %s: %w", name, err)
	}
	_, err = tw.Write(data)
	return err
}
