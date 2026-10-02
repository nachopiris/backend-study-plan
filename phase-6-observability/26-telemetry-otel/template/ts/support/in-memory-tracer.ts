// GIVEN test helper: builds a BasicTracerProvider wired to an InMemorySpanExporter via a
// SimpleSpanProcessor (synchronous: a span's data is available in the exporter the instant
// span.end() returns, with no batching delay to wait out) — the in-memory test double this
// practice's TS tests use instead of a real OTel Collector. Lives under support/, not test/,
// because node --test treats every file under a test/ directory (or any file matching its
// test-*/*-test naming convention) as a test file (see
// phase-5-microservices/24-bff-contract-testing/template/bff/support/fakes.ts for the same
// convention — this file is named in-memory-tracer.ts, not test-tracer.ts, for exactly that
// reason).
//
// It also registers a global ContextManager (AsyncLocalStorageContextManager), once per process:
// @opentelemetry/api's context.active()/context.with() are no-ops by default — real applications
// register a context manager at startup (NodeSDK does this for you outside of a learning
// exercise), and without one, a correct handleMessage implementation would still fail these
// tests, not because of a bug in your code, but because nothing is propagating "the active span"
// across the fn() callback at all.
import {
  BasicTracerProvider,
  InMemorySpanExporter,
  SimpleSpanProcessor,
} from "@opentelemetry/sdk-trace-base";
import { AsyncLocalStorageContextManager } from "@opentelemetry/context-async-hooks";
import { context as otelContext, type Tracer } from "@opentelemetry/api";

let contextManagerRegistered = false;

function ensureContextManagerRegistered(): void {
  if (contextManagerRegistered) return;
  otelContext.setGlobalContextManager(new AsyncLocalStorageContextManager().enable());
  contextManagerRegistered = true;
}

export interface TestTracer {
  tracer: Tracer;
  exporter: InMemorySpanExporter;
  shutdown: () => Promise<void>;
}

export function newTestTracer(): TestTracer {
  ensureContextManagerRegistered();
  const exporter = new InMemorySpanExporter();
  const provider = new BasicTracerProvider({
    spanProcessors: [new SimpleSpanProcessor(exporter)],
  });
  return {
    tracer: provider.getTracer("test"),
    exporter,
    shutdown: () => provider.shutdown(),
  };
}
