import { test } from "node:test";
import assert from "node:assert/strict";
import { yieldingSum } from "../src/yielding-sum.ts";

test("computes the correct sum", async () => {
  const values = Array.from({ length: 10_000 }, (_, i) => i + 1);
  const expected = values.reduce((a, b) => a + b, 0);

  const result = await yieldingSum(values, 500);
  assert.equal(result, expected);
});

test("yields to the event loop between chunks instead of blocking it", async () => {
  // This test does not rely on wall-clock timing (no flaky sleeps). Instead,
  // it schedules a "probe" via setImmediate before calling yieldingSum, and
  // has the probe reschedule itself. Because setImmediate callbacks run in
  // the order they were scheduled (FIFO, within Node's "check" phase), the
  // probe can only tick more than once while yieldingSum is still in
  // progress if yieldingSum itself yields back to the event loop via
  // setImmediate between chunks — a fully synchronous implementation would
  // resolve its promise through the microtask queue before the event loop
  // ever reaches the check phase, so the probe would tick zero times.
  const values = Array.from({ length: 200_000 }, () => 1);
  const expected = values.length;

  let ticks = 0;
  let stop = false;
  function probe() {
    ticks++;
    if (!stop) setImmediate(probe);
  }
  setImmediate(probe);

  try {
    const result = await yieldingSum(values, 500);
    assert.equal(result, expected);
    assert.ok(
      ticks > 0,
      "expected at least one event-loop tick (setImmediate) to interleave with yieldingSum, meaning it yields between chunks instead of running as one blocking synchronous pass",
    );
  } finally {
    // Always stop the self-rescheduling probe, even if yieldingSum throws,
    // so a failing/unimplemented version doesn't leave a runaway setImmediate
    // loop and hang the test process.
    stop = true;
  }
});
