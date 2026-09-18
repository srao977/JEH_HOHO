# JEH-HOHO Artifact and Documentation Standard

| Metadata | Value |
|---|---|
| Date | 2026-09-17 |
| Version | 1.0 |
| Status | Authoritative |
| Product | JEH-HOHO |
| Component | Repository-wide |
| Document Type | Design Standard |
| Authority | Chino Rao, Human Authority |

## Purpose

This document establishes the mandatory artifact and documentation standard for the independent JEH-HOHO product repository. It governs formal Markdown documents, source files, protobuf contracts, configuration, scripts, build and packaging files, deployment definitions, and adjacent documentation for formats that cannot contain comments.

## Executive Overview

JEH-HOHO artifacts must state their authority, purpose, boundaries, inputs, outputs, dependencies, invariants, lifecycle, and change history clearly enough to support independent review, packaging, operation, and maintenance. Documentation must explain semantic and architectural intent rather than repeat syntax. The standard is deliberately strict for contracts and long-running components because the product preserves validated JEH behavior and must remain operable through a Child's Play workflow.

## Scope

This standard applies to every new or materially revised artifact owned by JEH-HOHO, including:

- formal Markdown documents;
- Go and other substantive source files;
- protobuf files, services, RPCs, messages, enums, and fields;
- configuration templates and schemas;
- PowerShell and shell scripts;
- build, packaging, and deployment files where comments are supported;
- generated-artifact ownership and regeneration instructions;
- operator and developer guides.

## Out of Scope

- Retrofitting comments into reference repositories under `C:\Users\chino\tramuthus`.
- Rewriting generated files that must retain generator-owned structure.
- Repeating implementation syntax as prose.
- Authorizing product implementation or the product protobuf contract.

## Dependencies / References

- Human design authorization dated 2026-09-17.
- `C:\Users\chino\tramuthus\DSE_JEH_Lab` as the corrected validated JEH reference lineage.
- `C:\Users\chino\tramuthus\Bar_Sequence_Lab` as the Bar Sequence implementation reference.
- `C:\Users\chino\tramuthus\Capital_Reservoir_Viewer` as the specialist Viewer reference.
- `JEH_HOHO_DEFINITIVE_PRODUCT_SYSTEM_DESIGN_V1_0_2026-09-17.md` when approved.

## Assumptions

- Git is the authoritative source-control history.
- Every artifact has one identifiable product owner or component owner.
- Generated artifacts are reproducible from product-owned sources and pinned tools.
- File formats that reject comments are documented by an adjacent authoritative Markdown file.

## Invariants / Frozen Decisions

1. JEH-HOHO is an independent product repository.
2. The corrected reference folder name is `DSE_JEH_Lab`.
3. No artifact may create a runtime dependency on `tramuthus` or Fin_FeedSat_1.
4. Documentation must not redefine frozen JEH mathematics, semantics, or governed transitions.
5. Investigation findings become design authority only through an explicit human decision recorded in an authoritative design.
6. One authoritative document is preferred over overlapping version-fragment documents.
7. Every formal artifact carries a date and status.

## Repository Document Placement

| Directory | Purpose | Must not contain |
|---|---|---|
| `docs/design/` | Authoritative architecture, standards, product and contract designs | Unpromoted forensic claims or implementation completion reports |
| `docs/implementation/` | Approved implementation plans, records, validation results, completion reports | Product authority that has not been approved in design |
| `docs/investigation/` | Audits, source forensics, alternatives, evidence reports | Silent product decisions or implementation authorization |

An investigation may cite evidence and recommend a decision. Only an authoritative design or explicit human record may promote that recommendation into product authority.

## Formal Markdown Document Standard

Every formal Markdown document must contain the following sections or metadata. A section may state `Not applicable` with a reason; it must not be silently omitted.

1. Title
2. Date
3. Version
4. Status
5. Product
6. Component
7. Document Type
8. Authority
9. Purpose
10. Executive Overview
11. Scope
12. Out of Scope
13. Dependencies / References
14. Assumptions
15. Invariants / Frozen Decisions
16. Design or Investigation Body
17. Validation / Acceptance Criteria
18. Risks / Open Questions
19. Human Decisions Required
20. Change Log

### Document Status Vocabulary

Use one of these statuses unless the authoritative design defines a more specific controlled value:

