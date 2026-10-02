import type { Order, OrdersPort } from "./types.ts";
import { fetchWithTimeout, type FetchLike } from "./http.ts";

export class OrderNotFoundError extends Error {
  readonly orderId: string;

  // No parameter properties: this is erasable-only TypeScript (see src/http.ts).
  constructor(orderId: string) {
    super(`order ${orderId} not found`);
    this.name = "OrderNotFoundError";
    this.orderId = orderId;
  }
}

export interface OrdersClientOptions {
  baseUrl: string;
  timeoutMs: number;
  fetchImpl?: FetchLike;
}

/**
 * OrdersClient is the consumer side of the contract this practice verifies with Pact: see
 * test/orders-client.pact.test.ts for the consumer tests that define it, and ../provider-go for
 * the real Go provider the generated pact file gets verified against.
 */
export class OrdersClient implements OrdersPort {
  private readonly options: OrdersClientOptions;
  private readonly fetchImpl: FetchLike;

  constructor(options: OrdersClientOptions) {
    this.options = options;
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  /**
   * Contract:
   *   - Sends `GET ${baseUrl}/orders/${orderId}`, bounded by timeoutMs (see fetchWithTimeout).
   *   - A 200 response is parsed as JSON and returned as an Order.
   *   - A 404 response throws OrderNotFoundError(orderId).
   *   - Any other non-OK status, a timeout, or a network error propagates as a thrown error —
   *     this client's job is to report what happened accurately, not to hide it. (Deciding what
   *     to do about a failed order fetch belongs to OrderViewService.)
   */
  async getOrder(orderId: string): Promise<Order> {
    throw new Error("not implemented");
  }
}
