// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase.

export interface OrderItem {
  sku: string;
  quantity: number;
  unitPriceCents: number;
}

/** The shape the orders-api provider returns (see ../provider-go). */
export interface Order {
  id: string;
  customerId: string;
  status: "pending" | "confirmed" | "cancelled";
  items: OrderItem[];
  totalCents: number;
}

/**
 * "unknown" is not a status the payments service ever reports itself — it is this BFF's own
 * fallback when the payments service cannot be reached in time. See OrderViewService.
 */
export type PaymentStatus = "paid" | "pending" | "failed" | "unknown";

/** The aggregated, read-optimized view this BFF exists to produce for the frontend. */
export interface OrderView {
  orderId: string;
  customerId: string;
  status: Order["status"];
  items: OrderItem[];
  totalCents: number;
  paymentStatus: PaymentStatus;
}

/**
 * OrdersPort and PaymentsPort are the shapes OrderViewService depends on. OrdersClient and
 * PaymentsClient (the real HTTP adapters) satisfy them structurally; tests for OrderViewService
 * use lightweight fakes instead (see support/fakes.ts), the same hexagonal idea used throughout
 * this plan's Go practices, applied here to TypeScript.
 */
export interface OrdersPort {
  getOrder(orderId: string): Promise<Order>;
}

export interface PaymentsPort {
  getPaymentStatus(orderId: string): Promise<PaymentStatus>;
}
