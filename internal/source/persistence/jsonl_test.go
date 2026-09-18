// File: jsonl_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated regression authority
// Product/Component: JEH-HOHO / Source evidence persistence tests
// Purpose: Lock one-object-per-line, persisted-time, and owned-file behavior.
// Origin: Bar_Sequence_Lab/generator/internal/persistence/jsonl_test.go.
// Failure meaning: Source evidence persistence diverged from frozen behavior.

package persistence

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jeh-hoho/internal/source/types"
)

func TestWriteBatchOneObjectPerLine(t *testing.T) {
	directory := t.TempDir()
	writer, err := Open(directory, "run1", "A")
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	now := time.Now().UTC()
	observations := []types.Observation{
		{CollectionRunID: "run1", PartitionID: "A", Symbol: "AAPL", GeneratorSequenceNo: 1, SourceEventTime: now, ReceivedTime: now, Interval: "1Min", Open: 1, High: 1, Low: 1, Close: 1, Volume: 1, SourceID: "ALPACA_TEST", AlpacaMessageType: "b"},
		{CollectionRunID: "run1", PartitionID: "A", Symbol: "AAPL", GeneratorSequenceNo: 2, SourceEventTime: now.Add(time.Minute), ReceivedTime: now, Interval: "1Min", Open: 1, High: 1, Low: 1, Close: 1, Volume: 1, SourceID: "ALPACA_TEST", AlpacaMessageType: "u"},
	}
	if err := writer.WriteBatch(observations); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(directory, "run1", "partition_A.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
		var got types.Observation
		if err := json.Unmarshal(scanner.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.PersistedTime.IsZero() || got.GeneratorSequenceNo == 0 {
			t.Fatalf("%+v", got)
		}
	}
	if count != 2 {
		t.Fatalf("lines=%d", count)
	}
}

func TestInactivePartitionFileNotCreated(t *testing.T) {
	directory := t.TempDir()
	writer, err := Open(directory, "run1", "A")
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	if _, err := os.Stat(filepath.Join(directory, "run1", "partition_B.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("B file should not exist: %v", err)
	}
}
