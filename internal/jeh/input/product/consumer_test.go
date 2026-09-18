// File: consumer_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 3 regression authority
// Product/Component: JEH-HOHO / Product source to frozen JEH adapter tests
// Purpose: Prove semantic mapping, serial ordering, and callback failure propagation.
// Failure meaning: Product source and frozen JEH boundary are no longer equivalent.

package product

import (
	"context"
	"errors"
	"net"
	"testing"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func accepted(sequence uint64, messageType string) *jehhohov1.AcceptedBarObservation {
	volume, eventCount := uint64(42), uint32(7)
	return &jehhohov1.AcceptedBarObservation{
		SourceIdentity: &jehhohov1.SourceIdentity{SourceId: "ALPACA_IEX", CollectionRunId: "collection-1", Feed: "iex"},
		ObservationId:  "observation-" + messageType, Symbol: "aapl", EntitySequence: sequence, Interval: "1Min",
		IntervalStartUnixMs: 1_700_000_000_000, IntervalEndUnixMs: 1_700_000_060_000,
		Open: 100, High: 102, Low: 99, Close: 101, Volume: &volume, EventCount: &eventCount,
		SourceTimestamp: "2023-11-14T22:13:20Z", SourceEventUnixMs: 1_700_000_000_000,
		ReceivedUnixMs: 1_700_000_001_000, AlpacaMessageType: messageType, PayloadHash: "payload-" + messageType, IsFinal: true,
	}
}

func TestMapAcceptedBarPreservesFrozenJEHSemantics(t *testing.T) {
	event, err := MapAcceptedBar(accepted(7, "u"))
	if err != nil {
		t.Fatal(err)
	}
	if event.GetSymbol() != "AAPL" || event.GetProvenance().GetEntitySequence() != 7 || !event.GetProvenance().GetIsFinal() {
		t.Fatalf("identity/finality changed: %+v", event)
	}
	if event.GetProvenance().GetCollectionRunId() != "collection-1" || event.GetProvenance().GetPayloadHash() != "payload-u" {
		t.Fatalf("provenance changed: %+v", event.GetProvenance())
	}
	if event.GetOpen() != 100 || event.GetHigh() != 102 || event.GetLow() != 99 || event.GetClose() != 101 || event.GetVolume() != 42 || event.GetEventCount() != 7 {
		t.Fatalf("semantic values changed: %+v", event)
	}
}

type sourceServer struct {
	jehhohov1.UnimplementedAcceptedBarSourceServiceServer
	observations []*jehhohov1.AcceptedBarObservation
}

func (server *sourceServer) StreamAcceptedBars(_ *jehhohov1.StreamAcceptedBarsRequest, stream jehhohov1.AcceptedBarSourceService_StreamAcceptedBarsServer) error {
	for _, observation := range server.observations {
		if err := stream.Send(&jehhohov1.StreamAcceptedBarsResponse{Observation: observation}); err != nil {
			return err
		}
	}
	return nil
}

func TestConsumerInvokesCallbackSeriallyAndPropagatesFailure(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	jehhohov1.RegisterAcceptedBarSourceServiceServer(server, &sourceServer{observations: []*jehhohov1.AcceptedBarObservation{accepted(1, "b"), accepted(2, "u")}})
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	consumer := New("passthrough:///bufnet", "jeh-primary")
	want := errors.New("pipeline stopped")
	sequences := make([]uint64, 0, 2)
	ctx := context.Background()
	err := consumer.runWithDialer(ctx, func(context.Context, string) (net.Conn, error) { return listener.Dial() }, func(event *dsejehv1.BarEvent) error {
		sequences = append(sequences, event.GetProvenance().GetEntitySequence())
		if len(sequences) == 2 {
			return want
		}
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("callback failure = %v", err)
	}
	if len(sequences) != 2 || sequences[0] != 1 || sequences[1] != 2 {
		t.Fatalf("callback order = %v", sequences)
	}
}
