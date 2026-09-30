# Feature: practice-guides

## Objective

Create one guide document and one starter template per practice of the 26-week backend study plan
(resilience, distributed systems, microservices), so the learner knows what to build, how to approach it,
and how to verify it without being handed the solution.

## Problem / Why

The study plan (Claude Doc: https://claude.ai/code/artifact/948b4a36-89aa-4b10-84ed-e11d5a058885) lists weekly
practices in one line each. The learner asked for more detail: a folder, a document per practice, and a template
that orients how to solve it.

## Scope

- Root `README.md`: how to use the repo, index of all practices, link to the plan.
- `shared/`: reusable templates (practice notes, ADR, practice guide structure).
- One directory per practice under `phase-N-<name>/NN-<slug>/` with:
  - `README.md` guide (Spanish, neutral register): objective, prerequisites, context, numbered steps,
    acceptance criteria checklist, hints in `<details>`, reflection questions, going further, resources.
  - `template/`: runnable starter code (Go or TypeScript) with stubs returning "not implemented" and tests
    that fail until the learner implements them; or a worksheet when the practice is not code.
  - `NOTES.md` copied from the shared notes template.

Out of scope: reference solutions committed to the repo, remote push, PRs.

## Constraints

- Code, identifiers, comments and directory names in English; guides in neutral Spanish (they extend the
  Spanish study plan).
- Go 1.27.1, Node 26.8.1 (native TS type stripping, `node --test`).
- Templates must compile/run; their tests must fail only because of stubs, and must pass with a correct
  implementation (verified with a throwaway reference solution in the scratchpad, never committed).
- Implement patterns by hand first; libraries only where the plan says so.

## Configuration

- TDD: enabled (source: session config "Strict TDD Mode: enabled"). Runners: `go test ./...` for Go templates,
  `node --test` for TS templates. RED = stubbed template; GREEN = throwaway reference solution; the committed
  state is intentionally RED for the learner.
- RDD: off (decided by global) -> no review; delivery `disabled/unmanaged`.
- Delivery strategy: `exception-ok` — local study repo with no remote; generated study material; forecast
  ~6000 authored lines. No PRs planned; push/PR remain the user's decision.

## Tasks

| ID | Task | Route | Trigger evidence | Status | Commit |
|----|------|-------|------------------|--------|--------|
| T1 | Scaffold (root README, shared templates) + Phase 0 practices 01-05 | delegated writer | writer trigger: 2+ non-trivial files | done | see T1 evidence |
| T2 | Phase 1 practices 06-10 (data systems) | delegated writer | writer trigger | pending | |
| T3 | Phase 2 practices 11-13 (distributed theory) | delegated writer | writer trigger | pending | |
| T4 | Phase 3 practices 14-17 (resilience) | delegated writer | writer trigger | pending | |
| T5 | Phase 4 practices 18-21 (messaging) | delegated writer | writer trigger | pending | |
| T6 | Phase 5 practices 22-25 (microservices) | delegated writer | writer trigger | pending | |
| T7 | Phase 6 practices 26-27 (observability) + final index check | delegated writer | writer trigger | pending | |

### Practice list

- Phase 0: 01 request-anatomy (worksheet), 02 worker-pool-context (Go), 03 event-loop (TS),
  04 isolation-anomalies (Go + Postgres), 05 modular-monolith (Go)
- Phase 1: 06 data-modeling-indexes, 07 zero-downtime-migrations, 08 replication-lag,
  09 partitioning-design (worksheet), 10 concurrent-checkout (Go)
- Phase 2: 11 logical-clocks (Go), 12 consistency-models (worksheet), 13 raft (MIT 6.5840 guide + notes)
- Phase 3: 14 chaos-toxiproxy, 15 retries-idempotency (Go), 16 circuit-breaker-bulkhead (Go + TS),
  17 load-protection (Go + k6)
- Phase 4: 18 kafka-basics (Go), 19 outbox-idempotent-consumer (Go), 20 choreographed-saga (Go),
  21 cqrs-notifications (TS)
- Phase 5: 22 service-boundaries (ADR worksheet), 23 strangler-grpc (Go), 24 bff-contract-testing (TS + Pact),
  25 local-kubernetes
- Phase 6: 26 telemetry-otel (Go + TS), 27 slos-game-day (worksheet)

## Acceptance criteria

- Every practice has `README.md`, `NOTES.md`, and `template/` (code or worksheet).
- Every Go template: `go vet ./...` passes; `go test ./...` fails only with "not implemented" stubs;
  passes with the throwaway reference solution.
- Every TS template: `node --test` runs without syntax/type-stripping errors; fails only on stubs; passes with
  the throwaway reference solution.
- Root README indexes all 27 practices with working relative links.

## Progress / Evidence

- Engram mirror `odd/practice-guides/tasks`: PENDING — `mem_save` returned `ambiguous_project` (offered only
  unrelated projects `ai-job-search`, `livecap`). Resync once the project resolves.
- T1 (delegated writer): writer reported RED on stubs and GREEN with throwaway scratch solutions (deleted) for
  01 client, 02, 03, 04 (real postgres:17 via testcontainers; Docker available), 05 (incl. architecture test).
  Parent spot check: 02 `go vet` clean + `go test` 5x "not implemented"; 03 `node --test` tests 6 / fail 6, no hang.
  Notes: 04 docker-compose exposes Postgres on host port 5433 to avoid clashing with a local 5432.
  Review: RDD off -> disabled/unmanaged.

## Next step

T2.
