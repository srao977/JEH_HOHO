# JEH-HOHO Code-Derived Productization Behavior

| Metadata | Value |
|---|---|
| Date | 2026-09-17 |
| Version | 1.0 |
| Status | Verified Investigation |
| Product | JEH-HOHO |
| Component | Product behavior lineage |
| Document Type | Source-Code / Test / Runtime-Evidence Forensics |
| Authority | Chino Rao, Human Authority; executing reference code is behavioral evidence |

## Purpose

Resolve HD-01 through HD-09 from executing implementation, tests, fixtures, actual call paths, configuration/defaults, and persisted evidence. This investigation distinguishes established behavior from new product observability and removes artificial human decision gates without redesigning the source, JEH, Runtime Evidence, Capital Reservoir, or Viewer.

## Executive Overview

All nine former human-decision items have deterministic productization baselines. None satisfies the five conditions for `GENUINELY UNRESOLVED`.

The Bar Sequence source subscribes to both Alpaca bars (`b`) and updated bars (`u`), converts both through the same decoder, assigns each accepted arrival its own per-symbol sequence, and does not consolidate or replace observations. Reconnect creates a new WebSocket session and subscriptions with no gap fill, while the engine-owned sequence authority and collection identity survive. Delivery and persistence are bounded-channel or synchronous/blocking operations with established cancellation and error propagation rather than an unspecified queue policy.

DSE_JEH generates runtime and default pipeline identity, accepts configured collection/run metadata, and validates Capital events through the existing reservoir path. Productization mechanically owns configuration of those existing identities. Existing online composition omits the trace and reservoir writers; real-time Capital therefore requires injection of the same writer interfaces and reservoir used by validated offline composition. Those writers perform synchronous single-attempt persistence and propagate failure.

The Viewer replays Capital Reservoir events, not source Bars. Local V1 uses existing loopback defaults. Readiness is listener/source-adapter readiness before the first Bar, with no market-freshness rule. The verified collection identity `20260911T161623Z-1` is sufficient machine lineage; the unverified human label `Run C` has no runtime or implementation consequence.

## Scope

- `C:\Users\chino\tramuthus\Bar_Sequence_Lab`
- `C:\Users\chino\tramuthus\DSE_JEH_Lab`
- `C:\Users\chino\tramuthus\Capital_Reservoir_Viewer`
- Tests, fixtures, configuration/defaults, composition, persistence, and exported evidence directly relevant to HD-01 through HD-09.

## Out of Scope

- Product implementation.
- Changes to reference repositories.
- New source finalization, resume, queue, persistence, identity, security, or freshness semantics.
- Reconsideration of the approved single `JEH_HOHO.proto` and single Buf architecture.

## Dependencies / References

- `docs/design/JEH_HOHO_DEFINITIVE_PRODUCT_SYSTEM_DESIGN_V1_0_2026-09-17.md`
- `docs/design/JEH_HOHO_ARTIFACT_AND_DOCUMENTATION_STANDARD_V1_0_2026-09-17.md`
- Reference source paths cited below are external to this repository and were inspected read-only.

## Assumptions

- Executing implementation outranks earlier descriptive documentation unless a later explicit human decision supersedes it.
- Existing deterministic limitations are behavior to preserve and expose, not missing policies.
- Product-generated configuration may remove operator ceremony without changing identity semantics.

## Invariants / Frozen Decisions

1. Frozen JEH and Capital semantics are unchanged.
2. Existing Bar Sequence acceptance and sequencing behavior is unchanged.
3. Productization may expose existing states but may not claim guarantees the code does not provide.
4. JEH-HOHO V1 retains one `api/proto/jeh_hoho/v1/JEH_HOHO.proto`, package `jeh_hoho.v1`, one product proto module, one `buf.yaml`, one `buf.gen.yaml`, and one deterministic generation operation.
5. No conclusion below authorizes implementation.

## Method and Resolution Hierarchy

Evidence was evaluated in this order:

1. executing implementation;
2. tests proving behavior;
3. fixtures and golden evidence;
4. runtime composition and call paths;
5. configuration/defaults;
6. persisted runtime evidence;
7. documentation.

An item could remain `GENUINELY UNRESOLVED` only if implementation, tests/fixtures, runtime evidence, and approved product constraints all failed to determine behavior and implementation required selection between new semantic alternatives. No item met that standard.

