// File: mapper_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 regression authority
// Product/Component: JEH-HOHO / Accepted Bar product adapter tests
// Purpose: Prove product mapping preserves all source evidence and accepted finality.
// Failure meaning: The product boundary diverged from graduated source semantics.

package contract

import (
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"jeh-hoho/internal/source/types"
)

func TestToAcceptedBarPreservesSourceEvidence(t *testing.T) {
	start := time.Date(2026, 9, 17, 12, 0, 0, 123000000, time.UTC)
	received := start.Add(2 * time.Second)
	observation := types.Observation{
		CollectionRunID: "collection-1", Symbol: "AAPL", GeneratorSequenceNo: 7,
		SourceEventTime: start, SourceTimestampText: "2026-09-17T12:00:00.123Z", ReceivedTime: received,
		Interval: types.Interval1Min, Open: 10, High: 12, Low: 9, Close: 11, Volume: 20, EventCount: 3,
		SourceID: "ALPACA_IEX", AlpacaMessageType: "u", PayloadHash: "raw-sha256",
		SourceTimeRegression: true, DuplicateArrival: true,
	}
	mapped, err := ToAcceptedBar("observation-7", "iex", observation)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.ObservationId != "observation-7" || mapped.EntitySequence != 7 || mapped.AlpacaMessageType != "u" {
		t.Fatalf("identity/type changed: %+v", mapped)
	}
	if mapped.PayloadHash != "raw-sha256" || !mapped.DuplicateArrival || !mapped.SourceTimeRegression || !mapped.IsFinal {
		t.Fatalf("source evidence changed: %+v", mapped)
	}
	if mapped.IntervalStartUnixMs != start.UnixMilli() || mapped.IntervalEndUnixMs != start.Add(time.Minute).UnixMilli() || mapped.ReceivedUnixMs != received.UnixMilli() {
		t.Fatalf("time mapping changed: %+v", mapped)
	}
	if mapped.Volume == nil || *mapped.Volume != 20 || mapped.EventCount == nil || *mapped.EventCount != 3 {
		t.Fatalf("optional values changed: %+v", mapped)
	}
	if got, want := ObservationID(observation), "10d3fcfcaf274841ec4b2de8cb8809bcd59592a3f5762a7c5493e45d06a6d695"; got != want {
		t.Fatalf("observation identity = %s, want %s", got, want)
	}
}

func TestToAcceptedBarRejectsIncompleteIdentity(t *testing.T) {
	if _, err := ToAcceptedBar("", "iex", types.Observation{}); err == nil {
		t.Fatal("expected incomplete identity rejection")
	}
}

func TestLiveAndPersistedObservationMapToEquivalentJEHFields(t *testing.T) {
	start := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	live := types.Observation{
		CollectionRunID: "collection-1", PartitionID: "A", Symbol: "AAPL", GeneratorSequenceNo: 9,
		SourceEventTime: start, SourceTimestampText: start.Format(time.RFC3339), ReceivedTime: start.Add(time.Second),
		Interval: types.Interval1Min, Open: 10, High: 12, Low: 9, Close: 11, Volume: 20, EventCount: 3,
		SourceID: "ALPACA_IEX", AlpacaMessageType: "b", PayloadHash: "payload",
	}
	encoded, err := json.Marshal(live)
	if err != nil {
		t.Fatal(err)
	}
	var persisted types.Observation
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	liveBar, err := ToAcceptedBar(ObservationID(live), "iex", live)
	if err != nil {
		t.Fatal(err)
	}
	persistedBar, err := ToAcceptedBar(ObservationID(persisted), "iex", persisted)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(liveBar, persistedBar) {
		t.Fatalf("live/persisted semantic mismatch:\nlive=%+v\npersisted=%+v", liveBar, persistedBar)
	}
}
