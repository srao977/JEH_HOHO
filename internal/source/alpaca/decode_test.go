// File: decode_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated regression authority
// Product/Component: JEH-HOHO / Alpaca decoder tests
// Purpose: Lock b/u separation, payload hashing, and malformed-message rejection.
// Origin: Bar_Sequence_Lab/generator/internal/alpaca/decode_test.go.
// Failure meaning: Source behavior changed and Slice 2 must not progress.

package alpaca

import (
	"testing"
	"time"
)

func TestObservationFromRawPreservesBU(t *testing.T) {
	receipt := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	for _, messageType := range []string{"b", "u"} {
		raw := []byte(`{"T":"` + messageType + `","S":"AAPL","o":1,"h":2,"l":1,"c":1.5,"v":10,"t":"2026-09-10T12:00:00Z","n":3}`)
		observation, err := observationFromRaw(raw, "ALPACA_IEX", receipt)
		if err != nil {
			t.Fatal(err)
		}
		if observation.AlpacaMessageType != messageType {
			t.Fatalf("type=%s", observation.AlpacaMessageType)
		}
		if observation.Symbol != "AAPL" || observation.Volume != 10 || observation.Interval != "1Min" {
			t.Fatalf("%+v", observation)
		}
		if observation.PayloadHash == "" {
			t.Fatal("missing payload hash")
		}
	}
}

func TestMalformedRejected(t *testing.T) {
	receipt := time.Now().UTC()
	cases := []string{
		`{"T":"b","S":"","o":1,"h":2,"l":1,"c":1,"v":1,"t":"2026-09-10T12:00:00Z"}`,
		`{"T":"b","S":"AAPL","o":1,"h":2,"l":1,"c":1,"v":1,"t":""}`,
		`{"T":"b","S":"AAPL","o":0,"h":2,"l":1,"c":1,"v":1,"t":"2026-09-10T12:00:00Z"}`,
		`{"T":"b","S":"AAPL","o":3,"h":2,"l":1,"c":1,"v":1,"t":"2026-09-10T12:00:00Z"}`,
	}
	for _, raw := range cases {
		if _, err := observationFromRaw([]byte(raw), "ALPACA_IEX", receipt); err == nil {
			t.Fatalf("expected reject for %s", raw)
		}
	}
}

func TestDecodeMessageArray(t *testing.T) {
	messages, err := decodeMessageArray([]byte(`[{"T":"success","msg":"authenticated"}]`))
	if err != nil || len(messages) != 1 {
		t.Fatalf("%v %v", messages, err)
	}
	decoded, err := decodeControl(messages[0])
	if err != nil || decoded.Message != "authenticated" {
		t.Fatalf("%+v %v", decoded, err)
	}
}
