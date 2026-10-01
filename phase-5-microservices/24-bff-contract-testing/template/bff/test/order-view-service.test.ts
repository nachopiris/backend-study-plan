// GIVEN test: you do not need to modify this file. Implementing OrderViewService
// (src/order-view-service.ts) correctly is what turns these from RED to GREEN. These tests use
// fakes (support/fakes.ts), not real HTTP or Pact — the Pact contract in this practice covers
// OrdersClient only (see orders-client.pact.test.ts); the payments side has no given provider to
// verify against, so it is exercised here through plain aggregation/degradation tests instead.
import { test } from "node:test";
import assert from "node:assert/strict";
import { OrderViewService } from "../src/order-view-service.ts";
import type { Order, PaymentStatus } from "../src/types.ts";
import {
  delayed,
  delayedFailure,
  FakeOrdersClient,
  FakePaymentsClient,
  immediate,
  immediateFailure,
  sampleOrder,
} from "../support/fakes.ts";

test("getOrderView: aggregates the order and payment status on the happy path", async () => {
  const order = sampleOrder();
  const orders = new FakeOrdersClient(immediate(order));
  const payments = new FakePaymentsClient(immediate<PaymentStatus>("paid"));
  const service = new OrderViewService(orders, payments);

  const view = await service.getOrderView("order-1");

  assert.deepEqual(view, {
    orderId: order.id,
    customerId: order.customerId,
    status: order.status,
    items: order.items,
    totalCents: order.totalCents,
    paymentStatus: "paid",
  });
});

test("getOrderView: degrades to paymentStatus 'unknown' when payments fails immediately", async () => {
  const orders = new FakeOrdersClient(immediate(sampleOrder()));
  const payments = new FakePaymentsClient(immediateFailure<PaymentStatus>(new Error("payments: connection refused")));
  const service = new OrderViewService(orders, payments);

  const view = await service.getOrderView("order-1");

  assert.equal(view.paymentStatus, "unknown");
  assert.equal(view.orderId, "order-1");
});

// Bounded wait: both fakes resolve within 100ms; a hanging implementation fails this test by
// timing out at the test runner's default timeout, not by looping forever.
test("getOrderView: degrades to 'unknown' when payments is slow (down), without failing the whole view", async () => {
  const orders = new FakeOrdersClient(immediate(sampleOrder()));
  const payments = new FakePaymentsClient(delayedFailure<PaymentStatus>(new Error("payments: timed out"), 50));
  const service = new OrderViewService(orders, payments);

  const view = await service.getOrderView("order-1");

  assert.equal(view.paymentStatus, "unknown");
});

test("getOrderView: still returns the real payment status when payments is slow but eventually succeeds", async () => {
  const orders = new FakeOrdersClient(immediate(sampleOrder()));
  const payments = new FakePaymentsClient(delayed<PaymentStatus>("pending", 50));
  const service = new OrderViewService(orders, payments);

  const view = await service.getOrderView("order-1");

  assert.equal(view.paymentStatus, "pending");
});

test("getOrderView: rejects when the order itself cannot be fetched", async () => {
  const orderError = new Error("orders: not found");
  const orders = new FakeOrdersClient(immediateFailure<Order>(orderError));
  const payments = new FakePaymentsClient(immediate<PaymentStatus>("paid"));
  const service = new OrderViewService(orders, payments);

  await assert.rejects(() => service.getOrderView("missing-order"), orderError);
});

// This test fails a correct implementation ONLY if it calls paymentsClient after awaiting
// ordersClient to finish (sequential), instead of starting both at once (concurrent): with a
// 60ms order delay and a 60ms payments delay, concurrent execution finishes in ~60ms;
// sequential execution would take ~120ms.
test("getOrderView: fetches the order and the payment status concurrently, not sequentially", async () => {
  const orders = new FakeOrdersClient(delayed(sampleOrder(), 60));
  const payments = new FakePaymentsClient(delayed<PaymentStatus>("paid", 60));
  const service = new OrderViewService(orders, payments);

  const start = performance.now();
  await service.getOrderView("order-1");
  const elapsed = performance.now() - start;

  assert.ok(elapsed < 100, `expected concurrent fetch to take ~60ms, took ${elapsed}ms (looks sequential)`);
});
