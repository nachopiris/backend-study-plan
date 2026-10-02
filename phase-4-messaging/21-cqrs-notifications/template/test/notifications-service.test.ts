import { test } from "node:test";
import assert from "node:assert/strict";
import { NotificationsService } from "../src/notifications-service.ts";
import { FakeEmailSender } from "../src/email-sender.ts";
import { loadCanonicalEvents } from "../support/fixtures.ts";

test("sends exactly one email per confirmed order, none for cancelled ones", () => {
  const sender = new FakeEmailSender();
  const notifications = new NotificationsService(sender);

  for (const event of loadCanonicalEvents()) notifications.handle(event);

  // order-1 and order-3 are confirmed; order-2 is cancelled.
  assert.equal(sender.sent.length, 2);
});

test("does not resend when the exact same confirmation event is delivered twice", () => {
  const sender = new FakeEmailSender();
  const notifications = new NotificationsService(sender);
  const confirmed = loadCanonicalEvents().find((e) => e.type === "OrderConfirmed" && e.orderId === "order-1")!;

  notifications.handle(confirmed);
  notifications.handle(confirmed);
  notifications.handle(confirmed);

  assert.equal(sender.sent.length, 1);
});

test("does not resend on a full replay of the stream", () => {
  const sender = new FakeEmailSender();
  const notifications = new NotificationsService(sender);
  const events = loadCanonicalEvents();

  for (const event of events) notifications.handle(event);
  assert.equal(sender.sent.length, 2);

  // Replay the entire stream again (e.g. the consumer restarted and
  // reprocessed from the beginning): still exactly 2 emails total.
  for (const event of events) notifications.handle(event);
  assert.equal(sender.sent.length, 2);
});
