// GIVEN helper: loads the canonical event stream fixture used by every test
// in this practice. Not something you need to implement.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import type { OrderEvent } from "../src/types.ts";

const dirname = path.dirname(fileURLToPath(import.meta.url));

export function loadCanonicalEvents(): OrderEvent[] {
  const raw = readFileSync(path.join(dirname, "..", "fixtures", "order-events.json"), "utf8");
  return JSON.parse(raw) as OrderEvent[];
}

/** Returns a NEW array with events reordered: for the given orderId, its
 * events are reversed in place (keeping every other order's events where
 * they were), simulating an out-of-order delivery for just that order. */
export function reverseEventsForOrder(events: OrderEvent[], orderId: string): OrderEvent[] {
  const result = [...events];
  const indices: number[] = [];
  result.forEach((e, i) => {
    if (e.orderId === orderId) indices.push(i);
  });
  const reversed = indices.map((i) => result[i]).reverse();
  indices.forEach((originalIndex, k) => {
    result[originalIndex] = reversed[k];
  });
  return result;
}
