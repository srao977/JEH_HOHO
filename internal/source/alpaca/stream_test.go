// File: stream_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 regression authority
// Product/Component: JEH-HOHO / Alpaca stream tests
// Purpose: Prove fresh-session reconnect, dual subscription, b/u delivery, and continuity.
// Inputs: Local websocket fixture only; no live credentials or external network.
// Failure meaning: Acquisition/reconnect behavior diverged from the frozen source.

package alpaca

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"jeh-hoho/internal/source/types"

	"github.com/gorilla/websocket"
)

func TestStreamReconnectsFreshAndPreservesBarsAndUpdatedBars(t *testing.T) {
	var mu sync.Mutex
	connections := 0
	authentications := 0
	subscriptions := 0
	secondConnected := make(chan struct{})
	releaseSecond := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		mu.Lock()
		connections++
		session := connections
		mu.Unlock()
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "connected"}})
		var auth map[string]any
		if connection.ReadJSON(&auth) != nil || auth["action"] != "auth" {
			return
		}
		mu.Lock()
		authentications++
		mu.Unlock()
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "authenticated"}})
		var subscribe map[string]any
		if connection.ReadJSON(&subscribe) != nil {
			return
		}
		bars, barsOK := subscribe["bars"].([]any)
		updated, updatedOK := subscribe["updatedBars"].([]any)
		if subscribe["action"] != "subscribe" || !barsOK || !updatedOK || len(bars) != 1 || len(updated) != 1 {
			return
		}
		mu.Lock()
		subscriptions++
		mu.Unlock()
		_ = connection.WriteJSON([]map[string]any{{"T": "subscription", "bars": []string{"AAPL"}}})
		if session == 2 {
			close(secondConnected)
			<-releaseSecond
		}
		messageType := "b"
		if session == 2 {
			messageType = "u"
		}
		payload := []map[string]any{{"T": messageType, "S": "AAPL", "o": 1, "h": 2, "l": 1, "c": 1.5, "v": 10, "t": "2026-09-17T12:00:00Z", "n": 3}}
		_ = connection.WriteJSON(payload)
	}))
	defer server.Close()

	streamURL := "ws" + strings.TrimPrefix(server.URL, "http")
	stream := NewStream(streamURL, "iex", Credentials{Key: "key", Secret: "secret"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observations := make(chan types.Observation, 2)
	result := make(chan error, 1)
	go func() { result <- stream.Run(ctx, []string{"AAPL"}, observations) }()

	first := <-observations
	select {
	case <-secondConnected:
	case <-time.After(2 * time.Second):
		t.Fatal("second fresh session was not established")
	}
	if health := stream.Health(); health.Continuity != "unknown" {
		t.Fatalf("disconnected continuity = %q", health.Continuity)
	}
	close(releaseSecond)
	second := <-observations
	cancel()
	<-result

	if first.AlpacaMessageType != "b" || second.AlpacaMessageType != "u" || first.PayloadHash == second.PayloadHash {
		t.Fatalf("separate b/u arrivals not preserved: first=%+v second=%+v", first, second)
	}
	mu.Lock()
	defer mu.Unlock()
	if connections != 2 || authentications != 2 || subscriptions != 2 {
		t.Fatalf("sessions=%d auth=%d subscriptions=%d", connections, authentications, subscriptions)
	}
}

func TestPreflightUsesProductionAuthenticationAndDualSubscription(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	verified := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return
		}
		defer connection.Close()
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "connected"}})
		var auth map[string]any
		if connection.ReadJSON(&auth) != nil || auth["action"] != "auth" || auth["key"] != "key" || auth["secret"] != "secret" {
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "success", "msg": "authenticated"}})
		var subscription map[string]any
		if connection.ReadJSON(&subscription) != nil || subscription["action"] != "subscribe" {
			return
		}
		if len(subscription["bars"].([]any)) != 1 || len(subscription["updatedBars"].([]any)) != 1 {
			return
		}
		_ = connection.WriteJSON([]map[string]any{{"T": "subscription", "bars": []string{"AAPL"}}})
		close(verified)
	}))
	defer server.Close()
	stream := NewStream("ws"+strings.TrimPrefix(server.URL, "http"), "iex", Credentials{Key: "key", Secret: "secret"})
	if err := stream.Preflight(context.Background(), []string{"AAPL"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-verified:
	case <-time.After(time.Second):
		t.Fatal("production auth/subscription path was not used")
	}
}
