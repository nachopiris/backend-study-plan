// A sum over a (potentially large) array of numbers that voluntarily yields
// control back to the event loop while it works, instead of blocking it with
// one long synchronous loop.

/**
 * Sum every number in values, returning the total.
 *
 * Contract:
 *   - The result must equal the exact sum of all elements in values.
 *   - The implementation must NOT sum the entire array in a single
 *     synchronous pass. Instead, process it in chunks and yield to the event
 *     loop (via setImmediate) between chunks, so that other pending work
 *     (timers, I/O callbacks) gets a chance to run while the sum is still in
 *     progress on a large input.
 *   - chunkSize controls how many elements are summed per synchronous chunk
 *     before yielding. It must be treated as at least 1.
 */
export async function yieldingSum(values: number[], chunkSize: number = 1000): Promise<number> {
  throw new Error("not implemented");
}
