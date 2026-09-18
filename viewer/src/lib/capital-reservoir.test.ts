// JEH_HOHO product viewer tests; graduated from Capital_Reservoir_Viewer on 2026-09-17.
import assert from "node:assert/strict";
import test from "node:test";
import { deriveCapitalSummary, derivePipeStates, mergeReservoirEvents, normalizeReservoirEvent } from "./capital-reservoir";

const mongoStart = {
  pipeline_run_id: "pipeline-1", collection_run_id: "collection-1", run_type: "RUN_A",
  event_sequence: 1, event_type: "RUN_START", processed_at: new Date("2026-09-14T12:00:00Z"),
  reservoir_before: 3_000_000, reservoir_after: 3_000_000, initial_reservoir: 3_000_000,
  symbol_count: 30, initial_symbol_capital: 100_000, stage_emit_value_id: null,
};

function flow(sequence: number, eventType: "OUTFLOW" | "INFLOW", amount: number, overrides: Record<string, unknown> = {}) {
  return normalizeReservoirEvent({
    ...mongoStart, event_sequence: sequence, event_type: eventType, symbol: "AAPL",
    cause: eventType === "OUTFLOW" ? "HOP_ON" : "HOP_OFF", quantity: 10,
    execution_price: Math.abs(amount) / 10, signed_flow_amount: amount,
    active_quantity_after: eventType === "OUTFLOW" ? 10 : 0,
    reservoir_before: 3_000_000, reservoir_after: 3_000_000 + amount, ...overrides,
  });
}

test("normalizes equivalent Mongo and generated-client JSON events", () => {
  const mongo = normalizeReservoirEvent(mongoStart);
  const grpc = normalizeReservoirEvent({
    pipelineRunId: "pipeline-1", collectionRunId: "collection-1", runType: "RUN_A",
    eventSequence: 1, eventType: "CAPITAL_RESERVOIR_EVENT_TYPE_RUN_START",
    processedUnixMs: 1789387200000, reservoirBefore: 3_000_000, reservoirAfter: 3_000_000,
    initialReservoir: 3_000_000, symbolCount: 30, initialSymbolCapital: 100_000,
  });
  assert.deepEqual({ ...grpc, eventId: null }, mongo);
});

test("deduplicates by pipeline and sequence and orders events", () => {
  const start = normalizeReservoirEvent(mongoStart);
  const end = normalizeReservoirEvent({ ...mongoStart, event_sequence: 3, event_type: "RUN_END" });
  assert.deepEqual(mergeReservoirEvents([end, start, start]).map((event) => event.eventSequence), [1, 3]);
});

test("derives active and completed capital cycles from confirmed flows", () => {
  const active = derivePipeStates([flow(2, "OUTFLOW", -99_848.54)], ["AAPL"])[0];
  assert.equal(active.capitalDeployed, 99_848.54);
  assert.equal(active.capitalReturned, null);
  const complete = derivePipeStates([flow(2, "OUTFLOW", -99_800), flow(3, "INFLOW", 99_920)], ["AAPL"])[0];
  assert.equal(complete.capitalReturned, 99_920);
  assert.equal(complete.realizedPnl, 120);
});

test("preserves replay prefixes and authoritative RUN_END metrics", () => {
  const prefix = deriveCapitalSummary([normalizeReservoirEvent(mongoStart), flow(2, "OUTFLOW", -99_800)], ["AAPL"]);
  assert.equal(prefix.totalMarkedCapital, null);
  const end = normalizeReservoirEvent({ ...mongoStart, event_sequence: 3, event_type: "RUN_END", total_marked_capital_after: 3_000_200, total_pnl: 200 });
  const complete = deriveCapitalSummary([normalizeReservoirEvent(mongoStart), flow(2, "OUTFLOW", -99_800), end], ["AAPL"]);
  assert.equal(complete.totalMarkedCapital, 3_000_200);
  assert.equal(complete.totalPnl, 200);
});

test("preserves safety liquidation causality", () => {
  const state = derivePipeStates([flow(2, "OUTFLOW", -99_800), flow(3, "INFLOW", 99_770, { cause: "SAFETY_LIQUIDATION" })], ["AAPL"])[0];
  assert.equal(state.completedCycles[0].exitCause, "SAFETY_LIQUIDATION");
  assert.equal(state.completedCycles[0].realizedPnl, -30);
});