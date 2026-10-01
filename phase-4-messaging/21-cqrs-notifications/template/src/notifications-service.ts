// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase.

import type { OrderEvent } from "./types.ts";
import type { EmailSender } from "./email-sender.ts";

/**
 * NotificationsService reacts to confirmed orders by sending exactly one
 * email per confirmation, no matter how many times handle() is called with
 * the same event (replay) or a duplicate delivery of it.
 *
 * Contract:
 *   - handle(event) is a no-op for every event.type other than
 *     "OrderConfirmed".
 *   - For an "OrderConfirmed" event, handle() sends exactly ONE email via
 *     sender the FIRST time it sees that event's id; every later call with
 *     an event carrying the SAME id (a duplicate delivery, or a replay of
 *     the whole stream) must NOT send another email.
 *   - The email's `to` does not matter for this practice's tests (you can
 *     use event.orderId as a placeholder recipient, or anything else
 *     reasonable); what the tests check is HOW MANY emails were sent.
 *
 * Not implemented yet: never sends anything.
 */
export class NotificationsService {
  private readonly sender: EmailSender;

  constructor(sender: EmailSender) {
    this.sender = sender;
  }

  handle(event: OrderEvent): void {
  }
}
