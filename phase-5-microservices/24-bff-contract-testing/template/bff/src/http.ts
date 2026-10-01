// GIVEN helper: wraps fetch with a hard timeout, using AbortController. You already built this
// exact mechanism by hand in earlier practices (Fase 0's worker pools, Fase 3's resilient
// client) — it is given here so this practice can focus on BFF aggregation and contract testing
// instead of re-deriving timeout handling.

export type FetchLike = typeof fetch;

export class TimeoutError extends Error {
  constructor(url: string, timeoutMs: number) {
    super(`request to ${url} timed out after ${timeoutMs}ms`);
    this.name = "TimeoutError";
  }
}

/**
 * fetchWithTimeout calls fetchImpl(url, init) and aborts it if it has not settled within
 * timeoutMs, throwing TimeoutError in that case. Any other rejection from fetchImpl (a network
 * error, for instance) propagates unchanged.
 */
export async function fetchWithTimeout(
  fetchImpl: FetchLike,
  url: string,
  timeoutMs: number,
  init: RequestInit = {},
): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetchImpl(url, { ...init, signal: controller.signal });
  } catch (err) {
    if (controller.signal.aborted) {
      throw new TimeoutError(url, timeoutMs);
    }
    throw err;
  } finally {
    clearTimeout(timer);
  }
}
