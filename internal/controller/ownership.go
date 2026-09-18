// File: ownership.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product process ownership
// Purpose: Coordinate operator STOP/STATUS without discovering or killing arbitrary processes.
// Responsibilities: Persist owned run identity and validate package-local stop requests.
// Inputs/Outputs: Controller state in; status JSON and authenticated stop request out.
// Configuration: Package runtime directory only.
// Dependencies: Cryptographic randomness and JSON files.
// Invariants: STOP never names, signals, or terminates an unrelated PID.
// Failure behavior: Missing/stale state returns a plain operator-readable message.
// Concurrency/Lifecycle: One controller owns one token until clean shutdown.
// Non-responsibilities: OS service management or broad process-name termination.

package controller

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type runtimeRecord struct {
	PID          int    `json:"pid"`
	Executable   string `json:"executable"`
	ProductRunID string `json:"product_run_id"`
	State        string `json:"state"`
	Reason       string `json:"reason,omitempty"`
	ViewerURL    string `json:"viewer_url"`
	StartedUTC   string `json:"started_utc"`
	UpdatedUTC   string `json:"updated_utc"`
	Token        string `json:"token"`
}

func newToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func runtimeDirectory(home string) string { return filepath.Join(home, "runtime") }
func statePath(home string) string {
	return filepath.Join(runtimeDirectory(home), "product-state.json")
}
func stopPath(home string) string { return filepath.Join(runtimeDirectory(home), "stop.request") }

func writeRecord(home string, record runtimeRecord) error {
	if err := os.MkdirAll(runtimeDirectory(home), 0o700); err != nil {
		return err
	}
	record.UpdatedUTC = time.Now().UTC().Format(time.RFC3339)
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	temporary := statePath(home) + ".tmp"
	if err := os.WriteFile(temporary, content, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, statePath(home))
}

// RequestStop creates an authenticated request consumed only by the owning controller.
func RequestStop(home string) error {
	content, err := os.ReadFile(statePath(home))
	if err != nil {
		return fmt.Errorf("JEH-HOHO is not running")
	}
	var record runtimeRecord
	if err := json.Unmarshal(content, &record); err != nil || record.Token == "" {
		return fmt.Errorf("JEH-HOHO runtime status is invalid; no process was stopped")
	}
	return os.WriteFile(stopPath(home), []byte(record.Token), 0o600)
}

// ReadOperatorStatus returns the plain current state recorded by the controller.
func ReadOperatorStatus(home string) (string, error) {
	content, err := os.ReadFile(statePath(home))
	if os.IsNotExist(err) {
		return "STOPPED", nil
	}
	if err != nil {
		return "", fmt.Errorf("JEH-HOHO status is unavailable")
	}
	var record runtimeRecord
	if err := json.Unmarshal(content, &record); err != nil {
		return "", fmt.Errorf("JEH-HOHO status is unavailable")
	}
	result := record.State
	if record.Reason != "" {
		result += ": " + record.Reason
	}
	if record.ViewerURL != "" {
		result += "\nViewer: " + record.ViewerURL
	}
	return result, nil
}

func stopRequested(home, token string) bool {
	content, err := os.ReadFile(stopPath(home))
	return err == nil && string(content) == token
}

func clearRuntime(home string) {
	_ = os.Remove(stopPath(home))
	_ = os.Remove(statePath(home))
}
