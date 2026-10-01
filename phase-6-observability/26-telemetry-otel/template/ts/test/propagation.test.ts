// GIVEN test: you do not need to modify this file. Implementing injectToMessage/extractFromMessage
// (src/propagation.ts) correctly is what turns these from RED to GREEN.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { context as otelContext, trace, SpanKind } from "@opentelemetry/api";
import { W3CTraceContextPropagator } from "@opentelemetry/core";
import { injectToMessage, extractFromMessage } from "../src/propagation.ts";
import type { MessageHeaders } from "../src/types.ts";
import { newTestTracer } from "../support/in-memory-tracer.ts";

const propagator = new W3CTraceContextPropagator();

test("injectToMessage: writes a traceparent header matching the active span's trace and span IDs", () => {
  const { tracer, shutdown } = newTestTracer();
  const span = tracer.startSpan("produce-order-created");
  const ctx = trace.setSpan(otelContext.active(), span);

  const headers: MessageHeaders = {};
  injectToMessage(propagator, ctx, headers);
  span.end();

  assert.ok(headers.traceparent, "expected a traceparent header to be written");
  const sc = span.spanContext();
  assert.ok(
    headers.traceparent!.includes(sc.traceId),
    `traceparent ${headers.traceparent} should contain trace id ${sc.traceId}`,
  );
  assert.ok(
    headers.traceparent!.includes(sc.spanId),
    `traceparent ${headers.traceparent} should contain span id ${sc.spanId}`,
  );
  return shutdown();
});

test("extractFromMessage: a span started from the extracted context is a child of the injected trace", async () => {
  const { tracer, exporter, shutdown } = newTestTracer();
  const producerSpan = tracer.startSpan("produce-order-created");
  const producerCtx = trace.setSpan(otelContext.active(), producerSpan);

  const headers: MessageHeaders = {};
  injectToMessage(propagator, producerCtx, headers);
  producerSpan.end();

  const consumerParentCtx = extractFromMessage(propagator, headers);
  const consumerSpan = tracer.startSpan(
    "consume-order-created",
    { kind: SpanKind.CONSUMER },
    consumerParentCtx,
  );
  consumerSpan.end();

  assert.equal(
    consumerSpan.spanContext().traceId,
    producerSpan.spanContext().traceId,
    "consumer span should share the producer span's trace ID",
  );
  // Read the exporter BEFORE shutting the provider down: InMemorySpanExporter.shutdown() clears
  // its recorded spans as part of stopping, so this assertion would otherwise always see 0.
  assert.equal(exporter.getFinishedSpans().length, 2);
  await shutdown();
});

test("extractFromMessage: with headers carrying no trace context, returns a usable context (new trace root)", () => {
  const { tracer, shutdown } = newTestTracer();
  const ctx = extractFromMessage(propagator, {});
  const span = tracer.startSpan("consume-with-no-parent", {}, ctx);
  span.end();

  assert.ok(span.spanContext().traceId, "span should still get a valid (new) trace id");
  return shutdown();
});

test("cross-language round trip: a trace context produced by the Go side is understood here", () => {
  const fixturePath = fileURLToPath(
    new URL("../fixtures/headers-from-go.json", import.meta.url),
  );
  const fixture = JSON.parse(readFileSync(fixturePath, "utf8")) as {
    expectedTraceId: string;
    headers: MessageHeaders;
  };

  const { tracer, shutdown } = newTestTracer();
  const ctx = extractFromMessage(propagator, fixture.headers);
  const span = tracer.startSpan("consume-order-created", { kind: SpanKind.CONSUMER }, ctx);
  span.end();

  assert.equal(
    span.spanContext().traceId,
    fixture.expectedTraceId,
    "the span started from the extracted context must continue the SAME trace the Go-shaped fixture carried",
  );
  return shutdown();
});
