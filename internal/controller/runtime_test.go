// File: runtime_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 acceptance authority
// Product/Component: JEH-HOHO / Product controller tests
// Purpose: Prove same-path startup, READY, ownership-safe STOP, port release, and restart.
// Responsibilities: Exercise the real Alpaca client, source RPC, DSE_JEH, and status RPC.
// Inputs/Outputs: Local websocket and persistence fixtures only; no external service.
// Invariants: READY requires no first Bar; STOP affects only controller-owned handles.
// Failure meaning: Lifecycle composition or operator safety regressed.

package controller

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/jeh/execution"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type testTraceWriter struct{}

func (*testTraceWriter) Write(context.Context, execution.TraceRecord) (bson.ObjectID, error) {
	return bson.NewObjectID(), nil
}
func (*testTraceWriter) Close(context.Context) error { return nil }

type testReservoirWriter struct {
	mu     sync.Mutex
	events []execution.CapitalReservoirEvent
}

func (writer *testReservoirWriter) Write(_ context.Context, event execution.CapitalReservoirEvent) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.events = append(writer.events, event)
	return nil
}
func (*testReservoirWriter) Close(context.Context) error { return nil }

type testViewer struct {
	errors  chan error
	stopped chan struct{}
	once    sync.Once
}

func newTestViewer() *testViewer {
	return &testViewer{errors: make(chan error), stopped: make(chan struct{})}
}
func (viewer *testViewer) Errors() <-chan error { return viewer.errors }
func (viewer *testViewer) Stop()                { viewer.once.Do(func() { close(viewer.stopped) }) }

func TestProductLifecycleReadyStopAndRestart(t *testing.T) {
	market := newAuthenticatedAlpacaFixture(t)
	defer market.Close()
	home := t.TempDir()
	config := Config{
		ProductVersion: "test", AlpacaStreamURL: "ws" + strings.TrimPrefix(market.URL, "http"), AlpacaFeed: "iex", Symbols: []string{"AAPL"},
		SourceAddress: "127.0.0.1:0", JEHAddress: freeAddress(t), ViewerAddress: "127.0.0.1:3010",
		MongoURI: "mongodb://unused", MongoDatabase: "test", MongoTraceCollection: "trace", MongoReservoirCollection: "capital",
		SourceDataDirectory: filepath.Join(home, "data"), RunType: "RUN_A", StartingCapital: 100_000, AllocationPercent: 1,
		SourceBufferCapacity: 2, Home: home,
	}
	credentials := Credentials{AlpacaKey: "key", AlpacaSecret: "secret"}
	for attempt := 1; attempt <= 2; attempt++ {
		viewer := newTestViewer()
		reservoirWriter := &testReservoirWriter{}
		dependencies := runtimeDependencies{
			traceWriter: &testTraceWriter{}, reservoirWriter: reservoirWriter,
			startViewer: func(Config, io.Writer) (viewerRuntime, error) { return viewer, nil },
		}
		result := make(chan error, 1)
		go func() {
			result <- runConfiguredWith(context.Background(), config, credentials, io.Discard, dependencies)
		}()
		record := waitForState(t, home, "READY")
		connection, err := grpc.NewClient(config.JEHAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatal(err)
		}
		response, err := jehhohov1.NewProductOperationsServiceClient(connection).GetStatus(context.Background(), &jehhohov1.GetStatusRequest{ProductRunId: record.ProductRunID})
		_ = connection.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.GetStatus().GetLifecycleState() != jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_READY || response.GetStatus().GetActivity().GetBarsReceived() != 0 {
			t.Fatalf("attempt %d idle READY status = %+v", attempt, response.GetStatus())
		}
		if err := RequestStop(home); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-result:
			if err != nil {
				t.Fatalf("attempt %d stop: %v", attempt, err)
			}
		case <-time.After(12 * time.Second):
			t.Fatalf("attempt %d did not stop", attempt)
		}
		select {
		case <-viewer.stopped:
		default:
			t.Fatalf("attempt %d Viewer was not stopped", attempt)
		}
		listener, err := net.Listen("tcp", config.JEHAddress)
		if err != nil {
			t.Fatalf("attempt %d did not release JEH port: %v", attempt, err)
		}
		_ = listener.Close()
		reservoirWriter.mu.Lock()
		if len(reservoirWriter.events) != 2 || reservoirWriter.events[0].EventType != execution.CapitalReservoirEventRunStart || reservoirWriter.events[1].EventType != execution.CapitalReservoirEventRunEnd {
			t.Fatalf("attempt %d Capital boundaries = %+v", attempt, reservoirWriter.events)
		}
		reservoirWriter.mu.Unlock()
	}
}

func TestPreflightRejectsOccupiedPortBeforeStartingAnything(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	config := Config{AlpacaStreamURL: "wss://example.test", AlpacaFeed: "iex", Symbols: []string{"AAPL"}, SourceAddress: listener.Addr().String(), JEHAddress: freeAddress(t), ViewerAddress: freeAddress(t), MongoURI: "mongodb://unused", MongoDatabase: "test", MongoTraceCollection: "trace", MongoReservoirCollection: "capital", RunType: "RUN_A", StartingCapital: 1, AllocationPercent: 1, SourceBufferCapacity: 1}
	err = Preflight(context.Background(), config, Credentials{AlpacaKey: "key", AlpacaSecret: "secret"})
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("preflight error = %v", err)
	}
}

func TestStopRequestRequiresCurrentOwnershipToken(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(runtimeDirectory(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeRecord(home, runtimeRecord{Token: "current-token", State: "READY"}); err != nil {
		t.Fatal(err)
	}
	if err := RequestStop(home); err != nil {
		t.Fatal(err)
	}
	if !stopRequested(home, "current-token") || stopRequested(home, "different-token") {
		t.Fatal("stop request was not restricted to the current ownership token")
	}
}

func waitForState(t *testing.T, home, wanted string) runtimeRecord {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(statePath(home))
		if err == nil {
			var record runtimeRecord
			if decodeJSON(statePath(home), &record) == nil && record.State == wanted {
				return record
			}
			_ = content
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("product did not reach %s", wanted)
	return runtimeRecord{}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func newAuthenticatedAlpacaFixture(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "connected"}})
		var auth map[string]any
		if connection.ReadJSON(&auth) != nil || auth["action"] != "auth" {
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "authenticated"}})
		var subscription map[string]any
		if connection.ReadJSON(&subscription) != nil || subscription["action"] != "subscribe" {
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "subscription", "bars": []string{"AAPL"}, "updatedBars": []string{"AAPL"}}})
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}))
}
