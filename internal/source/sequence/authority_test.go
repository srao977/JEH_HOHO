// File: authority_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated regression authority
// Product/Component: JEH-HOHO / Bar Sequence Source tests
// Purpose: Lock arrival-order, per-symbol sequence, and duplicate-preservation behavior.
// Origin: Bar_Sequence_Lab/generator/internal/sequence/authority_test.go.
// Failure meaning: Source behavior changed and Slice 2 must not progress.

package sequence

import (
	"testing"
	"time"

	"jeh-hoho/internal/source/types"
)

func bar(symbol string, eventTime time.Time, hash string) types.Observation {
	return types.Observation{
		Symbol:              symbol,
		SourceEventTime:     eventTime,
		SourceTimestampText: eventTime.Format(time.RFC3339),
		ReceivedTime:        time.Now().UTC(),
		PayloadHash:         hash,
		AlpacaMessageType:   "b",
	}
}

func TestAcceptedArrivalNotSourceSorted(t *testing.T) {
	authority := New("run1")
	t1 := time.Date(2026, 9, 10, 12, 1, 0, 0, time.UTC)
	t0 := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	first := authority.Accept(bar("AAPL", t1, "h1"))
	second := authority.Accept(bar("AAPL", t0, "h2"))
	if first.GeneratorSequenceNo != 1 || second.GeneratorSequenceNo != 2 {
		t.Fatalf("seq %d %d", first.GeneratorSequenceNo, second.GeneratorSequenceNo)
	}
	if second.SourceTimeRegression != true {
		t.Fatal("expected source_time_regression flag")
	}
	if second.SourceEventTime.Equal(first.SourceEventTime) {
		t.Fatal("source times should remain distinct")
	}
}

func TestIndependentSymbols(t *testing.T) {
	authority := New("run1")
	now := time.Now().UTC()
	aapl := authority.Accept(bar("AAPL", now, "a"))
	msft := authority.Accept(bar("MSFT", now, "m"))
	if aapl.GeneratorSequenceNo != 1 || msft.GeneratorSequenceNo != 1 {
		t.Fatalf("expected independent sequences, got %d %d", aapl.GeneratorSequenceNo, msft.GeneratorSequenceNo)
	}
}

func TestDuplicatePreserved(t *testing.T) {
	authority := New("run1")
	now := time.Now().UTC()
	first := authority.Accept(bar("AAPL", now, "same"))
	second := authority.Accept(bar("AAPL", now, "same"))
	if first.GeneratorSequenceNo != 1 || second.GeneratorSequenceNo != 2 {
		t.Fatalf("duplicate must still receive next sequence, got %d %d", first.GeneratorSequenceNo, second.GeneratorSequenceNo)
	}
	if !second.DuplicateArrival {
		t.Fatal("duplicate evidence flag missing")
	}
}
