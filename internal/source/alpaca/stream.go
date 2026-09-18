// File: stream.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated frozen source behavior
// Product/Component: JEH-HOHO / Alpaca Bar stream
// Purpose: Authenticate, subscribe to bars and updatedBars, and deliver accepted arrivals.
// Origin: Bar_Sequence_Lab/generator/internal/alpaca/stream.go, Slice 0 fingerprint.
// Inputs/Outputs: Alpaca websocket messages in; blocking Observation channel delivery out.
// Invariants: Every reconnect uses a fresh socket, reauthenticates, resubscribes, and
// performs no gap fill or resume. Both T=b and T=u use the same decoder and remain separate.
// Failure behavior: Session errors trigger capped backoff; context cancellation terminates.
// Security: Credentials are written only to the authenticated websocket and never logged.
// Non-responsibilities: Sequencing, persistence, JEH admission, or continuity invention.

package alpaca

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"jeh-hoho/internal/source/types"

	"github.com/gorilla/websocket"
)

const (
	maxSymbolsBasicPlan = 30
	heartbeatInterval   = time.Second
	heartbeatMaxRTT     = 1500 * time.Millisecond
	heartbeatMaxMisses  = 3
	reconnectBase       = 100 * time.Millisecond
	reconnectCap        = 30 * time.Second
	streamReadIdle      = 90 * time.Second
)

// Credentials contains the Alpaca key pair supplied by product configuration.
type Credentials struct {
	Key    string
	Secret string
}

// Health is the stream's current connection evidence without freshness semantics.
type Health struct {
	State             string
	Continuity        string
	LastError         string
	LastMessage       time.Time
	LastPongRTT       time.Duration
	HeartbeatMisses   uint32
	SubscribedSymbols []string
}

// Stream owns Alpaca websocket sessions and reconnection.
type Stream struct {
	url         string
	feed        string
	credentials Credentials
	dialer      *websocket.Dialer

	mu     sync.RWMutex
	health Health
}

// NewStream constructs a stream without opening a network connection.
func NewStream(url, feed string, credentials Credentials) *Stream {
	return &Stream{
		url:         url,
		feed:        feed,
		credentials: credentials,
		dialer:      &websocket.Dialer{HandshakeTimeout: 10 * time.Second},
		health:      Health{State: "failed", Continuity: "unknown"},
	}
}

// Health returns a race-safe snapshot of stream state.
func (s *Stream) Health() Health {
	s.mu.RLock()
	defer s.mu.RUnlock()
	health := s.health
	health.SubscribedSymbols = append([]string(nil), s.health.SubscribedSymbols...)
	return health
}

// Preflight opens one short-lived session through the production authentication and
// subscription path. It consumes no Bars and establishes no replay or continuity claim.
func (s *Stream) Preflight(ctx context.Context, symbols []string) error {
	if len(symbols) == 0 || len(symbols) > maxSymbolsBasicPlan {
		return fmt.Errorf("symbol count must be between 1 and %d", maxSymbolsBasicPlan)
	}
	conn, _, err := s.dialer.DialContext(ctx, s.url, http.Header{})
	if err != nil {
		return fmt.Errorf("connect to Alpaca: %w", err)
	}
	defer conn.Close()
	if err := s.authenticate(conn); err != nil {
		return err
	}
	if err := s.subscribe(conn, symbols); err != nil {
		return err
	}
	return nil
}

// Run reconnects until cancellation and blocks when the downstream channel blocks.
func (s *Stream) Run(ctx context.Context, symbols []string, out chan<- types.Observation) error {
	if len(symbols) > maxSymbolsBasicPlan {
		return fmt.Errorf("symbol cap exceeded: %d > %d", len(symbols), maxSymbolsBasicPlan)
	}
	backoff := reconnectBase
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.setState("recovering", "")
		s.setContinuity("unknown")
		err := s.session(ctx, symbols, out)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.setState("failed", errorString(err))
		log.Printf("alpaca session ended: %v; reconnect in %s (no gap fill)", err, backoff)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if backoff < reconnectCap {
			backoff *= 2
			if backoff > reconnectCap {
				backoff = reconnectCap
			}
		}
	}
}

