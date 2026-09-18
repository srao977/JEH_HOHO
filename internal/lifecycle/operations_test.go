// File: operations_test.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 6 regression authority
// Product/Component: JEH-HOHO / Product lifecycle tests
// Purpose: Prove READY is independent of first-Bar activity and run selection is safe.
// Failure meaning: Product status semantics or current-run isolation regressed.

package lifecycle

import (
	"context"
	"testing"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOperationsReadyWithoutFirstBarAndRejectsDifferentRun(t *testing.T) {
	operations := NewOperations(func() *jehhohov1.ProductStatus {
		return &jehhohov1.ProductStatus{
			ProductIdentity: &jehhohov1.ProductIdentity{ProductRunId: "run-current"},
			LifecycleState:  jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_READY,
			Activity:        &jehhohov1.ProductActivity{},
		}
	})
	response, err := operations.GetStatus(context.Background(), &jehhohov1.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetStatus().GetLifecycleState() != jehhohov1.ProductLifecycleState_PRODUCT_LIFECYCLE_STATE_READY || response.GetStatus().GetActivity().GetBarsReceived() != 0 {
		t.Fatalf("idle READY status = %+v", response.GetStatus())
	}
	_, err = operations.GetStatus(context.Background(), &jehhohov1.GetStatusRequest{ProductRunId: "run-old"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("different run error = %v, want NOT_FOUND", err)
	}
}
