// Erasable-only TypeScript: no enums, no namespaces, no parameter
// properties — only types Node's native type stripping can erase (see
// phase-5-microservices/24-bff-contract-testing for the same convention).

/**
 * The shape this exercise uses for message headers: a plain string-to-string record, the same
 * shape propagation.TextMapPropagator's default getter/setter already understand (see
 * src/propagation.ts) without needing a custom TextMapGetter/TextMapSetter.
 *
 * Real message brokers vary in how they represent headers in their client library (Buffers,
 * arrays of {key, value} pairs with possibly repeated keys, and so on — see the Go side of this
 * practice's HeadersCarrier for one such shape). This exercise picks the simplest shape that
 * still lets W3C trace context survive the trip, so the exercise is about the PROPAGATION
 * mechanism, not about adapting to one specific client library's header type.
 */
export type MessageHeaders = Record<string, string>;
