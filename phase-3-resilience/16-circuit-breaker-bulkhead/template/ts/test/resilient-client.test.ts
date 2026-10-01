import { test } from "node:test";
import assert from "node:assert/strict";
import { isBrokenCircuitError, isBulkheadRejectedError, isTaskCancelledError } from "cockatiel";
import { createResilientClient, type FetchLike, type ResilientClientOptions } from "../src/resilient-client.ts";

function baseOptions(overrides: Partial<ResilientClientOptions> = {}): ResilientClientOptions {
  return {
    timeoutMs: 100,
    retryAttempts: 1,
    retryBackoffMs: 5,
    breakerThreshold: 3,
    breakerHalfOpenAfterMs: 1000,
    bulkheadLimit: 10,
    bulkheadQueue: 10,
    ...overrides,
  };
}

function okResponse(): Response {
  return new Response(JSON.stringify({ ok: true }), { status: 200 });
}

test("resolves normally when the underlying fetch succeeds", async () => {
  let calls = 0;
  const fetchLike: FetchLike = async () => {
    calls++;
    return okResponse();
  };

  const client = createResilientClient(fetchLike, baseOptions());
  const res = await client("https://example.test/resource");

  assert.equal(res.status, 200);
  assert.equal(calls, 1);
});

// timeout option: a hard safety net while the client is still an
// unimplemented stub (which never times out fetchLike on its own) — once
// implemented, this test settles in well under timeoutMs anyway.
test("rejects with TaskCancelledError when the call exceeds the timeout", { timeout: 2000 }, async () => {
  const fetchLike: FetchLike = () => new Promise<Response>(() => {}); // never resolves
  const client = createResilientClient(fetchLike, baseOptions({ timeoutMs: 30, retryAttempts: 1 }));

  const start = performance.now();
  await assert.rejects(() => client("https://example.test/slow"), (err: unknown) => {
    assert.ok(isTaskCancelledError(err), `expected a TaskCancelledError, got ${String(err)}`);
    return true;
  });
  const elapsed = performance.now() - start;
  assert.ok(elapsed < 1000, `expected the call to fail fast (<1000ms), took ${elapsed}ms`);
});

test("retries a transiently failing call and eventually succeeds", async () => {
  let calls = 0;
  const fetchLike: FetchLike = async () => {
    calls++;
    if (calls < 3) {
      throw new Error("transient failure");
    }
    return okResponse();
  };

  const client = createResilientClient(fetchLike, baseOptions({ retryAttempts: 3, retryBackoffMs: 5 }));
  const res = await client("https://example.test/flaky");

  assert.equal(res.status, 200);
  assert.equal(calls, 3, "fetchLike should have been called once per attempt");
});

test("opens the circuit breaker after enough consecutive failures and stops calling fetchLike", async () => {
  let calls = 0;
  const fetchLike: FetchLike = async () => {
    calls++;
    throw new Error("downstream always fails");
  };

  // retryAttempts: 1 isolates the breaker's own behavior: each client call
  // maps to exactly one fetchLike call when the breaker is still closed.
  const threshold = 3;
  const client = createResilientClient(
    fetchLike,
    baseOptions({ retryAttempts: 1, breakerThreshold: threshold, breakerHalfOpenAfterMs: 60_000 }),
  );

  for (let i = 0; i < threshold; i++) {
    await assert.rejects(() => client("https://example.test/broken"));
  }
  assert.equal(calls, threshold, "fetchLike should run once per call while the breaker is still closed");

  await assert.rejects(() => client("https://example.test/broken"), (err: unknown) => {
    assert.ok(isBrokenCircuitError(err), `expected a BrokenCircuitError, got ${String(err)}`);
    return true;
  });
  assert.equal(calls, threshold, "fetchLike must NOT run again once the breaker is open");
});

// timeout option: while unimplemented, the stub forwards both calls
// straight to fetchLike, so the second call also waits on `gate` instead of
// rejecting — this bound keeps that RED state from hanging the suite. The
// gate itself is released on its own short timer so the first call (and
// this test) can never block forever either way.
test("rejects beyond bulkhead capacity without calling fetchLike for the rejected call", { timeout: 2000 }, async () => {
  let calls = 0;
  const gate = new Promise<void>((resolve) => {
    setTimeout(resolve, 200);
  });
  const fetchLike: FetchLike = async () => {
    calls++;
    await gate;
    return okResponse();
  };

  // retryAttempts: 1 isolates the bulkhead's own behavior: a rejected call
  // never gets a second chance to find room.
  const client = createResilientClient(
    fetchLike,
    baseOptions({ retryAttempts: 1, bulkheadLimit: 1, bulkheadQueue: 0, timeoutMs: 5000 }),
  );

  const first = client("https://example.test/busy"); // takes the only slot

  // Give the first call time to actually reach fetchLike before firing the
  // second one, so the rejection is unambiguous.
  await new Promise((resolve) => setTimeout(resolve, 20));

  await assert.rejects(() => client("https://example.test/busy"), (err: unknown) => {
    assert.ok(isBulkheadRejectedError(err), `expected a BulkheadRejectedError, got ${String(err)}`);
    return true;
  });
  assert.equal(calls, 1, "the rejected call must never reach fetchLike");

  await first;
});
