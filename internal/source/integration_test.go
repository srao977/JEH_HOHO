// File: integration_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 integration authority
// Product/Component: JEH-HOHO / Accepted Bar source pipeline
// Purpose: Prove local Alpaca -> persistent authority -> product gRPC -> one JEH consumer.
// Inputs: In-process websocket and bufconn fixtures; no external network or credentials.
// Failure meaning: Slice 2 end-to-end ownership or contract delivery is broken.

package source

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/source/alpaca"
)

func TestAlpacaEngineGRPCExactlyOneJEHConsumer(t *testing.T) {
	releaseWebsocket := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	websocketServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "connected"}})
		var auth map[string]any
		if connection.ReadJSON(&auth) != nil {
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "authenticated"}})
		var subscription map[string]any
		if connection.ReadJSON(&subscription) != nil {
			return
		}
		if _, hasResume := subscription["resume"]; hasResume {
			t.Error("source subscription introduced a resume token")
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "subscription", "bars": []string{"AAPL"}}})
		_ = connection.WriteJSON([]map[string]any{
			{"T": "b", "S": "AAPL", "o": 10, "h": 12, "l": 9, "c": 11, "v": 20, "t": "2026-09-17T12:00:00Z", "n": 3},
			{"T": "u", "S": "AAPL", "o": 10, "h": 13, "l": 9, "c": 12, "v": 25, "t": "2026-09-17T12:00:00Z", "n": 4},
		})
		<-releaseWebsocket
	}))
	defer websocketServer.Close()

	streamURL := "ws" + strings.TrimPrefix(websocketServer.URL, "http")
	alpacaStream := alpaca.NewStream(streamURL, "iex", alpaca.Credentials{Key: "key", Secret: "secret"})
	engine, err := NewEngine(EngineConfig{CollectionRunID: "collection-live", Feed: "iex", Symbols: []string{"AAPL"}, BufferCapacity: 2, Source: alpacaStream})
	if err != nil {
		t.Fatal(err)
	}
	runContext, cancelRun := context.WithCancel(context.Background())
	engineResult := make(chan error, 1)
	go func() { engineResult <- engine.Run(runContext) }()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	jehhohov1.RegisterAcceptedBarSourceServiceServer(grpcServer, NewServer(engine.Accepted()))
	go func() { _ = grpcServer.Serve(listener) }()
	defer grpcServer.Stop()
	connection, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := jehhohov1.NewAcceptedBarSourceServiceClient(connection)
	clientContext, cancelClient := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelClient()
	firstConsumer, err := client.StreamAcceptedBars(clientContext, &jehhohov1.StreamAcceptedBarsRequest{ConsumerId: "jeh-primary"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := firstConsumer.Recv()
	if err != nil {
		t.Fatal(err)
	}
	secondConsumer, err := client.StreamAcceptedBars(clientContext, &jehhohov1.StreamAcceptedBarsRequest{ConsumerId: "jeh-secondary"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := secondConsumer.Recv(); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("second JEH consumer = %v", err)
	}
	second, err := firstConsumer.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if first.Observation.EntitySequence != 1 || second.Observation.EntitySequence != 2 {
		t.Fatalf("sequence = %d, %d", first.Observation.EntitySequence, second.Observation.EntitySequence)
	}
	if first.Observation.AlpacaMessageType != "b" || second.Observation.AlpacaMessageType != "u" {
		t.Fatalf("message types = %s, %s", first.Observation.AlpacaMessageType, second.Observation.AlpacaMessageType)
	}
	if first.Observation.SourceIdentity.CollectionRunId != "collection-live" || second.Observation.SourceIdentity.CollectionRunId != "collection-live" {
		t.Fatal("collection identity changed across accepted observations")
	}

	cancelRun()
	close(releaseWebsocket)
	select {
	case <-engineResult:
	case <-time.After(2 * time.Second):
		t.Fatal("engine cancellation did not propagate")
	}
}
