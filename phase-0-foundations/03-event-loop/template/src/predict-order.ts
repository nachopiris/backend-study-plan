// This module runs a fixed snippet that mixes process.nextTick, a Promise
// microtask, setTimeout(fn, 0), and setImmediate, all scheduled from inside an
// I/O callback (fs.readFile on this very file) so the relative order of
// setImmediate vs setTimeout is deterministic across runs — the classic
// "immediate always wins over timeout when scheduled from an I/O callback"
// rule from the Node docs.
//
// runSnippet() is already implemented: it executes the snippet and resolves
// with the REAL order in which each labeled step ran, as an array of labels.
//
// Your job is predictOrder(): before running anything, read the snippet below
// and write down, in code, what you believe the real order will be. The test
// runs runSnippet() and compares it against your prediction with a strict
// deep-equal — so this only passes once your mental model of the event loop
// matches reality, not by trial and error against the test output.
//
// The six labels that can appear, in the order they are scheduled in the
// snippet (NOT necessarily the order they run):
//   "io-callback-start" — first synchronous line inside the I/O callback
//   "io-callback-end"   — last synchronous line inside the I/O callback
//   "nextTick"          — process.nextTick callback
//   "promise"           — Promise.resolve().then callback
//   "immediate"         — setImmediate callback
//   "timeout"           — setTimeout(fn, 0) callback

import { readFile } from "node:fs";
import { fileURLToPath } from "node:url";

export function runSnippet(): Promise<string[]> {
  const order: string[] = [];
  const selfPath = fileURLToPath(import.meta.url);

  return new Promise((resolve, reject) => {
    readFile(selfPath, (err) => {
      if (err) {
        reject(err);
        return;
      }

      order.push("io-callback-start");

      setTimeout(() => {
        order.push("timeout");
        resolve(order);
      }, 0);

      setImmediate(() => {
        order.push("immediate");
      });

      Promise.resolve().then(() => {
        order.push("promise");
      });

      process.nextTick(() => {
        order.push("nextTick");
      });

      order.push("io-callback-end");
    });
  });
}

/**
 * Return your predicted order of the six labels above, as they would be
 * logged by runSnippet(). Do this BEFORE running the test — reason about it
 * from what you know of the event loop phases and microtask draining, not by
 * running runSnippet() first and copying its output.
 */
export function predictOrder(): string[] {
  throw new Error("not implemented");
}
