// File: authority.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated frozen source behavior
// Product/Component: JEH-HOHO / Bar Sequence Source
// Purpose: Assign collection identity and accepted-arrival sequence per symbol.
// Origin: Bar_Sequence_Lab/generator/internal/sequence/authority.go, Slice 0 fingerprint.
// Inputs/Outputs: Accepted observations in; immutable sequence-stamped observations out.
// Invariants: Arrival order wins; duplicates remain observations; sequence is per symbol.
// Failure behavior: Thread-safe in-memory authority; no persistence or recovery claim.
// Non-responsibilities: Source sorting, suppression, reconnect, gap fill, or JEH admission.

package sequence

import (
	"sync"
	"time"

	"jeh-hoho/internal/source/types"
)

// Authority owns per-symbol sequence and duplicate/time-regression evidence.
type Authority struct {
	runID string

	mu          sync.Mutex
	next        map[string]uint64
	lastTime    map[string]time.Time
	seen        map[string]struct{}
	regressions uint64
	duplicates  uint64
}

// New creates one authority for a collection run.
func New(runID string) *Authority {
	return &Authority{
		runID:    runID,
		next:     map[string]uint64{},
		lastTime: map[string]time.Time{},
		seen:     map[string]struct{}{},
	}
}

// Accept stamps one arrival without sorting or suppressing it.
func (a *Authority) Accept(obs types.Observation) types.Observation {
	a.mu.Lock()
	defer a.mu.Unlock()
	obs.CollectionRunID = a.runID
	seq := a.next[obs.Symbol] + 1
	a.next[obs.Symbol] = seq
	obs.GeneratorSequenceNo = seq

	if last, ok := a.lastTime[obs.Symbol]; ok && !obs.SourceEventTime.IsZero() && obs.SourceEventTime.Before(last) {
		obs.SourceTimeRegression = true
		a.regressions++
	}
	if !obs.SourceEventTime.IsZero() {
		a.lastTime[obs.Symbol] = obs.SourceEventTime
	}

	key := obs.DuplicateKey()
	if _, ok := a.seen[key]; ok {
		obs.DuplicateArrival = true
		a.duplicates++
	} else {
		a.seen[key] = struct{}{}
	}
	return obs
}

// Stats returns counters and a copy of the latest per-symbol sequences.
func (a *Authority) Stats() (regressions, duplicates uint64, latest map[string]uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	latest = make(map[string]uint64, len(a.next))
	for k, v := range a.next {
		latest[k] = v
	}
	return a.regressions, a.duplicates, latest
}
