// File: operations.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 implementation
// Product/Component: JEH-HOHO / Product lifecycle
// Purpose: Serve the controller-owned current product status through the product contract.
// Responsibilities: Return a race-safe snapshot and reject noncurrent product run IDs.
// Inputs/Outputs: Controller status snapshots in; read-only GetStatus responses out.
// Configuration: None; the composition root supplies the snapshot function.
// Dependencies: Product-generated Go contract and gRPC status codes.
// Invariants: READY never depends on receipt or freshness of a market Bar.
// Failure behavior: Missing status is UNAVAILABLE; a different run is NOT_FOUND.
// Concurrency/Lifecycle: Safe when the supplied snapshot function is race-safe.
// Non-responsibilities: Starting, stopping, source repair, or JEH/Capital behavior.

package lifecycle

import (
	"context"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Operations serves a controller-owned product status snapshot.
type Operations struct {
	jehhohov1.UnimplementedProductOperationsServiceServer
	snapshot func() *jehhohov1.ProductStatus
}

// NewOperations creates a read-only product operations service.
func NewOperations(snapshot func() *jehhohov1.ProductStatus) *Operations {
	return &Operations{snapshot: snapshot}
}

// GetStatus returns the current product status without changing lifecycle.
func (operations *Operations) GetStatus(_ context.Context, request *jehhohov1.GetStatusRequest) (*jehhohov1.GetStatusResponse, error) {
	if operations.snapshot == nil {
		return nil, status.Error(codes.Unavailable, "product status is unavailable")
	}
	snapshot := operations.snapshot()
	if snapshot == nil || snapshot.GetProductIdentity() == nil {
		return nil, status.Error(codes.Unavailable, "product status is unavailable")
	}
	if requested := request.GetProductRunId(); requested != "" && requested != snapshot.GetProductIdentity().GetProductRunId() {
		return nil, status.Error(codes.NotFound, "product run is not current")
	}
	return &jehhohov1.GetStatusResponse{Status: snapshot}, nil
}
