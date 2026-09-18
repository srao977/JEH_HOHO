// File: store.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product lifecycle
// Purpose: Aggregate controller, source, DSE_JEH, persistence, and Viewer status.
// Responsibilities: Preserve supported lifecycle states and expose race-safe snapshots.
// Inputs/Outputs: Component state updates in; ProductStatus snapshots out.
// Configuration: Product identity, version, symbols, and dynamic health providers.
// Dependencies: Existing Alpaca and DSE_JEH status snapshots.
// Invariants: READY is independent of first-Bar receipt or freshness.
// Failure behavior: Missing providers produce explicit non-live component status.
// Concurrency/Lifecycle: Mutex-protected for controller and RPC goroutines.
// Non-responsibilities: Component startup, persistence, or frozen calculations.

package lifecycle

import (
	"sync"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"
	"jeh-hoho/internal/source/alpaca"
)

// Store owns the current aggregate product status.
type Store struct {
	mu         sync.RWMutex
	identity   *jehhohov1.ProductIdentity
	lifecycle  jehhohov1.ProductLifecycleState
	reason     string
	symbols    []string
	source     func() alpaca.Health
	jeh        func() *dsejehv1.RuntimeStatusEvidence
	viewerLive bool
}

// NewStore creates a STARTING product status.
func NewStore(identity *jehhohov1.ProductIdentity, symbols []string) *Store {
	return &Store{identity: identity, lifecycle: jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_STARTING, symbols: append([]string(nil), symbols...)}
}

// Bind attaches dynamic component health providers.
func (store *Store) Bind(source func() alpaca.Health, jeh func() *dsejehv1.RuntimeStatusEvidence) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.source, store.jeh = source, jeh
}

// Transition changes aggregate lifecycle without inventing additional states.
func (store *Store) Transition(state jehhohov1.ProductLifecycleState, reason string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.lifecycle, store.reason = state, reason
}

// Viewer marks the owned Viewer process state.
func (store *Store) Viewer(live bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.viewerLive = live
}

// Snapshot returns one aggregate product status snapshot.
func (store *Store) Snapshot() *jehhohov1.ProductStatus {
	store.mu.RLock()
	identity, lifecycle, reason, viewerLive := store.identity, store.lifecycle, store.reason, store.viewerLive
	symbols, sourceProvider, jehProvider := append([]string(nil), store.symbols...), store.source, store.jeh
	store.mu.RUnlock()
	now := time.Now().UnixMilli()
	status := &jehhohov1.ProductStatus{ProductIdentity: identity, LifecycleState: lifecycle, Reason: reason, ProducedUnixMs: now}
	status.Viewer = component("viewer", viewerLive, now)
	status.Evidence = component("persistence", jehProvider != nil, now)
	status.Jeh = component("DSE_JEH", jehProvider != nil, now)
	status.Source = &jehhohov1.SourceStreamStatus{SubscribedSymbols: symbols, ProducedUnixMs: now, Continuity: &jehhohov1.ContinuityStatus{State: jehhohov1.ContinuityState_CONTINUITY_STATE_UNKNOWN, ProducedUnixMs: now}}
	if sourceProvider != nil {
		health := sourceProvider()
		status.Source.ConnectionState = sourceState(health.State)
		status.Source.Reason = health.LastError
		status.Source.LastObservationUnixMs = health.LastMessage.UnixMilli()
		status.Source.Continuity.Reason = "continuity cannot be proven across a disconnect"
		if health.Continuity == "continuous" {
			status.Source.Continuity.State = jehhohov1.ContinuityState_CONTINUITY_STATE_CONTINUOUS
			status.Source.Continuity.Reason = ""
		}
	}
	status.Activity = &jehhohov1.ProductActivity{}
	if jehProvider != nil {
		snapshot := jehProvider()
		activity := snapshot.GetActivity()
		status.Activity.BarsReceived = activity.GetBarsReceived()
		status.Activity.BarsAdmitted = activity.GetBarsAdmitted()
		status.Activity.BarsRejected = activity.GetBarsRejected()
		status.Activity.ExecutionEvents = activity.GetExecutionEvents()
	}
	return status
}

func component(name string, live bool, now int64) *jehhohov1.ComponentStatus {
	state := jehhohov1.ComponentLifecycleState_COMPONENT_LIFECYCLE_STATE_STOPPED
	if live {
		state = jehhohov1.ComponentLifecycleState_COMPONENT_LIFECYCLE_STATE_RUNNING
	}
	return &jehhohov1.ComponentStatus{Component: name, LifecycleState: state, ProcessLive: live, ProducedUnixMs: now}
}

func sourceState(state string) jehhohov1.SourceConnectionState {
	switch state {
	case "healthy":
		return jehhohov1.SourceConnectionState_SOURCE_CONNECTION_STATE_SUBSCRIBED
	case "recovering":
		return jehhohov1.SourceConnectionState_SOURCE_CONNECTION_STATE_RECONNECTING
	case "failed":
		return jehhohov1.SourceConnectionState_SOURCE_CONNECTION_STATE_FAILED
	default:
		return jehhohov1.SourceConnectionState_SOURCE_CONNECTION_STATE_CONNECTING
	}
}
