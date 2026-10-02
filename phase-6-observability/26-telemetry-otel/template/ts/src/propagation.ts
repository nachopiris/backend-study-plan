import { context as otelContext, type Context } from "@opentelemetry/api";
import type { TextMapPropagator } from "@opentelemetry/api";
import type { MessageHeaders } from "./types.ts";

/**
 * injectToMessage writes the trace context active in ctx into headers (in place), using
 * propagator. Call this right before "publishing" a message — see test/propagation.test.ts for
 * what the resulting headers look like, and the Go side of this practice (carrier.go) for the
 * same idea applied to a Kafka-shaped header type instead of a plain record.
 *
 * MessageHeaders (a plain Record<string, string>) is already a valid carrier for
 * propagator.inject's default setter — you do NOT need a custom TextMapSetter for this exercise's
 * header shape (see src/types.ts).
 *
 * Not implemented yet: no-op (headers is left unchanged).
 */
export function injectToMessage(
  propagator: TextMapPropagator,
  ctx: Context,
  headers: MessageHeaders,
): void {
  // not implemented
}

/**
 * extractFromMessage returns a Context derived from otelContext.active() carrying the trace
 * context found in headers (if any), using propagator. Call this when a consumer receives a
 * message, before starting its own span for processing it, so that new span becomes a CHILD of
 * whatever producer span called injectToMessage.
 *
 * Contract: if headers carries no valid trace context (for example, {} or headers published by
 * code that never called injectToMessage), extractFromMessage returns a usable Context in which a
 * span started afterwards simply becomes a new trace root — not an error, and not a thrown
 * exception.
 *
 * Not implemented yet: returns otelContext.active() unchanged, ignoring headers entirely.
 */
export function extractFromMessage(
  propagator: TextMapPropagator,
  headers: MessageHeaders,
): Context {
  return otelContext.active();
}