## HD-01: Accepted Alpaca `b` and `u` Behavior

**Disposition: CODE-DERIVED**

**Baseline:** Subscribe to both `bars` and `updatedBars`. Accept `T=b` and `T=u` through the same path as separate immutable arrivals. Do not consolidate, replace, or filter one type. Assign each arrival a per-symbol sequence. Preserve message type and raw-payload hash. Persist each accepted observation. At the established DSE offline boundary, accepted persisted observations map with `is_final=true`.

**Call path and evidence:**

- `Bar_Sequence_Lab/generator/internal/alpaca/stream.go`, `Stream.subscribe`: sends one subscription containing both `bars` and `updatedBars` for the configured symbols.
- Same file, `Stream.readLoop`: ignores message types other than `b` and `u`; both accepted types call `observationFromRaw` and are sent on the same output channel.
- `generator/internal/alpaca/decode.go`, `observationFromRaw`: validates `T` as `b` or `u`, writes it to `Observation.AlpacaMessageType`, hashes the complete raw message into `PayloadHash`, and creates no finality or consolidation state.
- `generator/internal/runtime/engine.go`, `Engine.Run` and `dispatch`: one engine-owned `sequence.Authority` accepts observations after decode and routes them to the selected partition.
- `generator/internal/sequence/authority.go`, `Authority.Accept`: assigns the next per-symbol sequence to every arrival, stamps the same collection run, and flags duplicate payload/time regression without removing the observation.
- `generator/internal/pipeline/partition.go`: accepted observations are written as records; there is no update-in-place path.
- `generator/internal/alpaca/decode_test.go`, `TestObservationFromRawPreservesBU`: proves both message types are preserved.
- `generator/internal/sequence/authority_test.go`, independent-symbol, accepted-arrival-order, and duplicate-preservation tests: prove sequence and duplicate behavior.
- `DSE_JEH_Lab/internal/input/offline/producer.go`, offline mapping: maps each persisted accepted observation to `BarEvent` with `SourceProvenance.IsFinal=true`.
- `DSE_JEH_Lab/exports/offline_20260911T161623Z-1_phase_evidence.jsonl`: validated historical events carry `is_final=true`.

**Product observability only:** expose `alpaca_message_type` and accepted identity in diagnostics. Do not infer a new finalization policy or consolidate arrivals.

**Contradiction resolved:** earlier design language treated missing generator `IsFinal` as an open policy. The executing generator defines acceptance, and the validated replay mapper defines how accepted persisted observations enter JEH. The baseline is both arrivals preserved and mapped as accepted/final at the existing DSE boundary.

## HD-02: Reconnect and Continuity

**Disposition: CODE-DERIVED**

**Baseline:** On dial, authentication, subscription, read, heartbeat, or stream error, end the current WebSocket session, wait with capped exponential backoff, create a fresh connection, authenticate, and recreate both subscriptions. Perform no gap fill and use no resume token. Preserve the engine-owned collection identity and per-symbol sequence authority across reconnect. Do not claim that missing observations can be detected or recovered.

**Call path and evidence:**

- `Bar_Sequence_Lab/generator/internal/alpaca/stream.go`, `Stream.Run`: owns the reconnect loop, state transitions, capped backoff, cancellation, and explicit `no gap fill` behavior.
- Same file, `Stream.session`: dials a fresh socket, authenticates, subscribes, starts the read loop, and exits on read/heartbeat/cancellation error.
- `generator/internal/runtime/engine.go`, `NewEngine`: creates collection `runID` and one `sequence.Authority` before any source session.
- Same file, `Engine.Run`: invokes the reconnecting source while retaining the engine and partition instances; reconnect does not recreate sequence authority.

There is no connection ID in accepted-observation identity, no replay request, no resume token, and no source-gap detector. Sequence numbers continue for arrivals observed after reconnection; they cannot prove that the market emitted nothing during disconnection.

**Product observability only:** report `RECONNECTING` and `CONTINUITY UNKNOWN` after a disconnected interval. These labels expose existing truth and add no resume semantics.

## HD-03: Identity Creation and Ownership

**Disposition: PRODUCTIZATION-DERIVED**

**Baseline:** Preserve existing identity algorithms and relationships while the controller supplies or generates their existing configuration inputs without operator ceremony.

