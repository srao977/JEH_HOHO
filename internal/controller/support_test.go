// File: support_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 regression authority
// Product/Component: JEH-HOHO / Product support tests
// Purpose: Prove support output excludes credentials and URI user information.
// Failure meaning: Product support tooling can disclose secrets.

package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportReportIsSanitized(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	product := `{"product_version":"test","alpaca_stream_url":"wss://stream.example.test/v2/iex","alpaca_feed":"iex","symbols":["AAPL"],"source_address":"127.0.0.1:1","jeh_address":"127.0.0.1:2","viewer_address":"127.0.0.1:3","mongo_uri":"mongodb://secret-user:secret-password@mongo.example.test:27017","mongo_database":"db"}`
	if err := os.WriteFile(filepath.Join(home, "config", "product.json"), []byte(product), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := CreateSupportReport(home)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, forbidden := range []string{"secret-user", "secret-password", "alpaca_key", "alpaca_secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("support report contains %q", forbidden)
		}
	}
	if !strings.Contains(text, "mongo.example.test:27017") {
		t.Fatal("support report omitted sanitized Mongo host")
	}
}
