// File: decode.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated frozen source behavior
// Product/Component: JEH-HOHO / Alpaca Bar decoder
// Purpose: Decode accepted Alpaca bar and updated-bar payloads without consolidation.
// Origin: Bar_Sequence_Lab/generator/internal/alpaca/decode.go, Slice 0 fingerprint.
// Inputs/Outputs: Exact Alpaca JSON messages in; validated source observations out.
// Invariants: T=b and T=u use one decoder and preserve type/raw payload hash separately.
// Failure behavior: Malformed, unsupported, or inconsistent messages return an error.
// Non-responsibilities: Authentication, reconnect, sequencing, persistence, or JEH logic.

package alpaca

import (
	"encoding/json"
	"fmt"
	"time"

	"jeh-hoho/internal/source/types"
)

type control struct {
	Type    string   `json:"T"`
	Message string   `json:"msg"`
	Code    int      `json:"code"`
	Bars    []string `json:"bars"`
}

func decodeMessageArray(raw []byte) ([]json.RawMessage, error) {
	var messages []json.RawMessage
	if err := json.Unmarshal(raw, &messages); err != nil {
		return nil, fmt.Errorf("decode alpaca array: %w", err)
	}
	return messages, nil
}

func decodeControl(raw json.RawMessage) (control, error) {
	fields, err := rawObject(raw)
	if err != nil {
		return control{}, err
	}
	var decoded control
	decoded.Type = rawString(fields, "T")
	decoded.Message = rawString(fields, "msg")
	decoded.Code = rawInt(fields, "code")
	if bars, ok := fields["bars"]; ok {
		_ = json.Unmarshal(bars, &decoded.Bars)
	}
	return decoded, nil
}

func observationFromRaw(raw json.RawMessage, sourceID string, receipt time.Time) (types.Observation, error) {
	fields, err := rawObject(raw)
	if err != nil {
		return types.Observation{}, err
	}
	open, err := requiredFloat(fields, "o")
	if err != nil {
		return types.Observation{}, err
	}
	high, err := requiredFloat(fields, "h")
	if err != nil {
		return types.Observation{}, err
	}
	low, err := requiredFloat(fields, "l")
	if err != nil {
		return types.Observation{}, err
	}
	closePx, err := requiredFloat(fields, "c")
	if err != nil {
		return types.Observation{}, err
	}
	volume, err := requiredUint(fields, "v")
	if err != nil {
		return types.Observation{}, err
	}
	symbol := rawString(fields, "S")
	if symbol == "" {
		return types.Observation{}, fmt.Errorf("alpaca bar missing symbol")
	}
	timestamp := rawString(fields, "t")
	if timestamp == "" {
		return types.Observation{}, fmt.Errorf("alpaca bar missing timestamp")
	}
	start, err := parseAlpacaTime(timestamp)
	if err != nil {
		return types.Observation{}, fmt.Errorf("parse alpaca timestamp %q: %w", timestamp, err)
	}
	if err := validateOHLC(open, high, low, closePx); err != nil {
		return types.Observation{}, err
	}
	messageType := rawString(fields, "T")
	if messageType != "b" && messageType != "u" {
		return types.Observation{}, fmt.Errorf("unsupported alpaca message type %q", messageType)
	}
	trades, _ := optionalUint32(fields, "n")
	return types.Observation{
		Symbol:              symbol,
		SourceEventTime:     start,
		SourceTimestampText: timestamp,
		ReceivedTime:        receipt,
		Interval:            types.Interval1Min,
		Open:                open,
		High:                high,
		Low:                 low,
		Close:               closePx,
		Volume:              volume,
		EventCount:          trades,
		SourceID:            sourceID,
		AlpacaMessageType:   messageType,
		PayloadHash:         types.HashRaw(raw),
	}, nil
}

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func rawString(fields map[string]json.RawMessage, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}
	var parsed string
	if err := json.Unmarshal(value, &parsed); err != nil {
		return ""
	}
	return parsed
}

func rawInt(fields map[string]json.RawMessage, key string) int {
	value, ok := fields[key]
	if !ok {
		return 0
	}
	var parsed int
	if err := json.Unmarshal(value, &parsed); err != nil {
		return 0
	}
	return parsed
}

func requiredFloat(fields map[string]json.RawMessage, key string) (float64, error) {
	value, ok := fields[key]
	if !ok {
		return 0, fmt.Errorf("alpaca bar missing %s", key)
	}
	var parsed float64
	if err := json.Unmarshal(value, &parsed); err != nil {
		return 0, fmt.Errorf("alpaca bar unreadable %s", key)
	}
	return parsed, nil
}

func requiredUint(fields map[string]json.RawMessage, key string) (uint64, error) {
	value, ok := fields[key]
	if !ok {
		return 0, fmt.Errorf("alpaca bar missing %s", key)
	}
	var number json.Number
	if err := json.Unmarshal(value, &number); err == nil {
		parsed, err := number.Int64()
		if err != nil {
			floatValue, floatErr := number.Float64()
			if floatErr != nil || floatValue < 0 {
				return 0, fmt.Errorf("alpaca bar unreadable %s", key)
			}
			return uint64(floatValue), nil
		}
		if parsed < 0 {
			return 0, fmt.Errorf("negative volume")
		}
		return uint64(parsed), nil
	}
	var asString string
	if err := json.Unmarshal(value, &asString); err == nil {
		number = json.Number(asString)
		parsed, err := number.Int64()
		if err != nil || parsed < 0 {
			return 0, fmt.Errorf("alpaca bar unreadable %s", key)
		}
		return uint64(parsed), nil
	}
	return 0, fmt.Errorf("alpaca bar unreadable %s", key)
}

func optionalUint32(fields map[string]json.RawMessage, key string) (uint32, error) {
	if _, ok := fields[key]; !ok {
		return 0, nil
	}
	number, err := requiredUint(fields, key)
	return uint32(number), err
}

func validateOHLC(open, high, low, close float64) error {
	for _, value := range []float64{open, high, low, close} {
		if value <= 0 || value != value {
			return fmt.Errorf("alpaca bar incomplete ohlc")
		}
	}
	if high < low || high < open || high < close || low > open || low > close {
		return fmt.Errorf("alpaca bar inconsistent ohlc")
	}
	return nil
}

func parseAlpacaTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp %q", value)
}
