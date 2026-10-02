import type { PaymentStatus, PaymentsPort } from "./types.ts";
import { fetchWithTimeout, type FetchLike } from "./http.ts";

export interface PaymentsClientOptions {
  baseUrl: string;
  timeoutMs: number;
  fetchImpl?: FetchLike;
}

/**
 * PaymentsClient has no Pact contract in this practice — there is no given payments provider to
 * verify a pact file against (only orders-api is, via ../provider-go). It is exercised instead
 * through OrderViewService's own tests (order-view-service.test.ts), which simulate the payments
 * service being slow or unreachable to prove the BFF degrades gracefully.
 */
export class PaymentsClient implements PaymentsPort {
  private readonly options: PaymentsClientOptions;
  private readonly fetchImpl: FetchLike;

  constructor(options: PaymentsClientOptions) {
    this.options = options;
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  /**
   * Contract:
   *   - Sends `GET ${baseUrl}/payments/by-order/${orderId}`, bounded by timeoutMs.
   *   - A 200 response is parsed as JSON; its `status` field is returned as PaymentStatus.
   *   - Any non-OK status, a timeout, or a network error propagates as a thrown error — this
   *     client never substitutes "unknown" itself. That fallback policy belongs to
   *     OrderViewService, the only place that knows the view is still useful without it.
   */
  async getPaymentStatus(orderId: string): Promise<PaymentStatus> {
    throw new Error("not implemented");
  }
}
