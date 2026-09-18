// File: consumer.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 3 implementation
// Product/Component: JEH-HOHO / Product source to frozen JEH adapter
// Purpose: Consume AcceptedBarSourceService and map immutable Bars into frozen JEH.
// Inputs/Outputs: Product AcceptedBarObservation stream in; serial dsejeh BarEvent callback out.
// Invariants: Source identity/sequence/finality are preserved; no consolidation or replay.
// Failure behavior: Callback errors return immediately; transport retries are bounded.
// Non-responsibilities: Source acquisition, gap fill, resume, JEH mathematics, or persistence.

package product

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	"jeh-hoho/internal/jeh/analytical"
	"jeh-hoho/internal/jeh/evidence"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	initialBackoff  = 200 * time.Millisecond
	maximumBackoff  = 3 * time.Second
	maximumAttempts = 5
)

// Consumer owns the local product source subscription for one JEH runtime.
type Consumer struct {
	address    string
	consumerID string
}

// New creates a product source adapter without opening a connection.
func New(address, consumerID string) *Consumer {
	return &Consumer{address: address, consumerID: consumerID}
}

// Run implements app.Source with serial callback invocation in received order.
func (consumer *Consumer) Run(ctx context.Context, emit func(*dsejehv1.BarEvent) error) error {
	return consumer.runWithDialer(ctx, nil, emit)
}

func (consumer *Consumer) runWithDialer(ctx context.Context, dialer func(context.Context, string) (net.Conn, error), emit func(*dsejehv1.BarEvent) error) error {
	options := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if dialer != nil {
		options = append(options, grpc.WithContextDialer(dialer))
	}
	connection, err := grpc.NewClient(consumer.address, options...)
	if err != nil {
		return fmt.Errorf("create product source client: %w", err)
	}
	defer connection.Close()
	client := jehhohov1.NewAcceptedBarSourceServiceClient(connection)
	backoff := initialBackoff
	for attempt := 0; attempt < maximumAttempts; attempt++ {
		stream, streamErr := client.StreamAcceptedBars(ctx, &jehhohov1.StreamAcceptedBarsRequest{ConsumerId: consumer.consumerID})
		if streamErr == nil {
			for {
				response, receiveErr := stream.Recv()
				if receiveErr == nil {
					event, mapErr := MapAcceptedBar(response.GetObservation())
					if mapErr != nil {
						return mapErr
					}
					if err := emit(event); err != nil {
						return err
					}
					continue
				}
				if receiveErr == io.EOF {
					streamErr = fmt.Errorf("product source stream ended")
				} else {
					streamErr = receiveErr
				}
				break
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt == maximumAttempts-1 {
			return fmt.Errorf("product source stream failed after %d attempts: %w", maximumAttempts, streamErr)
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff = min(backoff*2, maximumBackoff)
	}
	return nil
}

// MapAcceptedBar maps one product observation to the frozen internal JEH boundary.
func MapAcceptedBar(observation *jehhohov1.AcceptedBarObservation) (*dsejehv1.BarEvent, error) {
	if observation == nil || observation.GetSourceIdentity() == nil {
		return nil, fmt.Errorf("accepted Bar source identity is required")
	}
	symbol := strings.ToUpper(strings.TrimSpace(observation.GetSymbol()))
	if symbol == "" || observation.GetObservationId() == "" || observation.GetEntitySequence() == 0 {
		return nil, fmt.Errorf("accepted Bar identity is incomplete")
	}
	volume := observation.GetVolume()
	eventCount := observation.GetEventCount()
	source := observation.GetSourceIdentity()
	return &dsejehv1.BarEvent{
		EventId:             evidence.ID("bar-event", observation.GetObservationId(), observation.GetPayloadHash()),
		EntityId:            symbol,
		Symbol:              symbol,
		Interval:            observation.GetInterval(),
		IntervalStartUnixMs: observation.GetIntervalStartUnixMs(),
		IntervalEndUnixMs:   observation.GetIntervalEndUnixMs(),
		Open:                observation.GetOpen(),
		High:                observation.GetHigh(),
		Low:                 observation.GetLow(),
		Close:               observation.GetClose(),
		Volume:              &volume,
		EventCount:          &eventCount,
		Provenance: &dsejehv1.SourceProvenance{
			SourceMode:          dsejehv1.RuntimeMode_RUNTIME_MODE_ONLINE,
			SourceId:            source.GetSourceId(),
			SourceObservationId: observation.GetObservationId(),
			CollectionRunId:     source.GetCollectionRunId(),
			EntitySequence:      observation.GetEntitySequence(),
			EntitySequenceScope: analytical.Scope(source.GetCollectionRunId(), symbol),
			SourceTimestamp:     observation.GetSourceTimestamp(),
			SourceEventUnixMs:   observation.GetSourceEventUnixMs(),
			ReceivedUnixMs:      observation.GetReceivedUnixMs(),
			IsFinal:             observation.GetIsFinal(),
			PayloadHash:         observation.GetPayloadHash(),
		},
	}, nil
}