| Identity | Existing authority and behavior |
|---|---|
| Collection run | `Bar_Sequence_Lab/generator/internal/runtime/engine.go`, `NewEngine`: generates UTC `YYYYMMDDTHHMMSSZ-<process sequence>` and creates one authority for the engine lifetime |
| Per-symbol sequence | `sequence.Authority.Accept`: generated independently per symbol within collection scope |
| Source identity | `alpaca.observationFromRaw`: supplied from configured feed through `config.SourceID` |
| Source observation/payload identity | Raw Alpaca payload hash plus collection/symbol/sequence mapping in the DSE offline producer |
| Runtime ID | `DSE_JEH_Lab/internal/app/application.go`, `New`: generated from startup time with `evidence.ID("runtime", timestamp)` |
| Pipeline run ID | Same function: uses configured value or derives `evidence.ID("pipeline-run", collectionRunID, startup timestamp)` |
| Run type | Existing configuration value; `CapitalReservoirEvent.Validate` enforces the existing run-type vocabulary whenever Capital events are produced |
| Trace/stage identity | Mongo trace writer returns the inserted document `ObjectID`; pipeline passes it to execution-derived Capital events |
| Capital event identity | `CapitalReservoirEvent.Proto`: derives ID from pipeline run and event sequence |

The normal operator supplies none of these. The product starts the source, obtains its generated collection identity, supplies that existing value to JEH composition, leaves pipeline/runtime IDs to existing derivation unless explicitly configured by the package, and uses an existing valid run-type configuration. This is automation of established inputs, not a new identity model.

## HD-04: Trace and Capital Writers

**Disposition: CODE-DERIVED**

**Baseline:** A Capital-producing run uses the existing trace writer and Capital Reservoir writer synchronously. The product online composition injects those existing interfaces and constructs the existing reservoir exactly as validated offline composition does. Mongo is the current writer implementation, not part of Capital arithmetic; successful writing is nevertheless part of current event commit behavior.

**Call path and evidence:**

- `DSE_JEH_Lab/internal/app/application.go`, `New`: offline production composition opens Mongo trace and reservoir writers; any mode constructs a reservoir when `Options.ReservoirWriter` is non-nil. Default online composition supplies neither and therefore produces no Capital events.
- `internal/app/pipeline.go`, execution path: writes execution trace first through `traceWithID`; writer failure returns immediately. A nil trace writer returns `bson.NilObjectID`.
- Same path: if reservoir is nil, Capital recording is skipped. Otherwise the stage object ID is passed to `CapitalReservoir.RecordExecution`.
- `internal/execution/capital_reservoir.go`, `RecordExecution` and `CapitalReservoirEvent.Validate`: execution-derived events require a nonzero stage ID and existing governed execution fields.
- Same file, `persist`: validates first, calls the writer synchronously when present, wraps writer failure, and mutates reservoir state only after successful persistence.
- `internal/adapter/mongo/trace_writer.go`: one synchronous `InsertOne` supplies the stage `ObjectID`.

The product assembly change is mechanical: provide the existing writers to online composition so the existing constructor, validation, ordering, IDs, state mutation, error propagation, and event publication path operate unchanged. A fake/discard writer would break established stage identity and evidence behavior.

## HD-05: Blocking, Backpressure, Failure, and Cancellation

**Disposition: CODE-DERIVED**

**Baseline:** Preserve existing bounded-channel and synchronous/blocking behavior. Do not add a new asynchronous queue, dropping policy, or retry protocol.

- Alpaca `Stream.readLoop` blocks when sending to the engine ingest channel once its configured buffer is full; cancellation releases the send.
- `Engine.Run` dispatches each observation to its selected partition. A dispatch error is logged `CRITICAL`, increments the dropped-valid metric, cancels the source, and ends normal dispatch.
- Partition ingress is bounded by existing configuration. Partition processing calls sequence authority and writers in order.
- Required JSONL source persistence is synchronous through the partition worker. Optional Mongo persistence uses the existing writer path; errors propagate through partition/engine behavior. There is no semantic drop-and-continue policy to invent.
- DSE trace Mongo persistence performs one synchronous insertion; failure returns from processing.
- Capital persistence performs one synchronous writer call; failure returns before reservoir state is committed.
- DSE evidence bus publication and Viewer stream disconnection do not redefine or roll back Capital semantics. Viewer/SSE cancellation ends that consumer path; processing remains owned by the DSE application lifecycle.
- Context cancellation terminates source reads/sends, closes or drains owned channels according to existing engine shutdown, and causes DSE application shutdown/owned writer close.