- `Draft`: incomplete and non-authoritative.
- `Proposed`: complete enough for human review but not approved.
- `Authoritative`: approved product authority.
- `Superseded`: retained for history and linked to its replacement.
- `Retired`: no longer applicable and not replaced.

### Document Identity and Filenames

Formal filenames must identify product, purpose, version where appropriate, and date:

`JEH_HOHO_<PURPOSE>_V<MAJOR>_<MINOR>_YYYY-MM-DD.md`

Do not create a new file merely to record a minor revision. Update the authoritative document, increment its version as required, and append its Change Log. Create a successor only for a deliberate major authority boundary or when retention requirements demand immutability.

### Evidence and Decision Language

Documents must distinguish:

- `Verified`: directly supported by identified source, test, build, runtime evidence, or historical artifact.
- `Inferred`: a reasoned conclusion not directly demonstrated.
- `Proposed`: a design recommendation awaiting approval.
- `Human Decision`: an explicit decision by the named human authority.
- `Unresolved`: insufficient authority to select a behavior.

Statements about historical use must identify historical evidence. Code capability alone must be written as `CODE SUPPORTS`; it must not be represented as `HISTORICAL EVIDENCE SHOWS IT WAS USED`.

### Diagrams

Use Mermaid or text diagrams where they materially clarify boundaries, lifecycle, identity, or data flow. Every diagram must agree with the surrounding prose and label frozen versus adaptable boundaries where relevant. A diagram is explanatory and does not override normative text.

### Human Decision Records

Every unresolved item that can affect source semantics, governed behavior, evidence, persistence, reconnect or gap behavior, product contracts, credentials, or operator safety must appear in a Human Decision Table containing:

| Field | Requirement |
|---|---|
| Decision ID | Stable identifier |
| Question | One decision stated precisely |
| Evidence | Verified basis and uncertainty |
| Alternatives | Complete credible choices |
| Consequences | Operational and semantic effect of each choice |
| Recommendation | Design recommendation, not silent authority |
| Implementation blocked | `Yes`, `Partially`, or `No` with scope |
| Human decision | Blank or `Pending` until explicitly decided |

## Source Code File Header Requirements

Every substantive JEH-HOHO source file must begin with a language-appropriate header that states:

- File;
- Date;
- Version and status where appropriate;
- Product;
- Component or module;
- Purpose;
- Responsibilities;
- Inputs;
- Outputs;
- Configuration or parameters;
- Dependencies;
- Invariants;
- Non-responsibilities;
- important error and failure behavior;
- concurrency and lifecycle behavior where applicable;
- Change Log or a repository-approved source-control change reference.

Generated files must use the generator's required header and identify the authoritative source and regeneration command where the format permits. Do not hand-edit generated files.

### Symbol Documentation

Every exported type, interface, service, constant group, variable with semantic authority, and function must be documented according to the language's documentation convention.

Important internal functions must document enough to establish:

- purpose;
- inputs and accepted ranges;
- outputs;
- side effects;
- errors and failure behavior;
- invariants;
- concurrency assumptions and ownership where applicable.

Comments must explain architectural or semantic intent. Comments that merely translate syntax into English are prohibited.

### Long-Running Lifecycle Requirements

Services and long-running components must document:

1. startup and preconditions;
2. readiness criteria;
3. normal running state;
4. dependency and market connection state;
5. retry and reconnect policy;
6. degraded behavior;
7. terminal failure behavior;
8. graceful shutdown;
9. goroutine, thread, process, socket, file, and client ownership;
10. cancellation propagation;
11. bounded queues and backpressure behavior;
12. external dependency ownership.

## Protobuf Contract Standard

Every future JEH-HOHO `.proto` file must begin with a strong file header containing:

- File / Contract Title;
- Date;
- Version;
- Status;
- Product;
- Component / Contract Owner;
- Authority;
- Purpose;
- Executive Overview;
- Inputs;
- Outputs;
- Dependencies;
- Invariants;
- Non-Responsibilities;
- Compatibility / Versioning Notes;
- Security / Trust Boundary Notes where applicable;
- Change Log.

No actual JEH-HOHO proto is authorized by this standard.

### JEH-HOHO V1 Physical Contract Decision

By human architectural decision, JEH-HOHO V1 uses one product-owned protobuf source file:

