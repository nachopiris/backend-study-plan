import type { TextMapPropagator, Tracer } from "@opentelemetry/api";
import type { MessageHeaders } from "./types.ts";

// Implementing handleMessage needs, from "@opentelemetry/api": `context` (for context.active()
// and context.with(...)), `trace` (for trace.setSpan/trace.getSpan if needed), `SpanKind` (for
// SpanKind.CONSUMER), and `SpanStatusCode` (for SpanStatusCode.ERROR) — none of those are
// imported here yet because the stub body below does not use them. You will also want
// extractFromMessage from "./propagation.ts". Add the imports yourself when you implement this.

/**
 * handleMessage wraps fn with the tracing a message consumer needs:
 *
 *  1. Extracts the parent trace context from headers (extractFromMessage), so the span started
 *     below becomes a CHILD of whatever producer span injected those headers — closing the trace
 *     across the asynchronous boundary, the same idea as the Go side's ExtractFromMessage +
 *     ServerMiddleware pairing, applied here on the consumer side of a message instead of an
 *     inbound HTTP request.
 *  2. Starts a new span, kind SpanKind.CONSUMER, named spanName, as a child of that extracted
 *     context (tracer.startSpan(name, options, parentContext), NOT tracer.startActiveSpan, so
 *     this function stays easy to unit test without relying on ambient active-context state).
 *  3. Runs fn with that span made "current" for its duration (otelContext.with(contextWithSpan,
 *     fn) — fn itself does not receive the span or context as an argument; it reads it back via
 *     trace.getSpan(otelContext.active()) / otelContext.active() if it needs to, exactly like
 *     real instrumented code does).
 *  4. On success, ends the span. On a thrown/rejected error from fn, records the exception
 *     (span.recordException), sets the span status to SpanStatusCode.ERROR, ends the span, and
 *     RE-THROWS the original error unchanged — handleMessage must never swallow a processing
 *     failure.
 *
 * Not implemented yet: calls fn() directly, with no span, no context propagation, and no error
 * handling beyond whatever fn itself does.
 */
export async function handleMessage(
  tracer: Tracer,
  propagator: TextMapPropagator,
  headers: MessageHeaders,
  spanName: string,
  fn: () => void | Promise<void>,
): Promise<void> {
  await fn();
}
