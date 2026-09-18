# JEH-HOHO Child's Play User Guide

| Metadata | Value |
|---|---|
| Date | 2026-09-17 |
| Version | 1.0 |
| Status | Authoritative candidate guide; human end-user acceptance pending |
| Product | JEH-HOHO |
| Component | Packaged product |
| Document Type | End-user guide |
| Authority | Chino Rao, Human Authority |

## Purpose

Start, use, and stop JEH-HOHO without developer tools. This guide is for a non-technical intern testing the product candidate.

## First Use

1. Double-click **CONFIGURE JEH-HOHO**.
2. Replace the two placeholder values with the supplied Alpaca key and secret.
3. Save and close Notepad. Do not share that file.

MongoDB and its required JEH-HOHO collections must already be available on the test computer. Setup is performed by the product administrator, not the operator.

## Start

1. Double-click **START JEH-HOHO**.
2. Leave the status window open.
3. Wait for `JEH-HOHO READY`. The Viewer normally opens automatically. If it does not, open the address printed after `Viewer:`.

Startup states mean:

| State | Meaning |
|---|---|
| `STARTING` | JEH-HOHO is beginning startup. |
| `PREFLIGHT` | Required settings, connections, and product files are being checked. |
| `CONNECTING` | JEH-HOHO is connecting to Alpaca. |
| `READY` | The product is operating. A market Bar is not required to become ready. |
| `DEGRADED` | The product is running but a connection is recovering; continuity may be unknown. |
| `FAILED` | Startup or a required component failed. Read the plain message in the window. |

Double-click **STATUS JEH-HOHO** at any time to see the current state and Viewer address.

## Use The Viewer

- **LIVE** follows Capital events committed by the currently running JEH-HOHO product. It does not select an old stored run.
- **REPLAY** lets you select and play a completed stored run.
- To return to current activity, select **LIVE**. The Viewer reconnects automatically.
- A reconnecting or unavailable message means the current connection is not healthy. Wait briefly. If it does not recover, stop and start JEH-HOHO once.

Market inactivity is not itself a failure. The Viewer updates when the running product commits a new Capital event.

## Stop

1. Double-click **STOP JEH-HOHO**.
2. Wait for `JEH-HOHO STOPPED` in the original status window.
3. The Viewer and product services close automatically.

Do not end unknown processes in Task Manager. STOP addresses only the JEH-HOHO instance that owns the current package run.

For an ordinary problem, stop once, wait for `STOPPED`, and start again. If the problem repeats, double-click **CREATE SUPPORT REPORT** and give the reported file to product support. The report excludes credentials.

## Scope

This guide covers normal candidate testing: configure, start, observe LIVE, select REPLAY, return to LIVE, understand basic status, create a support report, and stop.

## Out Of Scope

Product administration, database setup, software development, and product freeze approval.

## Dependencies / References

The packaged JEH-HOHO candidate and administrator-provided Alpaca credentials and MongoDB service.

## Assumptions

The administrator prepared MongoDB and supplied valid Alpaca credentials.

## Invariants / Frozen Decisions

LIVE follows the current run; REPLAY uses a deliberately selected stored run. READY does not require a first or recent Bar.

## Validation / Acceptance Criteria

A non-technical tester completes START, READY observation, LIVE, REPLAY, return to LIVE, STATUS, support report, and STOP without developer commands.

Record the acceptance result before product freeze:

| Check | Result |
|---|---|
| First-use configuration completed without developer tools | PENDING |
| START reached READY and LIVE displayed current-run Capital evidence | PENDING |
| Connection fault and recovery/continuity status were understandable | PENDING |
| A deliberate REPLAY was selected and return to LIVE succeeded | PENDING |
| STATUS and sanitized support report succeeded | PENDING |
| STOP reached STOPPED with no orphan product process | PENDING |

## Risks / Open Questions

Human end-user testing may identify wording or workflow improvements before product freeze.

## Human Decisions Required

Human end-user acceptance and any later product baseline freeze remain pending.

## Change Log

| Date | Version | Status | Change | Authority |
|---|---|---|---|---|
| 2026-09-17 | 1.0 | Candidate guide | Initial Child's Play workflow for Slices 6 and 7. | Chino Rao |