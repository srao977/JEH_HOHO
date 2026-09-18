// File: support.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product support
// Purpose: Create a bounded, sanitized product support report with no credentials.
// Responsibilities: Capture package version, safe configuration, status, and manifest identity.
// Inputs/Outputs: Package-owned files in; one timestamped JSON report out.
// Configuration: Reads product.json but never credentials.json or environment dumps.
// Dependencies: Standard JSON, URL, and filesystem packages.
// Invariants: Credentials and Mongo URI user information are never emitted.
// Failure behavior: Returns a concise error and leaves product lifecycle unchanged.
// Non-responsibilities: Log archiving, process mutation, or external upload.

package controller

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// CreateSupportReport writes a sanitized report and returns its path.
func CreateSupportReport(home string) (string, error) {
	var config Config
	if err := decodeJSON(filepath.Join(home, "config", "product.json"), &config); err != nil {
		return "", err
	}
	status := "STOPPED"
	if current, err := ReadOperatorStatus(home); err == nil {
		status = current
	}
	report := map[string]any{
		"created_utc":     time.Now().UTC().Format(time.RFC3339),
		"product_version": config.ProductVersion,
		"status":          status,
		"configuration": map[string]any{
			"alpaca_stream_host": safeHost(config.AlpacaStreamURL), "alpaca_feed": config.AlpacaFeed,
			"symbol_count": len(config.Symbols), "source_address": config.SourceAddress, "jeh_address": config.JEHAddress,
			"viewer_address": config.ViewerAddress, "mongo_host": safeHost(config.MongoURI), "mongo_database": config.MongoDatabase,
		},
	}
	if manifest, err := os.ReadFile(filepath.Join(home, "package-manifest.json")); err == nil {
		var value any
		if json.Unmarshal(manifest, &value) == nil {
			report["package_manifest"] = value
		}
	}
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	directory := filepath.Join(home, "support")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, "JEH-HOHO-support-"+time.Now().UTC().Format("20060102T150405Z")+".json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write support report: %w", err)
	}
	return path, nil
}

func safeHost(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "unavailable"
	}
	return parsed.Host
}
