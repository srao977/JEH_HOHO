// File: engine.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 implementation
// Product/Component: JEH-HOHO / Bar Sequence Source engine
// Purpose: Own collection identity, sequencing, persistence, and bounded live delivery.
// Inputs/Outputs: Accepted source arrivals in; persisted and contract-mapped Bars out.
// Invariants: One authority spans source reconnects; delivery blocks; no observations drop.
// Failure behavior: Source, mapping, and persistence failures cancel and return synchronously.
// Non-responsibilities: Alpaca session recovery, replay, gap fill, or JEH processing.

package source

import (
	"context"
	"errors"
	"fmt"
	"sync"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/source/contract"
	"jeh-hoho/internal/source/sequence"
	"jeh-hoho/internal/source/types"
)

// ObservationSource is implemented by the graduated Alpaca stream.
type ObservationSource interface {
	Run(context.Context, []string, chan<- types.Observation) error
}

// ObservationWriter preserves accepted source evidence before live delivery.
type ObservationWriter interface {
	WriteBatch([]types.Observation) error
}

// EngineConfig contains immutable source-engine ownership configuration.
type EngineConfig struct {
	CollectionRunID string
	Feed            string
	Symbols         []string
	BufferCapacity  int
	Source          ObservationSource
	Writer          ObservationWriter
}

// Engine owns one collection and sequence authority for its complete lifetime.
type Engine struct {
	config    EngineConfig
	authority *sequence.Authority
	accepted  chan *jehhohov1.AcceptedBarObservation

	runMu   sync.Mutex
	running bool
	ran     bool
}

// NewEngine validates configuration and creates the persistent authority and queue.
func NewEngine(config EngineConfig) (*Engine, error) {
	if config.CollectionRunID == "" {
		return nil, fmt.Errorf("collection_run_id is required")
	}
	if config.Source == nil {
		return nil, fmt.Errorf("observation source is required")
	}
	if config.BufferCapacity <= 0 {
		return nil, fmt.Errorf("buffer capacity must be positive")
	}
	config.Symbols = append([]string(nil), config.Symbols...)
	return &Engine{
		config:    config,
		authority: sequence.New(config.CollectionRunID),
		accepted:  make(chan *jehhohov1.AcceptedBarObservation, config.BufferCapacity),
	}, nil
}

// Accepted returns the sole bounded delivery queue consumed by the gRPC service.
func (e *Engine) Accepted() <-chan *jehhohov1.AcceptedBarObservation {
	return e.accepted
}

// Run receives source arrivals until cancellation or a propagated source/write error.
func (e *Engine) Run(ctx context.Context) error {
	e.runMu.Lock()
	if e.running || e.ran {
		e.runMu.Unlock()
		return fmt.Errorf("source engine may run exactly once")
	}
	e.running, e.ran = true, true
	e.runMu.Unlock()
	defer func() {
		e.runMu.Lock()
		e.running = false
		e.runMu.Unlock()
		close(e.accepted)
	}()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	raw := make(chan types.Observation, e.config.BufferCapacity)
	sourceResult := make(chan error, 1)
	go func() { sourceResult <- e.config.Source.Run(runCtx, e.config.Symbols, raw) }()

	for {
		select {
		case <-ctx.Done():
			cancel()
			<-sourceResult
			return ctx.Err()
		case err := <-sourceResult:
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return ctx.Err()
			}
			if err == nil {
				return fmt.Errorf("observation source stopped unexpectedly")
			}
			return fmt.Errorf("observation source failed: %w", err)
		case observation := <-raw:
			stamped := e.authority.Accept(observation)
			if e.config.Writer != nil {
				if err := e.config.Writer.WriteBatch([]types.Observation{stamped}); err != nil {
					cancel()
					<-sourceResult
					return fmt.Errorf("persist accepted observation: %w", err)
				}
			}
			mapped, err := contract.ToAcceptedBar(contract.ObservationID(stamped), e.config.Feed, stamped)
			if err != nil {
				cancel()
				<-sourceResult
				return fmt.Errorf("map accepted observation: %w", err)
			}
			select {
			case e.accepted <- mapped:
			case <-ctx.Done():
				cancel()
				<-sourceResult
				return ctx.Err()
			}
		}
	}
}