func (s *Stream) session(ctx context.Context, symbols []string, out chan<- types.Observation) error {
	conn, _, err := s.dialer.DialContext(ctx, s.url, http.Header{})
	if err != nil {
		return fmt.Errorf("dial alpaca stream: %w", err)
	}
	defer conn.Close()
	if err := s.authenticate(conn); err != nil {
		return err
	}
	if err := s.subscribe(conn, symbols); err != nil {
		return err
	}
	s.setSubscribed(symbols)
	s.setState("healthy", "")
	s.setContinuity("continuous")

	misses := 0
	lastPong := time.Now()
	conn.SetPongHandler(func(string) error {
		now := time.Now()
		rtt := now.Sub(lastPong)
		if rtt < 0 {
			rtt = 0
		}
		s.armReadDeadline(conn)
		s.recordPong(rtt)
		if rtt > heartbeatMaxRTT {
			return fmt.Errorf("pong rtt %s exceeds %s", rtt, heartbeatMaxRTT)
		}
		return nil
	})

	ping := time.NewTicker(heartbeatInterval)
	defer ping.Stop()
	readErr := make(chan error, 1)
	go func() { readErr <- s.readLoop(ctx, conn, out) }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-readErr:
			return err
		case tick := <-ping.C:
			lastPong = tick
			if err := conn.WriteControl(websocket.PingMessage, []byte("Bar_Sequence_Lab"), tick.Add(heartbeatInterval)); err != nil {
				misses++
				s.recordMiss(uint32(misses))
				if misses >= heartbeatMaxMisses {
					return fmt.Errorf("heartbeat write failed %d times: %w", misses, err)
				}
				continue
			}
			misses = 0
			s.recordMiss(0)
		}
	}
}

func (s *Stream) authenticate(conn *websocket.Conn) error {
	greeting, err := readControl(conn)
	if err != nil {
		return fmt.Errorf("read alpaca greeting: %w", err)
	}
	if greeting.Type == "error" {
		return controlError("greeting", greeting)
	}
	if greeting.Message != "authenticated" {
		if err := conn.WriteJSON(map[string]string{"action": "auth", "key": s.credentials.Key, "secret": s.credentials.Secret}); err != nil {
			return fmt.Errorf("write auth: %w", err)
		}
		response, err := readControl(conn)
		if err != nil {
			return fmt.Errorf("read auth response: %w", err)
		}
		if response.Type == "error" {
			return controlError("auth", response)
		}
		if response.Message != "authenticated" {
			return fmt.Errorf("alpaca auth incomplete: type=%s msg=%s", response.Type, response.Message)
		}
	}
	s.recordMessage()
	return nil
}

func (s *Stream) subscribe(conn *websocket.Conn, symbols []string) error {
	message := map[string]any{"action": "subscribe", "bars": symbols, "updatedBars": symbols}
	if err := conn.WriteJSON(message); err != nil {
		return fmt.Errorf("write subscribe: %w", err)
	}
	response, err := readControl(conn)
	if err != nil {
		return fmt.Errorf("read subscribe: %w", err)
	}
	if response.Type == "error" {
		return controlError("subscribe", response)
	}
	log.Printf("alpaca subscribed type=%s bars=%v (T=b and T=u preserved; OE-10 open)", response.Type, response.Bars)
	s.recordMessage()
	return nil
}

func readControl(conn *websocket.Conn) (control, error) {
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return control{}, err
	}
	messages, err := decodeMessageArray(raw)
	if err != nil {
		return control{}, err
	}
	if len(messages) == 0 {
		return control{}, fmt.Errorf("empty alpaca control")
	}
	return decodeControl(messages[0])
}

func controlError(stage string, message control) error {
	return fmt.Errorf("alpaca %s error %d %s", stage, message.Code, message.Message)
}

func (s *Stream) readLoop(ctx context.Context, conn *websocket.Conn, out chan<- types.Observation) error {
	source := sourceID(s.feed)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.armReadDeadline(conn)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read alpaca stream: %w", err)
		}
		s.recordMessage()
		messages, err := decodeMessageArray(raw)
		if err != nil {
			log.Printf("reject alpaca array: %v", err)
			continue
		}
		receipt := time.Now().UTC()
		for _, message := range messages {
			controlMessage, _ := decodeControl(message)
			if controlMessage.Type == "error" {
				return fmt.Errorf("alpaca stream error %d %s", controlMessage.Code, controlMessage.Message)
			}
			if controlMessage.Type != "b" && controlMessage.Type != "u" {
				continue
			}
			observation, err := observationFromRaw(message, source, receipt)
			if err != nil {
				log.Printf("reject malformed alpaca bar: %v", err)
				continue
			}
			select {
			case out <- observation:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func sourceID(feed string) string {
	if feed == "test" {
		return "ALPACA_TEST"
	}
	return "ALPACA_IEX"
}

func (s *Stream) armReadDeadline(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(streamReadIdle))
}
func (s *Stream) setState(state, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.State, s.health.LastError = state, lastError
}
func (s *Stream) setSubscribed(symbols []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.SubscribedSymbols = append([]string(nil), symbols...)
}
func (s *Stream) recordMessage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.LastMessage, s.health.HeartbeatMisses = time.Now(), 0
}
func (s *Stream) recordPong(rtt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.LastPongRTT, s.health.LastMessage, s.health.HeartbeatMisses = rtt, time.Now(), 0
}
func (s *Stream) recordMiss(count uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.HeartbeatMisses = count
}

func (s *Stream) setContinuity(continuity string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Continuity = continuity
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
