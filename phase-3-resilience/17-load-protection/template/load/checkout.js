// k6 load test for the Shed-protected endpoint of this practice.
//
// Run it with and without load shedding enabled on your test server (see
// the README for how to wire Shed into a tiny HTTP server), pointed at the
// same ramp of virtual users, and compare:
//   - WITHOUT shedding: watch p95 latency climb (or the process fall over)
//     as concurrency exceeds what the handler can actually do.
//   - WITH shedding: p95 latency on the requests that get through should
//     stay bounded, at the cost of a visible rate of 503 responses once
//     load exceeds capacity. That is the whole point: a readable, bounded
//     degradation instead of an unbounded one.
import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  stages: [
    { duration: "10s", target: 10 }, // warm up, comfortably under capacity
    { duration: "20s", target: 60 }, // ramp well past the configured capacity
    { duration: "10s", target: 0 },  // cool down
  ],
  thresholds: {
    // p95 latency of requests that actually complete (success OR 503) must
    // stay bounded — a shed request should fail FAST, not hang.
    http_req_duration: ["p(95)<800"],
    // Not "errors must be zero": under intentional overload, some 503s are
    // the expected, correct outcome. Adjust this threshold to match what
    // you consider an acceptable shed rate for your own experiment.
    http_req_failed: ["rate<0.90"],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/checkout`);
  check(res, {
    "status is 200 or 503 (never something else, never a hang)": (r) =>
      r.status === 200 || r.status === 503,
  });
  sleep(0.2);
}
