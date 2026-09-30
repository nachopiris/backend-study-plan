import { test } from "node:test";
import assert from "node:assert/strict";
import { predictOrder, runSnippet } from "../src/predict-order.ts";

test("predicted order matches the real event loop order", async () => {
  const predicted = predictOrder();
  const actual = await runSnippet();
  assert.deepEqual(predicted, actual);
});
