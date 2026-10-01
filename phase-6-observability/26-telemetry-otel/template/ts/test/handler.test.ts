// GIVEN test: you do not need to modify this file. Implementing handleMessage
// (src/handler.ts) correctly is what turns these from RED to GREEN.
import { test } from "node:test";
import assert from "node:assert/strict";
import { context as otelContext, trace, SpanKind, SpanStatusCode } from "@opentelemetry/api";
import { W3CTraceContextPropagator } from "@opentelemetry/core";
import { injectToMessage } from "../src/propagation.ts";
import { handleMessage } from "../src/handler.ts";
import type { MessageHeaders } from "../src/types.ts";
import { newTestTracer } from "../support/in-memory-tracer.ts";

const propagator = new W3CTraceContextPropagator();

function headersFromANewProducerSpan(tracer: ReturnType<typeof newTestTracer>["tracer"]): {
  headers: MessageHeaders;
  producerTraceId: string;
} {
  const producerSpan = tracer.startSpan("produce-order-created");
  const ctx = trace.setSpan(otelContext.active(), producerSpan);
  const headers: MessageHeaders = {};
  injectToMessage(propagator, ctx, headers);
  const producerTraceId = producerSpan.spanContext().traceId;
  producerSpan.end();
  return { headers, producerTraceId };
}

test("handleMessage: runs fn and ends a CONSUMER span that is a child of the producer's trace", async () => {
  const { tracer, exporter, shutdown } = newTestTracer();
  const { headers, producerTraceId } = headersFromANewProducerSpan(tracer);

  let ranWithActiveSpan = false;
  await handleMessage(tracer, propagator, headers, "consume-order-created", () => {
    const active = trace.getSpan(otelContext.active());
    ranWithActiveSpan = active !== undefined;
  });

  assert.ok(ranWithActiveSpan, "fn should run with a span active in context");

  // Read the exporter BEFORE shutting the provider down: InMemorySpanExporter.shutdown() clears
  // its recorded spans as part of stopping.
  const spans = exporter.getFinishedSpans();
  const consumerSpan = spans.find((s) => s.name === "consume-order-created");
  assert.ok(consumerSpan, "expected a span named 'consume-order-created' to have been recorded");
  assert.equal(consumerSpan!.kind, SpanKind.CONSUMER);
  assert.equal(
    consumerSpan!.spanContext().traceId,
    producerTraceId,
    "the consumer span must belong to the same trace as the producer span that injected headers",
  );
  await shutdown();
});

test("handleMessage: resolves normally when fn succeeds, with an OK (not ERROR) span status", async () => {
  const { tracer, exporter, shutdown } = newTestTracer();
  const { headers } = headersFromANewProducerSpan(tracer);

  await handleMessage(tracer, propagator, headers, "consume-ok", async () => {
    await Promise.resolve();
  });

  const span = exporter.getFinishedSpans().find((s) => s.name === "consume-ok");
  assert.ok(span);
  assert.notEqual(span!.status.code, SpanStatusCode.ERROR);
  await shutdown();
});

test("handleMessage: on a thrown error, records it, marks the span ERROR, ends the span, and rethrows", async () => {
  const { tracer, exporter, shutdown } = newTestTracer();
  const { headers } = headersFromANewProducerSpan(tracer);
  const boom = new Error("processing failed");

  await assert.rejects(
    () =>
      handleMessage(tracer, propagator, headers, "consume-fails", () => {
        throw boom;
      }),
    boom,
  );

  const span = exporter.getFinishedSpans().find((s) => s.name === "consume-fails");
  assert.ok(span, "the span must still be ended even though fn threw");
  assert.equal(span!.status.code, SpanStatusCode.ERROR);
  assert.ok(
    span!.events.some((e) => e.name === "exception"),
    "expected span.recordException to have added an 'exception' event",
  );
  await shutdown();
});
