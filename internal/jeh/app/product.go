// File: product.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 4 implementation
// Product/Component: JEH-HOHO / Online JEH composition
// Purpose: Assemble the frozen online JEH application with required existing writers.
// Inputs/Outputs: Product source/configuration in; fully wired frozen Application out.
// Invariants: Trace opens before Capital; neither writer may be silently omitted.
// Failure behavior: Partial construction closes owned resources and returns the error.
// Non-responsibilities: Writer retry, Capital arithmetic, Mongo schema, or lifecycle UI.

package app

import (
	"context"
	"fmt"

	mongoadapter "jeh-hoho/internal/jeh/adapter/mongo"
	"jeh-hoho/internal/jeh/config"
	"jeh-hoho/internal/jeh/evidence"
)

// NewProduct builds ONLINE composition and opens any missing required evidence writers.
func NewProduct(ctx context.Context, cfg config.Config, options Options) (*Application, error) {
	if options.Source == nil {
		return nil, fmt.Errorf("product source is required")
	}
	ownedTrace := false
	ownedReservoir := false
	if options.TraceWriter == nil {
		writer, err := mongoadapter.OpenTraceWriter(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTraceCollection)
		if err != nil {
			return nil, err
		}
		options.TraceWriter = writer
		ownedTrace = true
	}
	if options.ReservoirWriter == nil {
		writer, err := mongoadapter.OpenCapitalReservoirWriter(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoReservoirCollection)
		if err != nil {
			if ownedTrace {
				_ = options.TraceWriter.Close(context.Background())
			}
			return nil, err
		}
		options.ReservoirWriter = writer
		ownedReservoir = true
	}
	application, err := New(cfg, options)
	if err != nil {
		if ownedTrace {
			_ = options.TraceWriter.Close(context.Background())
		}
		if ownedReservoir {
			_ = options.ReservoirWriter.Close(context.Background())
		}
		return nil, err
	}
	return application, nil
}

// EvidenceBus returns the frozen runtime publication source for the product gateway.
func (application *Application) EvidenceBus() *evidence.Bus { return application.bus }
