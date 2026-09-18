// File: internal/jeh/execution/mock_executor.go (mock_executor.go) // Date: 2026-09-17 // Version/Status: 1.0 / Graduated frozen JEH behavior // Product/Component: JEH-HOHO / Frozen JEH Engine // Purpose: Preserve the file's existing DSE_JEH implementation or regression. // Origin: DSE_JEH_Lab/internal/jeh/execution/mock_executor.go, Slice 0 fingerprint // Invariants: Governed JEH/Capital/runtime semantics unchanged // Non-responsibilities: No Fin transport and no public product contract.

package execution

import (
	"fmt"
	"strconv"

	"jeh-hoho/internal/jeh/evidence"
	dsejehv1 "jeh-hoho/internal/jeh/gen/dse_jeh/v1"
)

type MockExecutor struct{}

func (MockExecutor) Fill(instruction *dsejehv1.GovernedExecutionInstruction, price float64) (*dsejehv1.ExecutionEvent, error) {
	if instruction == nil || instruction.GetInstructionId() == "" {
		return nil, fmt.Errorf("execution instruction is required")
	}
	if instruction.GetStatus() != dsejehv1.GovernedExecutionInstructionStatus_GOVERNED_EXECUTION_INSTRUCTION_STATUS_ELIGIBLE {
		return nil, fmt.Errorf("execution instruction must be ELIGIBLE")
	}
	if instruction.GetRequestedAction() != dsejehv1.GovernedExecutionAction_GOVERNED_EXECUTION_ACTION_ALLOCATE && instruction.GetRequestedAction() != dsejehv1.GovernedExecutionAction_GOVERNED_EXECUTION_ACTION_LIQUIDATE {
		return nil, fmt.Errorf("mock executor supports ALLOCATE and LIQUIDATE only")
	}
	quantity, err := decimalQuantity(instruction.GetRequestedQuantity())
	if err != nil || quantity == 0 {
		return nil, fmt.Errorf("requested quantity must be a positive whole number")
	}
	if !positiveFinite(price) {
		return nil, fmt.Errorf("fill price must be finite and positive")
	}
	produced := instruction.GetProducedUnixMs()
	quantityValue := &dsejehv1.DecimalValue{Value: strconv.FormatUint(quantity, 10), Unit: "SHARES"}
	return &dsejehv1.ExecutionEvent{
		EventId: evidence.ID("mock-execution-event", instruction.GetInstructionId()), RuntimeId: instruction.GetRuntimeId(),
		EntityId: instruction.GetEntityId(), Status: dsejehv1.ExecutionEventStatus_EXECUTION_EVENT_STATUS_FILLED,
		ExecutorIdentity: &dsejehv1.ExecutorIdentity{ExecutorId: "LOCAL_MOCK_EXECUTOR", ExecutorVersion: "V1", ExecutionMode: dsejehv1.ExecutionMode_EXECUTION_MODE_LOCAL_PAPER, ConfigurationId: instruction.GetConfigurationId()},
		ExternalOrderId:  evidence.ID("mock-order", instruction.GetIdempotencyKey()), RequestedQuantity: quantityValue,
		FilledQuantity: quantityValue, FillPrice: &dsejehv1.DecimalValue{Value: strconv.FormatFloat(price, 'f', -1, 64), Unit: "USD"},
		Reason: "deterministic local paper fill", CorrelationId: instruction.GetCorrelationId(), ExecutionUnixMs: produced,
		ProducedUnixMs: produced, GovernedExecutionInstructionId: instruction.GetInstructionId(),
	}, nil
}
