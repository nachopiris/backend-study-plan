// A concurrency-limited task runner for asynchronous work, built by hand (no
// p-limit, no third-party queue library).

export type Task<T> = () => Promise<T>;

/**
 * Run every task in tasks, allowing at most `limit` of them to be in flight at
 * the same time, and return their results in the SAME ORDER as the input
 * tasks array (not the order in which they settle).
 *
 * Contract:
 *   - At most `limit` tasks are executing (i.e. their returned promise has not
 *     yet settled) at any given moment.
 *   - result[i] corresponds to tasks[i], regardless of which task finishes
 *     first.
 *   - If any task's promise rejects, runWithConcurrency must reject with that
 *     error. Tasks that were already started keep running to completion (you
 *     cannot abort an in-flight promise in JavaScript), but no NEW task should
 *     be started after a rejection is observed.
 *   - limit must be treated as at least 1.
 */
export async function runWithConcurrency<T>(tasks: Task<T>[], limit: number): Promise<T[]> {
  throw new Error("not implemented");
}
