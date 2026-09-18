// File: config.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product controller
// Purpose: Load the package-relative operator configuration and protected credentials.
// Responsibilities: Resolve package paths, apply safe defaults, and validate operator input.
// Inputs/Outputs: config/product.json and config/credentials.json in; typed Config out.
// Configuration: JSON only; no operator environment-variable plumbing is required.
// Dependencies: Go JSON and network URL libraries.
// Invariants: Product listeners are loopback-only and the symbol cap remains 30.
// Failure behavior: Returns one plain-language configuration error before startup.
// Non-responsibilities: Network connection, process startup, or business behavior.

package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config is the non-secret package configuration used by the product controller.
type Config struct {
	ProductVersion           string   `json:"product_version"`
	AlpacaStreamURL          string   `json:"alpaca_stream_url"`
	AlpacaFeed               string   `json:"alpaca_feed"`
	Symbols                  []string `json:"symbols"`
	SourceAddress            string   `json:"source_address"`
	JEHAddress               string   `json:"jeh_address"`
	ViewerAddress            string   `json:"viewer_address"`
	MongoURI                 string   `json:"mongo_uri"`
	MongoDatabase            string   `json:"mongo_database"`
	MongoTraceCollection     string   `json:"mongo_trace_collection"`
	MongoReservoirCollection string   `json:"mongo_reservoir_collection"`
	SourceDataDirectory      string   `json:"source_data_directory"`
	ViewerDirectory          string   `json:"viewer_directory"`
	NodeExecutable           string   `json:"node_executable"`
	RunType                  string   `json:"run_type"`
	StartingCapital          float64  `json:"starting_capital"`
	AllocationPercent        float64  `json:"allocation_percent"`
	Risk                     float64  `json:"risk_r"`
	SourceBufferCapacity     int      `json:"source_buffer_capacity"`
	OpenViewer               bool     `json:"open_viewer"`
	Home                     string   `json:"-"`
}

// Credentials contains the protected Alpaca credential pair.
type Credentials struct {
	AlpacaKey    string `json:"alpaca_key"`
	AlpacaSecret string `json:"alpaca_secret"`
}

// LoadConfig loads product configuration and credentials relative to home.
func LoadConfig(home string) (Config, Credentials, error) {
	home, err := filepath.Abs(home)
	if err != nil {
		return Config{}, Credentials{}, fmt.Errorf("resolve product folder: %w", err)
	}
	var config Config
	if err := decodeJSON(filepath.Join(home, "config", "product.json"), &config); err != nil {
		return Config{}, Credentials{}, err
	}
	var credentials Credentials
	if err := decodeJSON(filepath.Join(home, "config", "credentials.json"), &credentials); err != nil {
		return Config{}, Credentials{}, fmt.Errorf("credentials are not configured; run Configure JEH-HOHO: %w", err)
	}
	config.Home = home
	config.SourceDataDirectory = packagePath(home, config.SourceDataDirectory)
	config.ViewerDirectory = packagePath(home, config.ViewerDirectory)
	config.NodeExecutable = packagePath(home, config.NodeExecutable)
	if err := config.Validate(credentials); err != nil {
		return Config{}, Credentials{}, err
	}
	return config, credentials, nil
}

// Validate checks only established product choices and package safety constraints.
func (config Config) Validate(credentials Credentials) error {
	if strings.TrimSpace(credentials.AlpacaKey) == "" || strings.TrimSpace(credentials.AlpacaSecret) == "" {
		return fmt.Errorf("Alpaca credentials are missing; run Configure JEH-HOHO")
	}
	if len(config.Symbols) == 0 || len(config.Symbols) > 30 {
		return fmt.Errorf("symbols must contain between 1 and 30 entries")
	}
	for _, address := range []string{config.SourceAddress, config.JEHAddress, config.ViewerAddress} {
		if err := validateLoopbackAddress(address); err != nil {
			return err
		}
	}
	parsedURL, err := url.Parse(config.AlpacaStreamURL)
	if err != nil || (parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss") || parsedURL.Host == "" {
		return fmt.Errorf("alpaca_stream_url must be a ws or wss address")
	}
	if strings.TrimSpace(config.AlpacaFeed) == "" || strings.TrimSpace(config.MongoURI) == "" || strings.TrimSpace(config.MongoDatabase) == "" {
		return fmt.Errorf("Alpaca feed and MongoDB settings are required")
	}
	if strings.TrimSpace(config.MongoTraceCollection) == "" || strings.TrimSpace(config.MongoReservoirCollection) == "" {
		return fmt.Errorf("both required MongoDB collection names are required")
	}
	if config.RunType == "" || config.StartingCapital <= 0 || config.AllocationPercent < 0 || config.AllocationPercent > 1 || config.Risk < 0 || config.Risk > 1 {
		return fmt.Errorf("Capital run settings are invalid")
	}
	if config.SourceBufferCapacity <= 0 {
		return fmt.Errorf("source_buffer_capacity must be positive")
	}
	return nil
}

func decodeJSON(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	return nil
}

func packagePath(home, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(home, filepath.FromSlash(value))
}

func validateLoopbackAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return fmt.Errorf("invalid local address %q", address)
	}
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("product address %q must use loopback", address)
		}
	}
	return nil
}
