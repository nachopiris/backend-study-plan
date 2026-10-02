// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase.

/** The kinds of event this stream can carry. */
export type OrderEventType = "OrderCreated" | "OrderItemAdded" | "OrderConfirmed" | "OrderCancelled";

/**
 * OrderEvent is one entry in an order's event stream. version is a 1-based,
 * strictly increasing sequence number PER orderId (1 for the first event
 * about that order, 2 for the second, and so on) — it is what lets a
 * projection detect and tolerate out-of-order delivery, independently of
 * whatever order events happen to arrive in.
 *
 * data's shape depends on type:
 *   - OrderCreated:   { customerId: string }
 *   - OrderItemAdded: { unitPriceCents: number; quantity: number }
 *   - OrderConfirmed: {}
 *   - OrderCancelled: { reason: string }
 */
export interface OrderEvent {
  id: string;
  orderId: string;
  type: OrderEventType;
  version: number;
  occurredAt: string;
  data: Record<string, unknown>;
}

/** The CQRS read model this practice builds: one summary per order. */
export interface OrderSummary {
  orderId: string;
  status: "pending" | "confirmed" | "cancelled";
  customerId: string | null;
  totalCents: number;
  itemCount: number;
  createdAt: string | null;
  updatedAt: string | null;
}
