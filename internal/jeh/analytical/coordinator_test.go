// File: internal/jeh/analytical/coordinator_test.go (coordinator_test.go) // Date: 2026-09-17 // Version/Status: 1.0 / Graduated frozen JEH behavior // Product/Component: JEH-HOHO / Frozen JEH Engine // Purpose: Preserve the file's existing DSE_JEH implementation or regression. // Origin: DSE_JEH_Lab/internal/jeh/analytical/coordinator_test.go, Slice 0 fingerprint // Invariants: Governed JEH/Capital/runtime semantics unchanged // Non-responsibilities: No Fin transport and no public product contract.

package analytical

import (
	"fmt"
	"testing"
	"time"

	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"
)

func TestCoordinatorInitializationAndEntityIsolation(t *testing.T) {
	coordinator := NewCoordinator()
	for sequence := uint64(1); sequence <= 64; sequence++ {
		event := analyticalEvent("run|AAPL", "AAPL", sequence)
		result := coordinator.Process(event, admitted(event))
		want := dsejehv1.PhaseStatus_PHASE_STATUS_INITIALIZING
		if sequence == 64 {
			want = dsejehv1.PhaseStatus_PHASE_STATUS_OBSERVABLE
		}
		if result.GetStatus() != want {
			t.Fatalf("AAPL sequence %d status = %v; want %v", sequence, result.GetStatus(), want)
		}
		if sequence <= 63 && result.PhaseDegrees != nil {
			t.Fatalf("AAPL sequence %d phase must be unavailable", sequence)
		}
		if sequence == 64 && result.PhaseDegrees == nil {
			t.Fatal("AAPL sequence 64 phase must be production-observable")
		}
	}
	event := analyticalEvent("run|MSFT", "MSFT", 1)
	if got := coordinator.Process(event, admitted(event)).GetStatus(); got != dsejehv1.PhaseStatus_PHASE_STATUS_INITIALIZING {
		t.Fatalf("isolated MSFT status = %v", got)
	}
}

func TestCoordinatorUsesSequenceContinuityNotElapsedTime(t *testing.T) {
	coordinator := NewCoordinator()
	start := time.Date(2026, 9, 11, 10, 4, 0, 0, time.UTC)
	for sequence := uint64(1); sequence <= 64; sequence++ {
		event := analyticalEvent("run|AAPL", "AAPL", sequence)
		event.Provenance.SourceEventUnixMs = start.Add(time.Duration(sequence*sequence) * time.Minute).UnixMilli()
		event.Provenance.ReceivedUnixMs = start.Add(time.Duration(sequence*sequence+sequence) * time.Minute).UnixMilli()
		result := coordinator.Process(event, admitted(event))
		want := dsejehv1.PhaseStatus_PHASE_STATUS_INITIALIZING
		if sequence == 64 {
			want = dsejehv1.PhaseStatus_PHASE_STATUS_OBSERVABLE
		}
		if result.GetStatus() != want {
			t.Fatalf("sequence %d status = %v; want %v", sequence, result.GetStatus(), want)
		}
	}
}

func TestCoordinatorSparseSymbolRemainsInitializing(t *testing.T) {
	coordinator := NewCoordinator()
	for sequence := uint64(1); sequence <= 51; sequence++ {
		event := analyticalEvent("run|VXX", "VXX", sequence)
		result := coordinator.Process(event, admitted(event))
		if result.GetStatus() != dsejehv1.PhaseStatus_PHASE_STATUS_INITIALIZING || result.PhaseDegrees != nil {
			t.Fatalf("VXX sequence %d = %+v; want unavailable INITIALIZING", sequence, result)
		}
	}
}

func analyticalEvent(scope, entity string, sequence uint64) *dsejehv1.BarEvent {
	return &dsejehv1.BarEvent{
		EventId: fmt.Sprintf("%s-%d", entity, sequence), EntityId: entity,
		High: 101 + float64(sequence), Low: 99 + float64(sequence),
		Provenance: &dsejehv1.SourceProvenance{EntitySequenceScope: scope, EntitySequence: sequence},
	}
}

func admitted(event *dsejehv1.BarEvent) *dsejehv1.BarAdmissionEvidence {
	return &dsejehv1.BarAdmissionEvidence{
		AdmissionId: "admit-" + event.GetEventId(), BarEventId: event.GetEventId(),
		EntityId: event.GetEntityId(), EntitySequence: event.GetProvenance().GetEntitySequence(),
		Status: dsejehv1.BarAdmissionStatus_BAR_ADMISSION_STATUS_ADMITTED,
	}
}
