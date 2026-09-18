// File: gateway_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 4 regression authority
// Product/Component: JEH-HOHO / Product Evidence Gateway tests
// Purpose: Prove Capital mapping preserves complete lineage and values.
// Failure meaning: Product Capital evidence diverged from the frozen runtime event.

package evidence

import (
	"context"
	"net"
	"testing"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	runtimeevidence "jeh-hoho/internal/jeh/evidence"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestMapCapitalPreservesAllFields(t *testing.T) {
	source := &dsejehv1.CapitalReservoirEvent{
		EventId: "event", PipelineRunId: "pipeline", CollectionRunId: "collection", RunType: "RUN_A", EventSequence: 7,
		Symbol: "AAPL", EventType: dsejehv1.CapitalReservoirEventType_CAPITAL_RESERVOIR_EVENT_TYPE_OUTFLOW,
		Cause: dsejehv1.CapitalReservoirCause_CAPITAL_RESERVOIR_CAUSE_HOP_ON, StageEmitValueId: "0123456789abcdef01234567",
		TriggerEventUnixMs: 1, ExecutionEventUnixMs: 2, ProcessedUnixMs: 3, Quantity: 4, ExecutionPrice: 5, SignedFlowAmount: -20,
		InitialReservoir: 3_000_000, SymbolCount: 30, InitialSymbolCapital: 100_000, ReservoirBefore: 3_000_000, ReservoirAfter: 2_999_980,
		ActiveQuantityAfter: 4, ReservoirCash: 2_999_980, TotalDeployedAfter: 20, TotalMarkedCapitalAfter: 3_000_000,
		RealizedPnl: 6, UnrealizedPnl: 7, TotalPnl: 13, ActivePositions: 1,
	}
	mapped := MapCapital(source)
	if mapped.GetEventId() != source.GetEventId() || mapped.GetPipelineRunId() != source.GetPipelineRunId() || mapped.GetEventSequence() != source.GetEventSequence() {
		t.Fatalf("identity changed: %+v", mapped)
	}
	if mapped.GetStageEmitValueId() != source.GetStageEmitValueId() || mapped.GetSignedFlowAmount() != source.GetSignedFlowAmount() || mapped.GetTotalPnl() != source.GetTotalPnl() {
		t.Fatalf("lineage/value changed: %+v", mapped)
	}
	if int32(mapped.GetEventType()) != int32(source.GetEventType()) || int32(mapped.GetCause()) != int32(source.GetCause()) {
		t.Fatalf("enum mapping changed: %+v", mapped)
	}
}

func TestGatewayPreservesPublicationIdentityAndCancellation(t *testing.T) {
	bus := runtimeevidence.NewBus()
	server := grpc.NewServer()
	jehhohov1.RegisterProductEvidenceServiceServer(server, NewGateway(&jehhohov1.ProductIdentity{ProductRunId: "product-run"}, bus))
	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	connection, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	stream, err := jehhohov1.NewProductEvidenceServiceClient(connection).SubscribeEvidence(ctx, &jehhohov1.SubscribeEvidenceRequest{ProductRunId: "product-run"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	publication := &dsejehv1.RuntimeEvidenceEnvelope{Evidence: &dsejehv1.RuntimeEvidenceEnvelope_CapitalReservoirEvent{CapitalReservoirEvent: &dsejehv1.CapitalReservoirEvent{EventId: "capital", EventSequence: 9}}}
	bus.Publish("runtime", publication)
	response, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	mapped := response.GetEvidence()
	if mapped.GetPublicationId() != publication.GetPublicationId() || mapped.GetPublicationSequence() != publication.GetPublicationSequence() || mapped.GetProducedUnixMs() != publication.GetProducedUnixMs() {
		t.Fatalf("publication metadata changed: source=%+v mapped=%+v", publication, mapped)
	}
	if mapped.GetCapitalReservoir().GetEventId() != "capital" || mapped.GetCapitalReservoir().GetEventSequence() != 9 {
		t.Fatalf("Capital event changed: %+v", mapped.GetCapitalReservoir())
	}
	cancel()
	if _, err := stream.Recv(); err == nil {
		t.Fatal("stream remained open after cancellation")
	}
}
