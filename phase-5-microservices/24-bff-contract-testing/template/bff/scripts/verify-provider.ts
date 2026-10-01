// GIVEN script: builds and starts the real Go orders provider (../provider-go) locally, then
// runs Pact JS Verifier against it using the pact file orders-client.pact.test.ts produced, and
// prints the result. Not something you need to implement — the exercise is in src/, not here.
// Run it with `npm run verify-provider` AFTER `npm test` has generated the pact file at least
// once.
//
// Note: this builds the provider with `go build` and runs the resulting BINARY directly, rather
// than `go run .`. `go run` is itself a short-lived wrapper that compiles and then execs a
// separate child process — signaling the wrapper does not reliably stop that child, which would
// leave an orphaned provider holding the port across runs. Running the compiled binary directly
// means the process this script spawns IS the server, so SIGTERM reaches it directly.
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { setTimeout as sleep } from "node:timers/promises";
import { Verifier } from "@pact-foundation/pact";

const PROVIDER_DIR = path.resolve(import.meta.dirname, "..", "..", "provider-go");
const PROVIDER_PORT = 4567;
const PACT_FILE = path.resolve(import.meta.dirname, "..", "pacts", "bff-orders-api.json");

function run(cmd: string, args: string[], cwd: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const proc = spawn(cmd, args, { cwd, stdio: "inherit" });
    proc.once("error", reject);
    proc.once("exit", (code) => {
      if (code === 0) resolve();
      else reject(new Error(`${cmd} ${args.join(" ")} exited with code ${code}`));
    });
  });
}

async function buildProvider(buildDir: string): Promise<string> {
  const binaryPath = path.join(buildDir, "ordersprovider");
  console.log("Building the Go orders provider...");
  await run("go", ["build", "-o", binaryPath, "."], PROVIDER_DIR);
  return binaryPath;
}

async function waitForProvider(baseUrl: string, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseUrl}/healthz`);
      if (res.ok) return;
    } catch {
      // Not up yet — keep polling until the deadline.
    }
    await sleep(100);
  }
  throw new Error(`provider at ${baseUrl} did not become healthy within ${timeoutMs}ms`);
}

function stopProvider(proc: ChildProcess): Promise<void> {
  return new Promise((resolve) => {
    if (proc.exitCode !== null) {
      resolve();
      return;
    }
    proc.once("exit", () => resolve());
    proc.kill("SIGTERM");
    // Hard fallback in case the process ignores SIGTERM for some reason.
    setTimeout(() => {
      if (proc.exitCode === null) proc.kill("SIGKILL");
    }, 2000).unref();
  });
}

async function main(): Promise<void> {
  const providerBaseUrl = `http://127.0.0.1:${PROVIDER_PORT}`;
  const buildDir = await mkdtemp(path.join(os.tmpdir(), "orders-provider-build-"));

  let proc: ChildProcess | undefined;
  try {
    const binaryPath = await buildProvider(buildDir);

    console.log(`Starting the orders provider on port ${PROVIDER_PORT}...`);
    proc = spawn(binaryPath, [], {
      env: { ...process.env, PORT: String(PROVIDER_PORT) },
      stdio: "inherit",
    });

    await waitForProvider(providerBaseUrl, 15_000);
    console.log("Provider is up. Verifying the pact...");

    await new Verifier({
      provider: "orders-api",
      providerBaseUrl,
      pactUrls: [PACT_FILE],
    }).verifyProvider();

    console.log("Provider verification passed.");
  } catch (err) {
    console.error("Provider verification FAILED.");
    console.error(err instanceof Error ? err.message : err);
    process.exitCode = 1;
  } finally {
    if (proc) await stopProvider(proc);
    await rm(buildDir, { recursive: true, force: true });
  }
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
