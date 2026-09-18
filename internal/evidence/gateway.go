// File: gateway.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 4 implementation
// Product/Component: JEH-HOHO / Product Evidence Gateway
// Purpose: Publish unchanged frozen Capital events through the product contract.
// Inputs/Outputs: Internal RuntimeEvidenceEnvelope in; ProductEvidenceEnvelope out.
// Invariants: Capital identity, sequence, times, values, and causes are not recomputed.
// Failure behavior: Invalid run selection or stream send errors return synchronously.
// Non-responsibilities: Durable replay, retries, source continuity, or Capital arithmetic.

package evidence

import (
	"fmt"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"
)

// RuntimeSource is the frozen runtime bus subscription boundary.
type RuntimeSource interface {
	Subscribe() (<-chan *dsejehv1.RuntimeEvidenceEnvelope, func())
}

// Gateway streams selected frozen runtime evidence through the product contract.
type Gateway struct {
	jehhohov1.UnimplementedProductEvidenceServiceServer
	identity *jehhohov1.ProductIdentity
	source   RuntimeSource
}

// NewGateway creates a product evidence gateway for one product run.
func NewGateway(identity *jehhohov1.ProductIdentity, source RuntimeSource) *Gateway {
	return &Gateway{identity: identity, source: source}
}

// SubscribeEvidence streams mapped Capital events in frozen publication order.
func (gateway *Gateway) SubscribeEvidence(request *jehhohov1.SubscribeEvidenceRequest, stream jehhohov1.ProductEvidenceService_SubscribeEvidenceServer) error {
	if gateway.identity == nil || gateway.source == nil {
		return fmt.Errorf("product evidence gateway is not configured")
	}
	if request.GetProductRunId() != "" && request.GetProductRunId() != gateway.identity.GetProductRunId() {
		return fmt.Errorf("product run %q is not active", request.GetProductRunId())
	}
	publications, unsubscribe := gateway.source.Subscribe()
	defer unsubscribe()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case publication := <-publications:
			capital := publication.GetCapitalReservoirEvent()
			if capital == nil {
				continue
			}
			envelope := &jehhohov1.ProductEvidenceEnvelope{
				ProductIdentity: gateway.identity, PublicationId: publication.GetPublicationId(), PublicationSequence: publication.GetPublicationSequence(),
				EvidenceType:   jehhohov1.EvidenceType_EVIDENCE_TYPE_CAPITAL_RESERVOIR,
				Evidence:       &jehhohov1.ProductEvidenceEnvelope_CapitalReservoir{CapitalReservoir: MapCapital(capital)},
				ProducedUnixMs: publication.GetProducedUnixMs(),
			}
			if err := stream.Send(&jehhohov1.SubscribeEvidenceResponse{Evidence: envelope}); err != nil {
				return err
			}
		}
	}
}

// MapCapital performs a field-for-field product contract mapping.
func MapCapital(event *dsejehv1.CapitalReservoirEvent) *jehhohov1.CapitalReservoirEvidence {
	if event == nil {
		return nil
	}
	return &jehhohov1.CapitalReservoirEvidence{
		EventId: event.GetEventId(), PipelineRunId: event.GetPipelineRunId(), CollectionRunId: event.GetCollectionRunId(), RunType: event.GetRunType(),
		EventSequence: event.GetEventSequence(), Symbol: event.GetSymbol(), EventType: jehhohov1.CapitalReservoirEventType(event.GetEventType()),
		Cause: jehhohov1.CapitalReservoirCause(event.GetCause()), StageEmitValueId: event.GetStageEmitValueId(),
		TriggerEventUnixMs: event.GetTriggerEventUnixMs(), ExecutionEventUnixMs: event.GetExecutionEventUnixMs(), ProcessedUnixMs: event.GetProcessedUnixMs(),
		Quantity: event.GetQuantity(), ExecutionPrice: event.GetExecutionPrice(), SignedFlowAmount: event.GetSignedFlowAmount(), InitialReservoir: event.GetInitialReservoir(),
		SymbolCount: event.GetSymbolCount(), InitialSymbolCapital: event.GetInitialSymbolCapital(), ReservoirBefore: event.GetReservoirBefore(), ReservoirAfter: event.GetReservoirAfter(),
		ActiveQuantityAfter: event.GetActiveQuantityAfter(), ReservoirCash: event.GetReservoirCash(), TotalDeployedAfter: event.GetTotalDeployedAfter(),
		TotalMarkedCapitalAfter: event.GetTotalMarkedCapitalAfter(), RealizedPnl: event.GetRealizedPnl(), UnrealizedPnl: event.GetUnrealizedPnl(),
		TotalPnl: event.GetTotalPnl(), ActivePositions: event.GetActivePositions(),
	}
}
