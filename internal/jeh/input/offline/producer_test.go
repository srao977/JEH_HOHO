// File: internal/jeh/input/offline/producer_test.go (producer_test.go) // Date: 2026-09-17 // Version/Status: 1.0 / Graduated frozen JEH behavior // Product/Component: JEH-HOHO / Frozen JEH Engine // Purpose: Preserve the file's existing DSE_JEH implementation or regression. // Origin: DSE_JEH_Lab/internal/jeh/input/offline/producer_test.go, Slice 0 fingerprint // Invariants: Governed JEH/Capital/runtime semantics unchanged // Non-responsibilities: No Fin transport and no public product contract.

package offline

import (
	"context"
	"testing"

	"jeh-hoho/internal/jeh/adapter/mongo"
)

type recordingRepository struct {
	collectionRunID string
	observations    []mongo.Observation
}

func (repository *recordingRepository) ReadAll(_ context.Context, collectionRunID string) ([]mongo.Observation, error) {
	repository.collectionRunID = collectionRunID
	var selected []mongo.Observation
	for _, observation := range repository.observations {
		if observation.CollectionRunID == collectionRunID {
			selected = append(selected, observation)
		}
	}
	return selected, nil
}

func TestEventsSelectsConfiguredCollectionRun(t *testing.T) {
	repository := &recordingRepository{observations: []mongo.Observation{
		{CollectionRunID: "20260910T191246Z-1", Symbol: "AAPL", GeneratorSequenceNo: 1},
		{CollectionRunID: "20260911T161623Z-1", Symbol: "AAPL", GeneratorSequenceNo: 1},
		{CollectionRunID: "20260910T190625Z-1", Symbol: "MSFT", GeneratorSequenceNo: 1},
	}}
	events, err := New(repository, "20260911T161623Z-1").Events(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if repository.collectionRunID != "20260911T161623Z-1" {
		t.Fatalf("ReadAll collection run = %q", repository.collectionRunID)
	}
	if len(events) != 1 || events[0].GetProvenance().GetCollectionRunId() != "20260911T161623Z-1" {
		t.Fatalf("Events = %+v", events)
	}
}

func TestEventsRejectsMissingCollectionRun(t *testing.T) {
	repository := &recordingRepository{}
	if _, err := New(repository, " ").Events(context.Background()); err == nil {
		t.Fatal("Events must reject a missing collection run")
	}
	if repository.collectionRunID != "" {
		t.Fatal("Events must not query the repository without a collection run")
	}
}
