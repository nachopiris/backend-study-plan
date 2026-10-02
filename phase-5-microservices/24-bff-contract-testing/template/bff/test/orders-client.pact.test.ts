// Consumer-driven contract test (Pact, PactV4 API): this test defines what THIS consumer (the
// BFF) expects from the orders-api provider, and in doing so GENERATES a pact file under
// ../pacts/bff-orders-api.json. That file is what scripts/verify-provider.ts later replays
// against the real Go provider in ../../provider-go — the same contract, checked from both
// sides, in two different languages.
//
// GIVEN test: you do not need to modify this file. Implementing OrdersClient (src/orders-client.ts)
// correctly is what turns these from RED to GREEN.
import { test } from "node:test";
import assert from "node:assert/strict";
import { PactV4, Matchers } from "@pact-foundation/pact";
import { OrderNotFoundError, OrdersClient } from "../src/orders-client.ts";

const pact = new PactV4({ consumer: "bff", provider: "orders-api" });

test("getOrder: returns the order when it exists", async () => {
  await pact
    .addInteraction()
    .uponReceiving("a request for an existing order")
    .withRequest("GET", "/orders/order-1")
    .willRespondWith(200, (builder) => {
      builder.headers({ "Content-Type": "application/json" }).jsonBody({
        id: Matchers.like("order-1"),
        customerId: Matchers.like("cust-1"),
        status: Matchers.like("confirmed"),
        items: Matchers.eachLike({
          sku: Matchers.like("sku-1"),
          quantity: Matchers.integer(2),
          unitPriceCents: Matchers.integer(500),
        }),
        totalCents: Matchers.integer(1000),
      });
    })
    .executeTest(async (server) => {
      const client = new OrdersClient({ baseUrl: server.url, timeoutMs: 2000 });
      const order = await client.getOrder("order-1");

      assert.equal(order.id, "order-1");
      assert.equal(order.customerId, "cust-1");
      assert.equal(order.status, "confirmed");
      assert.equal(order.totalCents, 1000);
      assert.equal(order.items.length, 1);
      assert.equal(order.items[0]?.sku, "sku-1");
    });
});

test("getOrder: throws OrderNotFoundError for a missing order", async () => {
  await pact
    .addInteraction()
    .uponReceiving("a request for a missing order")
    .withRequest("GET", "/orders/does-not-exist")
    .willRespondWith(404)
    .executeTest(async (server) => {
      const client = new OrdersClient({ baseUrl: server.url, timeoutMs: 2000 });

      await assert.rejects(
        () => client.getOrder("does-not-exist"),
        (err: unknown) => {
          assert.ok(err instanceof OrderNotFoundError, `expected OrderNotFoundError, got ${String(err)}`);
          assert.equal(err.orderId, "does-not-exist");
          return true;
        },
      );
    });
});
