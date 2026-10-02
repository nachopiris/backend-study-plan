// A resilient HTTP client built by COMPOSING cockatiel policies, instead of
// hand-rolling timeout/retry/breaker/bulkhead the way you just did in Go
// (template/go). The point of this half of the practice is to see how the
// same four ideas look as off-the-shelf, composable building blocks.
//
// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase.

import {
  bulkhead,
  circuitBreaker,
  ConsecutiveBreaker,
  ConstantBackoff,
  handleAll,
  retry,
  timeout,
  TimeoutStrategy,
  wrap,
} from "cockatiel";

/** The minimal shape of fetch that this client wraps and returns. */
export type FetchLike = (url: string) => Promise<Response>;

export interface ResilientClientOptions {
  /** Hard timeout per underlying call, in milliseconds. */
  timeoutMs: number;
  /** Total attempts (including the first), for transient failures. */
  retryAttempts: number;
  /** Fixed delay between retry attempts, in milliseconds. */
  retryBackoffMs: number;
  /** Consecutive failures that open the circuit breaker. */
  breakerThreshold: number;
  /** How long the breaker stays open before allowing a half-open probe. */
  breakerHalfOpenAfterMs: number;
  /** Max concurrent calls allowed through at once. */
  bulkheadLimit: number;
  /** Max calls allowed to queue once bulkheadLimit is reached. */
  bulkheadQueue: number;
}

/**
 * Builds a FetchLike wrapping fetchLike with, in some composed order you
 * choose, a timeout, a retry with backoff, a circuit breaker, and a
 * bulkhead — all from cockatiel, combined with `wrap(...)`.
 *
 * Contract (enforced by the tests, not by this signature):
 *   - A call that exceeds `timeoutMs` rejects with cockatiel's
 *     TaskCancelledError (check with `isTaskCancelledError` from
 *     'cockatiel').
 *   - A call that fails but succeeds within `retryAttempts` resolves
 *     normally; fetchLike is called once per attempt.
 *   - After `breakerThreshold` consecutive failures, further calls reject
 *     immediately with a BrokenCircuitError (check with
 *     `isBrokenCircuitError`), WITHOUT calling fetchLike again, until
 *     `breakerHalfOpenAfterMs` has elapsed.
 *   - Once more than `bulkheadLimit + bulkheadQueue` calls are in flight at
 *     the same time, the excess calls reject immediately with a
 *     BulkheadRejectedError (check with `isBulkheadRejectedError`), without
 *     ever calling fetchLike.
 *
 * Not implemented yet: returns fetchLike completely unwrapped, with none of
 * the above behavior.
 */
export function createResilientClient(fetchLike: FetchLike, options: ResilientClientOptions): FetchLike {
  return fetchLike;
}
