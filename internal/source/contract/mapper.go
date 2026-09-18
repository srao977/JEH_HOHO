// File: mapper.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 implementation
// Product/Component: JEH-HOHO / Accepted Bar product adapter
// Purpose: Map graduated source observations to the single product contract.
// Inputs/Outputs: Sequence-stamped Observation plus explicit identity in; protobuf out.
// Invariants: Mapping is lossless; finality is the established accepted-Bar value true.
// Failure behavior: Missing identity or required source fields returns an error.
// Non-responsibilities: Identity invention, sequencing, filtering, replay, or JEH admission.

package contract

import (
	"fmt"
	"strconv"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/source/types"
)

// ObservationID returns the established collection/symbol/sequence source identity.
func ObservationID(observation types.Observation) string {
	return types.HashPayload(
		"offline-observation",
		observation.CollectionRunID,
		observation.Symbol,
		strconv.FormatUint(observation.GeneratorSequenceNo, 10),
	)
}

// ToAcceptedBar maps one sequence-stamped source observation without recomputation.
func ToAcceptedBar(observationID, feed string, observation types.Observation) (*jehhohov1.AcceptedBarObservation, error) {
	if observationID == "" || observation.CollectionRunID == "" || observation.Symbol == "" || observation.GeneratorSequenceNo == 0 {
		return nil, fmt.Errorf("accepted observation identity is incomplete")
	}
	if observation.SourceEventTime.IsZero() || observation.ReceivedTime.IsZero() {
		return nil, fmt.Errorf("accepted observation time is incomplete")
	}
	volume := observation.Volume
	eventCount := observation.EventCount
	return &jehhohov1.AcceptedBarObservation{
		SourceIdentity: &jehhohov1.SourceIdentity{
			SourceId:        observation.SourceID,
			CollectionRunId: observation.CollectionRunID,
			Feed:            feed,
		},
		ObservationId:        observationID,
		Symbol:               observation.Symbol,
		EntitySequence:       observation.GeneratorSequenceNo,
		Interval:             observation.Interval,
		IntervalStartUnixMs:  observation.SourceEventTime.UnixMilli(),
		IntervalEndUnixMs:    observation.SourceEventTime.Add(time.Minute).UnixMilli(),
		Open:                 observation.Open,
		High:                 observation.High,
		Low:                  observation.Low,
		Close:                observation.Close,
		Volume:               &volume,
		EventCount:           &eventCount,
		SourceTimestamp:      observation.SourceTimestampText,
		SourceEventUnixMs:    observation.SourceEventTime.UnixMilli(),
		ReceivedUnixMs:       observation.ReceivedTime.UnixMilli(),
		AlpacaMessageType:    observation.AlpacaMessageType,
		PayloadHash:          observation.PayloadHash,
		DuplicateArrival:     observation.DuplicateArrival,
		SourceTimeRegression: observation.SourceTimeRegression,
		IsFinal:              true,
	}, nil
}