Product status may expose `SOURCE BLOCKED`, `EVIDENCE STORE UNAVAILABLE`, or consumer disconnection when those existing conditions can be observed. It must not claim buffering, delivery, retry, or continuity guarantees absent from the code.

## HD-06: Persistence and Viewer Replay

**Disposition: CODE-DERIVED**

**Baseline:** Viewer REPLAY consumes persisted Capital Reservoir events. It does not replay source Bars. Source-Bar persistence is retained for source evidence, audit, regression, and exact source-boundary comparison, but it is not required for normal LIVE processing or existing Viewer REPLAY.

**Call path and evidence:**

- `Capital_Reservoir_Viewer/src/lib/replay-provider.ts`, `readRun`: queries `capital_reservoir_events` by `pipeline_run_id`, orders by event sequence, and returns Capital events.
- Viewer run/event API routes call the replay provider; no source-Bar collection is queried.
- `CapitalReservoirApp.tsx`: REPLAY fetches run Capital events, while LIVE consumes the evidence stream through SSE.
- `DSE_JEH_Lab/internal/input/offline`: source-Bar persistence is used for historical DSE offline validation/reproduction, a separate path from Viewer replay.

The product package therefore requires Capital event persistence for established REPLAY. Source persistence remains part of validation/evidence tooling and may run as the graduated source already does; it is not the live transport or a Viewer dependency.

## HD-07: Local V1 Security Boundary

**Disposition: PRODUCTIZATION-DERIVED**

**Baseline:** Bind JEH-HOHO V1 product services to loopback on the local product machine/VM. Do not introduce remote TLS, LAN, or reverse-proxy behavior.

**Evidence:**

- `DSE_JEH_Lab/internal/config/config.go`: DSE addresses default to `127.0.0.1` ports.
- `DSE_JEH_Lab/internal/app/application.go`, `Run`: binds the configured server address directly.
- `Capital_Reservoir_Viewer/src/lib/grpc-live.ts`: defaults to the DSE loopback server address.
- Existing configuration can override binding, but no authenticated remote-access architecture exists.

The approved local Child's Play target mechanically selects existing loopback behavior. Product preflight validates loopback rather than exposing an unsafe remote override. Remote deployment is outside V1, not a pending V1 decision.

## HD-08: Historical Collection and Run Label

**Disposition: RUNTIME-EVIDENCE-DERIVED**

**Baseline:** Use the verified machine identity `20260911T161623Z-1` as historical lineage. Do not require or claim an unverified `Run C` label.

**Evidence:**

- `DSE_JEH_Lab/exports/offline_20260911T161623Z-1_phase_evidence.jsonl` is named by and internally records the collection identity, source `ALPACA_IEX`, per-symbol sequences, and final provenance.
- DSE offline producer/repository code requires and queries collection identity directly.
- No inspected executable, fixture manifest, export, or configuration requires the text `Run C`.

The missing friendly label is a bounded documentation provenance note. It cannot affect source lookup, event identity, replay, Capital, package behavior, or an implementation gate.

## HD-09: Readiness and Market Idleness

**Disposition: CODE-DERIVED**

**Baseline:** JEH readiness is process/listener and source-adapter readiness, not receipt of a recent market event. The Viewer is available when its production HTTP server is ready and reports its evidence-stream connection separately. There is no market-calendar or last-Bar freshness gate.

**Call path and evidence:**

- `DSE_JEH_Lab/internal/runtime/state.go`, `NewState`: starts lifecycle at `STARTING`, process live, with source status and zero received Bars.
- `DSE_JEH_Lab/internal/app/application.go`, `Run`: binds and starts gRPC, marks the source adapter ready, transitions runtime to `RUNNING`, and publishes status before starting/receiving source Bars.
- Runtime state records last-Bar/activity when a Bar arrives but does not use it to change readiness.
- `Capital_Reservoir_Viewer/CapitalReservoirApp.tsx`: LIVE bridge connection and event state are distinct from historical replay data.

