// File: jsonl.go
// Date: 2026-09-17
// Version/Status: 1.0 / Graduated frozen source behavior
// Product/Component: JEH-HOHO / Source evidence persistence
// Purpose: Persist accepted observations as one synced JSON object per line.
// Origin: Bar_Sequence_Lab/generator/internal/persistence/jsonl.go, Slice 0 fingerprint.
// Inputs/Outputs: Ordered observation batches in; append-only JSONL source evidence out.
// Invariants: Writes flush and fsync synchronously; errors return on the same call.
// Non-responsibilities: Replay ordering, retries, Mongo persistence, or live delivery.

package persistence

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jeh-hoho/internal/source/types"
)

// Stats is a race-safe JSONL writer activity snapshot.
type Stats struct {
	Partition        string
	Path             string
	Persisted        uint64
	Batches          uint64
	FailedBatches    uint64
	LastBatchSize    int
	LastWriteLatency time.Duration
	Pending          int
}

// Writer owns one append-only partition file.
type Writer struct {
	partition string
	path      string
	mu        sync.Mutex
	file      *os.File
	buf       *bufio.Writer
	stats     Stats
}

// Open opens the run/partition JSONL writer and creates only its owned directory.
func Open(dataDir, runID, partition string) (*Writer, error) {
	dir := filepath.Join(dataDir, runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir jsonl dir: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("partition_%s.jsonl", partition))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open jsonl: %w", err)
	}
	return &Writer{partition: partition, path: path, file: file, buf: bufio.NewWriterSize(file, 64*1024), stats: Stats{Partition: partition, Path: path}}, nil
}

// WriteBatch appends, flushes, and synchronizes one ordered batch before returning.
func (w *Writer) WriteBatch(observations []types.Observation) error {
	if len(observations) == 0 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	start := time.Now()
	now := start.UTC()
	for index := range observations {
		observations[index].PersistedTime = now
		line, err := json.Marshal(observations[index])
		if err != nil {
			w.stats.FailedBatches++
			return fmt.Errorf("marshal jsonl: %w", err)
		}
		if _, err := w.buf.Write(line); err != nil {
			w.stats.FailedBatches++
			return fmt.Errorf("write jsonl: %w", err)
		}
		if err := w.buf.WriteByte('\n'); err != nil {
			w.stats.FailedBatches++
			return err
		}
	}
	if err := w.buf.Flush(); err != nil {
		w.stats.FailedBatches++
		return fmt.Errorf("flush jsonl: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		w.stats.FailedBatches++
		return fmt.Errorf("sync jsonl: %w", err)
	}
	w.stats.Persisted += uint64(len(observations))
	w.stats.Batches++
	w.stats.LastBatchSize = len(observations)
	w.stats.LastWriteLatency = time.Since(start)
	return nil
}

// Close flushes buffered data and closes the owned file.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf != nil {
		_ = w.buf.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Snapshot returns current writer statistics with caller-supplied pending depth.
func (w *Writer) Snapshot(pending int) Stats {
	w.mu.Lock()
	defer w.mu.Unlock()
	stats := w.stats
	stats.Pending = pending
	return stats
}

// Path returns the owned JSONL path.
func (w *Writer) Path() string { return w.path }
