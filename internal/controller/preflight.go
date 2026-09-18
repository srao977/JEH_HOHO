// File: preflight.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product preflight
// Purpose: Fail clearly before product processes start when a required dependency is unavailable.
// Responsibilities: Validate ports, Viewer assets, Node runtime, directories, and Mongo writers.
// Inputs/Outputs: Validated Config in; nil or an operator-readable fault out.
// Dependencies: Existing Mongo writer constructors preserve collection requirements.
// Invariants: Preflight creates no product listener and changes no product state.
// Failure behavior: First failed mandatory check blocks startup.
// Non-responsibilities: Alpaca session semantics, JEH readiness, or process supervision.

package controller

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	mongoadapter "jeh-hoho/internal/jeh/adapter/mongo"
	"jeh-hoho/internal/source/alpaca"
)

// Preflight validates all dependencies required before component startup.
func Preflight(ctx context.Context, config Config, credentials Credentials) error {
	if err := config.Validate(credentials); err != nil {
		return err
	}
	for _, address := range []string{config.SourceAddress, config.JEHAddress, config.ViewerAddress} {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return fmt.Errorf("required local port %s is already in use", address)
		}
		_ = listener.Close()
	}
	for _, required := range []string{config.NodeExecutable, filepath.Join(config.ViewerDirectory, "server.js"), filepath.Join(config.ViewerDirectory, ".next", "static")} {
		if _, err := os.Stat(required); err != nil {
			return fmt.Errorf("required packaged Viewer asset is unavailable: %s", required)
		}
	}
	if err := os.MkdirAll(config.SourceDataDirectory, 0o755); err != nil {
		return fmt.Errorf("prepare source data folder: %w", err)
	}
	stream := alpaca.NewStream(config.AlpacaStreamURL, config.AlpacaFeed, alpaca.Credentials{Key: credentials.AlpacaKey, Secret: credentials.AlpacaSecret})
	if err := stream.Preflight(ctx, config.Symbols); err != nil {
		return fmt.Errorf("Alpaca preflight failed: %w", err)
	}
	traceWriter, err := mongoadapter.OpenTraceWriter(ctx, config.MongoURI, config.MongoDatabase, config.MongoTraceCollection)
	if err != nil {
		return fmt.Errorf("MongoDB preflight failed: %w", err)
	}
	_ = traceWriter.Close(context.Background())
	reservoirWriter, err := mongoadapter.OpenCapitalReservoirWriter(ctx, config.MongoURI, config.MongoDatabase, config.MongoReservoirCollection)
	if err != nil {
		return fmt.Errorf("MongoDB preflight failed: %w", err)
	}
	_ = reservoirWriter.Close(context.Background())
	return nil
}