Product aggregation maps these facts to `STARTING`, `CONNECTING`, `READY`, `DEGRADED`, `STOPPING`, and `STOPPED`. No recent Bar is required for `READY`; status separately presents connection, received counts, last observation, evidence connection, and continuity truth.

## Consolidated Disposition

| Former ID | Disposition | Preserved product baseline | Human decision remains |
|---|---|---|---|
| HD-01 | CODE-DERIVED | Preserve both `b` and `u` as separate accepted/final observations; no consolidation | No |
| HD-02 | CODE-DERIVED | Reconnect/resubscribe with persistent sequence authority and no gap fill | No |
| HD-03 | PRODUCTIZATION-DERIVED | Automate existing identity generation/configuration | No |
| HD-04 | CODE-DERIVED | Inject existing synchronous trace/Capital writers into online composition | No |
| HD-05 | CODE-DERIVED | Preserve existing bounded/blocking, single-attempt failure propagation | No |
| HD-06 | CODE-DERIVED | Capital-event persistence supports Viewer REPLAY; source persistence supports evidence/regression | No |
| HD-07 | PRODUCTIZATION-DERIVED | Enforce existing loopback default for local V1 | No |
| HD-08 | RUNTIME-EVIDENCE-DERIVED | Use verified collection identity; omit unverified friendly label | No |
| HD-09 | CODE-DERIVED | Ready before first Bar; expose connection/activity separately; no freshness rule | No |

**Remaining human decisions: 0.**

## Contradictions and Resolution

| Conflict | Resolution |
|---|---|
| Earlier design called deterministic gaps “unresolved” | Executing behavior is now the baseline; limitations become visible status and validation assertions |
| Generator has no `IsFinal`, while DSE offline provenance is final | Preserve generator arrival/type and existing DSE accepted-observation mapping; do not infer Alpaca semantics or consolidate |
| ONLINE default composition has no Capital, while product requires LIVE Capital | Mechanical product assembly injects existing writers/reservoir; Capital code and semantics remain unchanged |
| Earlier design proposed new queue/failure choices | Preserve existing blocking and synchronous failure propagation |
| Earlier design proposed market-calendar freshness | Preserve readiness before first Bar; freshness is not a V1 semantic gate |
| `Run C` label is not evidenced | Use verified collection identity and make no unsupported label claim |

No code/test disagreement was found for the resolved behavior. Earlier documentation was less precise than executing code and is corrected by the definitive design.

## Validation / Acceptance Criteria

1. Existing Alpaca decoder and sequence tests pass and cover both message types, independent symbol sequences, accepted-arrival order, and duplicate preservation.
2. A reconnect test proves the same engine authority continues sequence after a replacement session and confirms no gap-fill request occurs.
3. Golden mapping proves both accepted message types preserve semantic fields and enter the existing JEH boundary using established final provenance.
4. Identity tests prove existing collection, runtime, pipeline, trace, and Capital derivations without operator-entered IDs.
5. Online Capital integration proves trace insertion precedes Capital persistence, stage identity is nonzero, and writer failure stops the existing processing path without state mutation.
6. Blocking/cancellation tests prove configured channel behavior and graceful release on cancellation.
7. Viewer replay tests prove only ordered Capital events are required.
8. Local package tests prove all listeners are loopback and no remote transport security is implied.
9. Readiness tests prove `READY` before first Bar and expose source/evidence activity separately.
10. Historical regression uses collection `20260911T161623Z-1` without depending on a friendly run label.

## Risks / Open Questions

There are no unresolved V1 semantic questions in HD-01 through HD-09. Preserved limitations remain engineering risks to expose and test:

- reconnect cannot prove source continuity and performs no gap fill;
- blocking delivery can propagate slow-consumer pressure;
- synchronous evidence persistence failure stops the affected processing path;
- readiness does not establish market freshness;
- the friendly `Run C` label remains unverified and must not be asserted.

These risks do not authorize redesign during graduation.

## Human Decisions Required

None. No former HD item satisfies all five conditions required to remain a human decision.

## Change Log

| Date | Version | Status | Change | Authority |
|---|---|---|---|---|
| 2026-09-17 | 1.0 | Verified Investigation | Established code-, test-, runtime-evidence-, and productization-derived baselines for HD-01 through HD-09; zero semantic human decisions remain. | Chino Rao |