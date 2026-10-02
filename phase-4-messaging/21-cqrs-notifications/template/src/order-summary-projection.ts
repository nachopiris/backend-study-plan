// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase.

import type { OrderEvent, OrderSummary } from "./types.ts";

/**
 * OrderSummaryProjection builds an OrderSummary read model from a stream of
 * OrderEvents that may arrive DUPLICATED and/or OUT OF ORDER (per orderId).
 *
 * Contract:
 *   - apply(event) is idempotent by event.id: applying the exact same
 *     event.id more than once has no additional effect after the first
 *     time, no matter how many times it is called.
 *   - Events for the same orderId must take effect in VERSION order
 *     (1, then 2, then 3, ...), no matter what order apply() is actually
 *     called in. An event whose version is not yet applicable — its
 *     predecessor version has not been applied yet — is held until its turn
 *     comes, and is then applied, together with any other now-applicable
 *     events it unblocks, in version order.
 *   - get(orderId) reflects every event that COULD be applied so far (no gap
 *     before it); it never reflects an event that is still waiting for an
 *     earlier version that has not arrived yet.
 *   - Field updates per event type:
 *       - OrderCreated:   status = "pending", customerId = data.customerId,
 *                         totalCents = 0, itemCount = 0,
 *                         createdAt = updatedAt = event.occurredAt.
 *       - OrderItemAdded: itemCount += data.quantity,
 *                         totalCents += data.unitPriceCents * data.quantity,
 *                         updatedAt = event.occurredAt.
 *       - OrderConfirmed: status = "confirmed", updatedAt = event.occurredAt.
 *       - OrderCancelled: status = "cancelled", updatedAt = event.occurredAt.
 *   - rebuild(events) replays every event in `events` (fed through apply(),
 *     in whatever order the array gives them) into a FRESH projection and
 *     returns it. Feeding the same set of events to rebuild(), regardless of
 *     the array's order, must produce the same final summaries as applying
 *     them one by one in whatever order they actually arrived live.
 *
 * Not implemented yet: every method throws.
 */
export class OrderSummaryProjection {
  apply(event: OrderEvent): void {
    throw new Error("not implemented");
  }

  get(orderId: string): OrderSummary | undefined {
    throw new Error("not implemented");
  }

  static rebuild(events: OrderEvent[]): OrderSummaryProjection {
    throw new Error("not implemented");
  }
}
