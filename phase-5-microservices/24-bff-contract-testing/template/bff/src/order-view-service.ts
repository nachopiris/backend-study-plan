import type { OrderView, OrdersPort, PaymentsPort } from "./types.ts";

/**
 * OrderViewService is the BFF's core responsibility: fan out to the orders and payments APIs and
 * aggregate the result into one read-optimized view, so the frontend makes one call instead of
 * two and never has to know these are separate services.
 *
 * It depends on OrdersPort/PaymentsPort (types.ts), not on the concrete HTTP clients, so its own
 * tests (order-view-service.test.ts) can use lightweight fakes (support/fakes.ts) instead of
 * real network calls.
 */
export class OrderViewService {
  private readonly ordersClient: OrdersPort;
  private readonly paymentsClient: PaymentsPort;

  // No parameter properties: this is erasable-only TypeScript (see src/http.ts), and parameter
  // properties are TS-only syntax Node's native type stripping cannot erase.
  constructor(ordersClient: OrdersPort, paymentsClient: PaymentsPort) {
    this.ordersClient = ordersClient;
    this.paymentsClient = paymentsClient;
  }

  /**
   * Contract:
   *   - Calls ordersClient.getOrder(orderId) and paymentsClient.getPaymentStatus(orderId)
   *     CONCURRENTLY (do not await the order before starting the payment status request).
   *   - If the order fetch fails for ANY reason (not found, timeout, network error),
   *     getOrderView itself rejects with that error — there is no view to build without the
   *     order.
   *   - If the payment status fetch fails for ANY reason, it must NOT propagate and must NOT
   *     prevent the view from being returned: the resulting OrderView has
   *     paymentStatus: "unknown" instead. This is the partial degradation this practice is
   *     about — a slow or unreachable payments service degrades the view, it does not break it.
   */
  async getOrderView(orderId: string): Promise<OrderView> {
    throw new Error("not implemented");
  }
}
