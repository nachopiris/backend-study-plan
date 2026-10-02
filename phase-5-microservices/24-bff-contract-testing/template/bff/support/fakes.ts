// GIVEN test helper: lightweight fakes for OrdersPort and PaymentsPort (see ../src/types.ts),
// used by order-view-service.test.ts to exercise OrderViewService's aggregation and degradation
// logic without any real HTTP calls. Lives under support/, not test/, because node --test treats
// every file under a test/ directory as a test file.
import type { Order, OrdersPort, PaymentsPort, PaymentStatus } from "../src/types.ts";

export function sampleOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: "order-1",
    customerId: "cust-1",
    status: "confirmed",
    items: [{ sku: "sku-1", quantity: 1, unitPriceCents: 1000 }],
    totalCents: 1000,
    ...overrides,
  };
}

export class FakeOrdersClient implements OrdersPort {
  calls = 0;
  private readonly behavior: () => Promise<Order>;

  constructor(behavior: () => Promise<Order>) {
    this.behavior = behavior;
  }

  async getOrder(_orderId: string): Promise<Order> {
    this.calls++;
    return this.behavior();
  }
}

export class FakePaymentsClient implements PaymentsPort {
  calls = 0;
  private readonly behavior: () => Promise<PaymentStatus>;

  constructor(behavior: () => Promise<PaymentStatus>) {
    this.behavior = behavior;
  }

  async getPaymentStatus(_orderId: string): Promise<PaymentStatus> {
    this.calls++;
    return this.behavior();
  }
}

export function immediate<T>(value: T): () => Promise<T> {
  return () => Promise.resolve(value);
}

export function immediateFailure<T>(error: Error): () => Promise<T> {
  return () => Promise.reject(error);
}

/** Resolves to value after delayMs — use to simulate a slow but eventually-responsive dependency. */
export function delayed<T>(value: T, delayMs: number): () => Promise<T> {
  return () => new Promise((resolve) => setTimeout(() => resolve(value), delayMs));
}

/** Rejects with error after delayMs — use to simulate a dependency that is down. */
export function delayedFailure<T>(error: Error, delayMs: number): () => Promise<T> {
  return () => new Promise((_resolve, reject) => setTimeout(() => reject(error), delayMs));
}
