// File: engine_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 regression authority
// Product/Component: JEH-HOHO / Source engine tests
// Purpose: Prove persistent authority, bounded blocking, cancellation, and failures.
// Failure meaning: Slice 2 source ownership no longer matches frozen behavior.

package source

import (
	"context"
	"errors"
	"testing"
	"time"

	"jeh-hoho/internal/source/types"
)

type sourceFunc func(context.Context, []string, chan<- types.Observation) error

func (function sourceFunc) Run(ctx context.Context, symbols []string, out chan<- types.Observation) error {
	return function(ctx, symbols, out)
}

type writerFunc func([]types.Observation) error

func (function writerFunc) WriteBatch(observations []types.Observation) error {
	return function(observations)
}

func validObservation(messageType, hash string, eventTime time.Time) types.Observation {
	return types.Observation{
		Symbol: "AAPL", SourceEventTime: eventTime, SourceTimestampText: eventTime.Format(time.RFC3339Nano),
		ReceivedTime: eventTime.Add(time.Second), Interval: types.Interval1Min,
		Open: 1, High: 2, Low: 1, Close: 1.5, Volume: 10, EventCount: 3,
		SourceID: "ALPACA_IEX", AlpacaMessageType: messageType, PayloadHash: hash,
	}
}

func TestEngineAuthoritySurvivesSourceSessionBoundary(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstTime := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	source := sourceFunc(func(ctx context.Context, _ []string, out chan<- types.Observation) error {
		out <- validObservation("b", "first", firstTime)
		// This second arrival represents delivery after the source's internal reconnect.
		out <- validObservation("u", "second", firstTime.Add(time.Minute))
		<-ctx.Done()
		return ctx.Err()
	})
	engine, err := NewEngine(EngineConfig{CollectionRunID: "collection-1", Feed: "iex", Symbols: []string{"AAPL"}, BufferCapacity: 2, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- engine.Run(ctx) }()
	first, second := <-engine.Accepted(), <-engine.Accepted()
	if first.EntitySequence != 1 || second.EntitySequence != 2 || first.SourceIdentity.CollectionRunId != second.SourceIdentity.CollectionRunId {
		t.Fatalf("authority reset across session boundary: first=%+v second=%+v", first, second)
	}
	if first.AlpacaMessageType != "b" || second.AlpacaMessageType != "u" {
		t.Fatalf("b/u consolidated: %s %s", first.AlpacaMessageType, second.AlpacaMessageType)
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}

func TestEngineBoundedDeliveryBlocksAndCancellationReleases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sentAll := make(chan struct{})
	source := sourceFunc(func(ctx context.Context, _ []string, out chan<- types.Observation) error {
		now := time.Now().UTC()
		// Four arrivals exceed the combined in-flight capacity of the one-slot raw
		// and accepted queues plus the observation currently being processed.
		for index := 0; index < 4; index++ {
			select {
			case out <- validObservation("b", string(rune('a'+index)), now.Add(time.Duration(index)*time.Minute)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		close(sentAll)
		<-ctx.Done()
		return ctx.Err()
	})
	engine, err := NewEngine(EngineConfig{CollectionRunID: "collection-1", Feed: "iex", BufferCapacity: 1, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- engine.Run(ctx) }()
	select {
	case <-sentAll:
		t.Fatal("source crossed bounded pipeline without consumer backpressure")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}

func TestEnginePropagatesPersistenceFailureBeforeDelivery(t *testing.T) {
	want := errors.New("write failed")
	source := sourceFunc(func(ctx context.Context, _ []string, out chan<- types.Observation) error {
		select {
		case out <- validObservation("b", "one", time.Now().UTC()):
		case <-ctx.Done():
			return ctx.Err()
		}
		<-ctx.Done()
		return ctx.Err()
	})
	engine, err := NewEngine(EngineConfig{CollectionRunID: "collection-1", Feed: "iex", BufferCapacity: 1, Source: source, Writer: writerFunc(func([]types.Observation) error { return want })})
	if err != nil {
		t.Fatal(err)
	}
	err = engine.Run(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("failure = %v", err)
	}
	if observation, ok := <-engine.Accepted(); ok || observation != nil {
		t.Fatal("failed persistence must not deliver")
	}
}
