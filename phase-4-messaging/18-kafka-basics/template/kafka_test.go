package kafkabasics

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// brokers is the address of the single-node Kafka broker started by
// TestMain via `docker compose up`, shared by every test in this package.
const brokers = "localhost:19092"

// skipReason, when non-empty, means the Docker/Kafka infrastructure this
// package's tests need is not available; every test skips itself with this
// message instead of failing.
var skipReason string

// TestMain brings up the single-node KRaft Kafka broker defined in
// docker-compose.yml before running the suite, and tears it down afterwards
// — unless -short was passed or Docker is unavailable, in which case every
// test skips itself with a clear reason instead of failing.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		skipReason = "skipping kafka-basics tests: -short flag set (these tests require Docker + Kafka)"
		os.Exit(m.Run())
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		skipReason = fmt.Sprintf("skipping kafka-basics tests: Docker does not appear to be available: %v", err)
		os.Exit(m.Run())
	}

	up := exec.Command("docker", "compose", "up", "-d", "--wait", "--wait-timeout", "120")
	if out, err := up.CombinedOutput(); err != nil {
		skipReason = fmt.Sprintf("skipping kafka-basics tests: `docker compose up` failed: %v\n%s", err, out)
		os.Exit(m.Run())
	}

	code := m.Run()

	_ = exec.Command("docker", "compose", "down", "-v").Run()
	os.Exit(code)
}

// requireKafka skips the calling test with a clear reason if the Kafka
// broker is not available, otherwise returns the broker address to connect
// to.
func requireKafka(t *testing.T) string {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
	return brokers
}

// uniqueTopic returns a topic name derived from the test name and the
// current time, so parallel test runs (and reruns against a broker that was
// not wiped between runs) never collide on topic state.
func uniqueTopic(t *testing.T, suffix string) string {
	t.Helper()
	safeName := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")
	return fmt.Sprintf("%s-%s-%d", safeName, suffix, time.Now().UnixNano())
}
