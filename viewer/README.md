# JEH-HOHO Capital Reservoir Viewer

Standalone, read-only Next.js viewer graduated from Capital_Reservoir_Viewer on 2026-09-17.

The app has two source adapters and one normalized browser model:

- **Replay** reads ordered events for the deliberately selected pipeline run from the configured Capital Reservoir MongoDB collection through server-only routes.
- **Live** follows Capital events from the currently running DSE_JEH pipeline after successful persistence. The server uses the generated JEH-HOHO TypeScript gRPC client and bridges typed Capital publications to the browser over SSE; it never substitutes the latest persisted run.
- **KlineCharts** renders common-reservoir continuity and selected-symbol signed flow. Moving either crosshair selects one authoritative event sequence and resolves the related reservoir event plus all 30 pipe states from that prefix.

The viewer does not calculate reservoir mathematics, submit trades, write MongoDB, start JEH-HOHO, or synthesize liquidation at `RUN_END`.

LIVE delivery preserves the runtime commit boundary: DSE_JEH generates the Capital event, the existing writer persists it synchronously, and only successful writes are published through the product gRPC stream. A stream fault remains visible and closes the SSE response so the browser reconnects automatically. REPLAY remains independent and reads only the selected historical run.

## Reservoir Flow Semantics

Presentation clarified 2026-09-15:

- Signed flow is always shown from the common reservoir perspective. A confirmed BUY is negative reservoir flow and a confirmed SELL or liquidation is positive reservoir flow.
- A negative BUY flow is capital deployment into a position, not a loss. The pipe table displays its positive magnitude as **Capital Deployed**.
- A positive SELL flow is capital returned to the reservoir, not automatically profit. The pipe table displays its positive magnitude as **Capital Returned**.
- **Realized P&L** is presented separately for a completed position cycle as capital returned minus capital deployed in the current zero-cost MockExecutor evidence.
- Every pipe value at replay event sequence N is derived only from authoritative events at or before N. Raw `signedFlowAmount` remains available in the event inspector.

## Capital Summary Semantics

UX refined 2026-09-15 around the single common-reservoir identity:

`Initial Capital = Reservoir Cash + Deployed Marked Capital = Total Marked Capital`

- Current reservoir cash comes from the latest event at or before the selected sequence.
- Deployed marked capital, total marked capital, realized P&L, unrealized P&L, total P&L, and utilization are displayed only when the selected prefix includes the authoritative `RUN_END` fields. The viewer does not estimate them earlier.
- Reservoir utilization is `RUN_END totalDeployedAfter / RUN_START initialReservoir`. Reservoir availability is current `reservoirAfter / RUN_START initialReservoir`.
- **Capital Currently Deployed** counts prefix-derived symbols with active quantity. **Symbols Participated** counts distinct symbols with confirmed OUTFLOW or INFLOW events in the selected prefix; it is not the endpoint active-position count.
- The compact pipe topology represents 30 bidirectional connections to one reservoir. It does not represent 30 isolated cash accounts.
- `CapitalReservoirEvent` does not contain the four-state Dynamic Execution state. The viewer therefore exposes the authoritative capital state (`IDLE` or `DEPLOYED`) without guessing or renaming DSE state.

No Dynamic Risk Interceptor fields, scores, gains, or behavior are implemented.

## Run

```powershell
npm install
npm test
npm run lint
npm exec tsc -- --noEmit
npm run build
npm run dev -- -H 127.0.0.1 -p 3010
```

Open `http://127.0.0.1:3010`.

Copy environment values from `.env.example` only when overriding the local defaults. All connection settings are server-only and must never use a `NEXT_PUBLIC_` prefix.

## Routes

- `GET /api/runs` lists persisted pipeline runs using aggregation only.
- `GET /api/runs/[run]/events` returns that run ordered by `event_sequence` using `find` only.
- `GET /api/live` opens the server-side gRPC subscription and emits normalized SSE events.

MongoDB credentials and the gRPC client remain outside the browser bundle.
