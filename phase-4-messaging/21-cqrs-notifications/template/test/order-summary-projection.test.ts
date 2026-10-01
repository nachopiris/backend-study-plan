import { test } from "node:test";
import assert from "node:assert/strict";
import { OrderSummaryProjection } from "../src/order-summary-projection.ts";
import { loadCanonicalEvents, reverseEventsForOrder } from "../support/fixtures.ts";

test("applies events in order and builds correct summaries", () => {
  const events = loadCanonicalEvents();
  const projection = new OrderSummaryProjection();
  for (const event of events) projection.apply(event);

  assert.deepEqual(projection.get("order-1"), {
    orderId: "order-1",
    status: "confirmed",
    customerId: "customer-a",
    totalCents: 3000,
    itemCount: 2,
    createdAt: "2026-01-10T10:00:00.000Z",
    updatedAt: "2026-01-10T10:02:00.000Z",
  });

  assert.deepEqual(projection.get("order-2"), {
    orderId: "order-2",
    status: "cancelled",
    customerId: "customer-b",
    totalCents: 0,
    itemCount: 0,
    createdAt: "2026-01-10T11:00:00.000Z",
    updatedAt: "2026-01-10T11:05:00.000Z",
  });

  assert.deepEqual(projection.get("order-3"), {
    orderId: "order-3",
    status: "confirmed",
    customerId: "customer-c",
    totalCents: 4000,
    itemCount: 5,
    createdAt: "2026-01-10T12:00:00.000Z",
    updatedAt: "2026-01-10T12:03:00.000Z",
  });
});

test("ignores a duplicate event (same id) with no additional effect", () => {
  const events = loadCanonicalEvents();
  const projection = new OrderSummaryProjection();
  for (const event of events) projection.apply(event);

  const before = projection.get("order-3");
  // Re-apply every order-3 event a second time: a buggy implementation that
  // does not dedupe by id would double the totals.
  for (const event of events.filter((e) => e.orderId === "order-3")) {
    projection.apply(event);
  }
  assert.deepEqual(projection.get("order-3"), before);
});

test("holds an out-of-order event until its predecessor arrives, then converges", () => {
  const events = loadCanonicalEvents();
  const reordered = reverseEventsForOrder(events, "order-1"); // order-1's events now arrive 3, 2, 1

  const projection = new OrderSummaryProjection();
  for (const event of reordered) projection.apply(event);

  // Final state must be identical to applying everything in true order.
  const reference = new OrderSummaryProjection();
  for (const event of events) reference.apply(event);

  assert.deepEqual(projection.get("order-1"), reference.get("order-1"));
  assert.deepEqual(projection.get("order-2"), reference.get("order-2"));
  assert.deepEqual(projection.get("order-3"), reference.get("order-3"));
});

test("rebuild() replays the whole stream and matches incremental application", () => {
  const events = loadCanonicalEvents();

  const incremental = new OrderSummaryProjection();
  // Feed events out of order AND with a duplicate, exercising both
  // tolerances at once.
  const scrambled = reverseEventsForOrder(events, "order-3");
  for (const event of scrambled) incremental.apply(event);
  for (const event of scrambled) incremental.apply(event); // duplicate pass

  const rebuilt = OrderSummaryProjection.rebuild(events); // canonical order

  for (const orderId of ["order-1", "order-2", "order-3"]) {
    assert.deepEqual(
      rebuilt.get(orderId),
      incremental.get(orderId),
      `rebuild() and incremental application disagree for ${orderId}`,
    );
  }
});