`api/proto/jeh_hoho/v1/JEH_HOHO.proto`

That file declares `package jeh_hoho.v1;` and contains the logically separated `AcceptedBarSourceService`, `ProductEvidenceService`, and `ProductOperationsService`, plus their common enums, identities, messages, requests, responses, and stream envelopes. Shared definitions have one authority and the three services are versioned together as one product contract.

Compliance statement: package `jeh_hoho.v1;`; one product proto module; one buf.yaml; one buf.gen.yaml; one deterministic generation operation.

JEH-HOHO V1 also uses one product proto module, one authoritative `buf.yaml`, one authoritative `buf.gen.yaml`, and one documented deterministic `buf generate` operation for all required Go and Viewer outputs. Separate physical proto files, per-service Buf modules, or per-service generation configurations require a future demonstrated technical reason and explicit human architectural approval. Contract size alone has no arbitrary numeric splitting threshold.

This decision simplifies the maintainer path to one contract edit, one generation operation, and validation of one coherent generated surface. Protobuf and Buf remain invisible to the normal Child's Play operator. It does not weaken any file, service, RPC, message, enum, or field documentation requirement.

### Package and Versioning Rules

- Use the approved product-owned V1 package `jeh_hoho.v1`.
- A package major version represents wire and semantic compatibility, not merely release branding.
- Existing field numbers must never be reused.
- Removed fields must be reserved by number and name.
- Additive compatibility does not authorize a semantic change to an existing field.
- Generated code is a package artifact and must never depend on a reference source-tree path at runtime.
- Internal DES_JEH contracts remain internal unless the authoritative product contract design explicitly promotes them.

### Service Documentation

Every service must document:

- purpose;
- owner;
- intended callers;
- lifecycle;
- authority;
- guarantees;
- explicit non-guarantees;
- trust boundary and authentication expectation;
- compatibility behavior.

### RPC Documentation

Every RPC must document:

- purpose;
- caller;
- request semantics;
- response semantics;
- streaming semantics where applicable;
- ordering;
- identity;
- cancellation and end semantics;
- errors and status codes;
- retry and resume behavior;
- side effects;
- invariants;
- non-responsibilities.

An RPC comment must identify whether retry can duplicate work and how callers recognize already observed events. A streaming RPC must state whether continuity is guaranteed, detectable, resumable, or explicitly not guaranteed.

### Message Documentation

Every message must document:

- semantic purpose;
- producer;
- consumer;
- identity semantics where applicable;
- lifecycle where applicable;
- whether it is command, status, evidence, or transport metadata.

Every enum must document its authority, zero value, valid transitions where applicable, and handling of unknown future values.

### Field Documentation

Every field whose meaning is not self-evident must document:

- units;
- allowed range;
- authority or source;
- optionality or presence semantics;
- zero or default meaning;
- temporal meaning;
- whether it is semantic data or transport provenance;
- identity scope where applicable;
- redaction or sensitivity where applicable.

Semantic values and transport provenance must remain distinguishable. Transport adaptation must not silently rewrite semantic data.

### Contract Review Gate

Before a proto file is implemented, review must confirm:

- it is the minimum coherent external product contract;
- no internal-only service was exposed for convenience;
- Viewer requirements are explicit;
- identity, ordering, cancellation, errors, retry, and resume semantics are decided or visibly blocked;
- frozen JEH contracts are adapted without semantic change;
- future-controller observability needs are possible without designing the parked controller;
- all fields and services satisfy this standard.

## Configuration Standard

Configuration must separate operator choices, deployment values, credentials, and internal governed identity.

- Provide a documented template with safe placeholders and no secrets.
- State type, units, allowed values, default, requiredness, precedence, reload behavior, and sensitivity for every setting.
- Validate configuration before starting owned processes.
- Fail with actionable product-language errors.
- Do not require normal operators to compose environment-variable syntax.
- Never encode a runtime dependency on `tramuthus`, Fin_FeedSat_1, or a developer source path.
- Product-generated identities must remain observable in evidence and support bundles.
- Secret values must not appear in logs, status output, command lines where avoidable, or support bundles.

Formats without comments, including JSON, require an adjacent authoritative configuration reference or schema documentation.

## Script and Lifecycle File Standard

Every script must document:

