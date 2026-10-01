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
| T1 | Scaffold (root README, shared templates) + Phase 0 practices 01-05 | delegated writer | writer trigger: 2+ non-trivial files | done | c5dc06c |
| T2 | Phase 1 practices 06-10 (data systems) | delegated writer | writer trigger | done | a407231 |
| T3 | Phase 2 practices 11-13 (distributed theory) | delegated writer | writer trigger | done | 02ab02c |
| T4 | Phase 3 practices 14-17 (resilience) | delegated writer | writer trigger | done | 756ad1a |
| T5 | Phase 4 practices 18-21 (messaging) | delegated writer | writer trigger | done | aed681d |
| T6 | Phase 5 practices 22-25 (microservices) | delegated writer | writer trigger | done | 4ab41fc |
| T7 | Phase 6 practices 26-27 (observability) + final index check | delegated writer | writer trigger | done | see final commit |

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
- T2 (delegated writer): RED on stubs, GREEN with deleted scratch solutions for 06-10 (real Postgres; 08 real
  primary/replica via compose with recovery_min_apply_delay=1500ms). Parent spot check: 09 `go vet` clean,
  `go test` 4 FAIL as designed; no leftover practice containers (only the user's own Supabase stacks run).
  Accepted decisions: 07 v2 relies on the sync trigger instead of app-level dual writes so it survives the
  contract step (documented + reflection question); 08 uses docker-compose instead of testcontainers.
  Compose host ports: 06=5434, 07=5435, 10=5436, 08=5440/5441.
  Review: RDD off -> disabled/unmanaged.

- T3 (delegated writer): 11 RED 13/15 fail (2 pass by design), GREEN 15/15 with deleted scratch solution;
  12 RED 5/8 fail (3 "want false" pass by design), GREEN 8/8; worksheet histories checked against a
  scratch-only answer key. 13: no stub code; run-many.sh `bash -n` clean, exercised against a flaky scratch
  test. SKIPPED: shellcheck (not installed). Parent spot check: 11 `go vet` clean + 13 FAIL; run-many `bash -n` ok.
  Review: RDD off -> disabled/unmanaged.

- T4 (delegated writer): 15, 16 (Go + TS/cockatiel ^4), 17 RED on stubs, GREEN with deleted scratch solutions
  under -race / node --test; 14 run for real via compose (naive service 2.016s > 1.5s budget = RED; scratch fix
  1.22s = GREEN), stack torn down; k6 script exercised via grafana/k6 container. Parent spot check: 15 `go vet`
  clean + 11 FAIL lines; no node_modules in repo; no leftover practice containers.
  Convention deviation (accepted): 14's RED is a working-but-naive service failing a latency budget, not a stub.
  Ports 14: 5450, 8474, 15450, 18450. 16 TS tests carry explicit per-test timeouts so RED never hangs.
  Review: RDD off -> disabled/unmanaged.

- T5 (delegated writer): 18 (real apache/kafka:3.9.0 KRaft via compose driven by TestMain, port 19092),
  19 (testcontainers Postgres), 20 (deterministic in-memory bus, GREEN stable at -count=20), 21 (TS) all RED on
  stubs and GREEN with deleted scratch solutions. Parent spot check: 20 `go vet` clean + 5 FAIL; 21 tests 7 / fail 7;
  no leftover practice containers (only the user's pre-existing Supabase and mongo containers).
  Accepted decisions: 18 uses compose because testcontainers' kafka module only accepts confluent-local; 19 fake
  publisher has a documented 5ms delay so the concurrent-relay test reliably catches a missing SKIP LOCKED;
  21 fixture helper lives in support/ because node --test treats every file under test/ as a test.
  Review: RDD off -> disabled/unmanaged.

- T6 (delegated writer): 22, 23 (real `buf generate`, generated code committed), 24 (Pact native lib works on
  linux x64; real consumer pacts + Pact JS Verifier against the Go provider, breaking-change failure observed),
  25 (image built + smoke-tested; manifests validated with kubeconform via Docker) all RED on stubs and GREEN
  with deleted scratch solutions. SKIPPED: running a real kind/k3d cluster (neither installed).
  Parent spot check: 23 `go vet` clean + FAIL lines present; no node_modules/pacts dirs; port 4567 free.
  Accepted decisions: 23 router keeps its own port types (one module per template); 24 Pact tests are given
  like every other practice's tests, with a "write a second interaction" stretch; 24 verify script builds the Go
  binary instead of `go run` to avoid orphaned providers.
  Review: RDD off -> disabled/unmanaged.

- T7 (delegated writer): 26 Go (18 tests) and TS (7 tests), 27 Go (17 cases with SRE workbook numbers) RED on
  stubs, GREEN with deleted scratch solutions; `promtool check rules` SUCCESS (40 rules); otel-lgtm stack started
  for real and a span confirmed in Tempo, then torn down. Root README: stale "created later" note removed,
  4 type labels corrected (10, 14, 18, 19), tools-per-practice table added.
  Parent final audit: 27 practices, 0 missing README/NOTES/template; 27 index links, 0 broken; 27 `go vet` clean
  + FAIL lines present; no leftover practice containers.
  Review: RDD off -> disabled/unmanaged.

## Next step

Feature complete. Push/PR remain the user's decision (no remote configured). Engram mirror still to resync.
