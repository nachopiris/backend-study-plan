import { test } from "node:test";
import assert from "node:assert/strict";
import { runWithConcurrency, type Task } from "../src/run-with-concurrency.ts";

test("runs all tasks and returns results in input order", async () => {
  const durations = [30, 10, 20, 5];
  const tasks: Task<number>[] = durations.map(
    (ms, i) => () => new Promise<number>((resolve) => setTimeout(() => resolve(i), ms)),
  );

  const results = await runWithConcurrency(tasks, 2);
  assert.deepEqual(results, [0, 1, 2, 3]);
});

test("never runs more than `limit` tasks concurrently", async () => {
  const limit = 3;
  let current = 0;
  let maxSeen = 0;

  const tasks: Task<void>[] = Array.from({ length: 12 }, () => async () => {
    current++;
    maxSeen = Math.max(maxSeen, current);
    await new Promise((resolve) => setTimeout(resolve, 10));
    current--;
  });

  await runWithConcurrency(tasks, limit);
  assert.ok(maxSeen <= limit, `observed concurrency ${maxSeen} exceeds limit ${limit}`);
});

test("rejects with the first task's error without hanging", async () => {
  const tasks: Task<number>[] = [
    () => new Promise((resolve) => setTimeout(() => resolve(1), 5)),
    () => Promise.reject(new Error("boom")),
    () => new Promise((resolve) => setTimeout(() => resolve(3), 5)),
  ];

  await assert.rejects(() => runWithConcurrency(tasks, 2), /boom/);
});