- purpose;
- operator or developer audience;
- parameters and defaults;
- preconditions;
- files, ports, services, and processes affected;
- process ownership rules;
- exit codes;
- failure and rollback behavior;
- idempotency or repeated-run behavior;
- safety constraints;
- examples where useful.

Lifecycle scripts must never kill an unknown process because a desired port is occupied. They must prove ownership before stopping a process, reject stale or mismatched PID records safely, propagate cancellation, and report partial startup and shutdown accurately.

## Build, Package, and Deployment Artifact Standard

Where comments are supported, build and deployment files must state their purpose, inputs, outputs, pinned tools, external downloads, package-relative assumptions, and failure behavior. Where comments are unsupported, an adjacent authoritative build/package reference must provide the same information.

The final package must include:

- product and component versions;
- manifest and integrity information;
- product-owned contracts and generated clients;
- package-relative runtime assets;
- configuration templates;
- lifecycle commands;
- operator guide;
- notices and dependency metadata;
- support-bundle instructions.

The package must not require Go, npm, a source checkout, `go run`, `npm run dev`, Fin_FeedSat_1, or runtime access to `tramuthus` for normal operation.

## Generated Artifacts

Generated code and assets must have:

- one product-owned source of truth;
- a pinned generator and version;
- a deterministic regeneration command;
- a documented output location;
- a clean-tree verification method;
- a rule against manual edits;
- package inclusion rules.

All JEH-HOHO V1 Go and Viewer protobuf outputs must derive from the same `JEH_HOHO.proto` source revision through the single authoritative `buf.yaml`, `buf.gen.yaml`, and deterministic generation operation. Generation provenance must record the Buf and plugin versions. A clean regeneration check must reject drift, manual edits, or service-specific generation paths.

Generated outputs may omit the full source header only when the generator format requires it. The source contract and adjacent generation documentation then carry the authority.

## Operator Documentation Standard

The normal-use portion of the Child's Play guide should fit on approximately one page and use product language:

1. First Use
2. Start
3. View Live
4. View Replay
5. Understand Basic Status
6. Stop

Troubleshooting follows the normal path. The guide must not require knowledge of Go, npm, protobuf, gRPC addresses, Mongo queries, collection IDs, pipeline IDs, environment-variable syntax, source-tree paths, internal stages, or PID management.

Documentation must describe only implemented and validated behavior. Planned commands and states must be labeled proposed until delivered.

## Review and Validation Checklist

An artifact is acceptable only when:

- required metadata and sections are present;
- authority and owner are named;
- inputs, outputs, dependencies, and non-responsibilities are explicit;
- semantic invariants and frozen behavior are protected;
- lifecycle, cancellation, retry, and failure behavior are explicit where applicable;
- security and trust boundaries are stated;
- open human decisions are visible and not silently selected;
- historical evidence and code capability are distinguished;
- filenames and placement match repository policy;
- references resolve or are explicitly external;
- Change Log is current;
- no secret or machine-specific runtime dependency is embedded;
- prose explains intent rather than syntax.

## Validation / Acceptance Criteria

1. Both required design documents satisfy the formal Markdown section inventory.
2. A future proto review can account for every file, service, RPC, message, enum, and non-obvious field using this standard.
3. A future source review can account for every substantive file and exported symbol.
4. A future lifecycle review can determine process ownership and safe stop behavior from documentation alone.
5. A package review can prove that all runtime assets are product-owned and package-relative.
6. An operator can use the normal guide without developer concepts.

## Risks / Open Questions

- Excessive commentary can obscure semantics; reviewers must reject syntax narration.
- Generated formats vary in header support; each generator requires an adjacent documented exception where necessary.
- The exact source-control-compatible change-reference convention for source headers remains subject to implementation planning.

## Human Decisions Required

No human decision blocks adoption of this documentation standard. The source-header change-reference convention may be selected during the first approved implementation slice and recorded here before substantive code is accepted.

## Change Log

| Date | Version | Status | Change | Authority |
|---|---|---|---|---|
| 2026-09-17 | 1.0 | Authoritative | Initial repository-wide artifact and documentation standard. | Chino Rao |
| 2026-09-17 | 1.0 | Human architectural decision incorporated | Recorded the JEH-HOHO V1 single `JEH_HOHO.proto`, single product proto module, single `buf.yaml`, single `buf.gen.yaml`, and deterministic one-operation generation standard. | Chino Rao |