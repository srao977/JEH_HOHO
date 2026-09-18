// File: server_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 regression authority
// Product/Component: JEH-HOHO / AcceptedBarSourceService tests
// Purpose: Prove exactly-one consumer leasing and cancellation isolation.
// Failure meaning: Slice 2 delivery ownership no longer matches the product contract.

package source

import (
	"context"
	"testing"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testStream struct {
	grpc.ServerStream
	ctx  context.Context
	sent chan *jehhohov1.StreamAcceptedBarsResponse
}

func (stream *testStream) Context() context.Context { return stream.ctx }
func (stream *testStream) Send(response *jehhohov1.StreamAcceptedBarsResponse) error {
	stream.sent <- response
	return nil
}
func (stream *testStream) SetHeader(metadata.MD) error  { return nil }
func (stream *testStream) SendHeader(metadata.MD) error { return nil }
func (stream *testStream) SetTrailer(metadata.MD)       {}
func (stream *testStream) SendMsg(any) error            { return nil }
func (stream *testStream) RecvMsg(any) error            { return nil }

func TestServerAllowsExactlyOneConsumerAndReleasesOnCancellation(t *testing.T) {
	accepted := make(chan *jehhohov1.AcceptedBarObservation)
	server := NewServer(accepted)
	if !server.acquire() {
		t.Fatal("initial consumer lease rejected")
	}
	second := server.StreamAcceptedBars(&jehhohov1.StreamAcceptedBarsRequest{ConsumerId: "jeh-2"}, &testStream{ctx: context.Background(), sent: make(chan *jehhohov1.StreamAcceptedBarsResponse)})
	if status.Code(second) != codes.AlreadyExists {
		t.Fatalf("second consumer = %v", second)
	}
	server.release()

	secondContext, cancelSecond := context.WithCancel(context.Background())
	cancelSecond()
	if err := server.StreamAcceptedBars(&jehhohov1.StreamAcceptedBarsRequest{ConsumerId: "jeh-2"}, &testStream{ctx: secondContext, sent: make(chan *jehhohov1.StreamAcceptedBarsResponse)}); err == nil {
		t.Fatal("released consumer lease should allow a new canceled stream")
	}
	if !server.acquire() {
		t.Fatal("canceled stream did not release consumer lease")
	}
	server.release()
}
