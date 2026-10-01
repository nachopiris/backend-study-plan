package replication

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	primaryPool *pgxpool.Pool
	replicaPool *pgxpool.Pool
	skipReason  string
)

// TestMain connects to a primary and a replica whose DSNs are read from the
// PRIMARY_DSN and REPLICA_DSN environment variables — this practice uses
// docker-compose (see README) instead of testcontainers, because setting up
// real streaming replication between two containers needs more plumbing
// (a replication role, pg_hba.conf, pg_basebackup) than testcontainers-go's
// single-container Postgres module targets.
//
// Every test skips itself with a clear reason if those variables are not
// set, or if -short was passed.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		skipReason = "skipping replication-lag tests: -short flag set (these tests require `docker compose up -d`, see README)"
		os.Exit(m.Run())
	}

	primaryDSN := os.Getenv("PRIMARY_DSN")
	replicaDSN := os.Getenv("REPLICA_DSN")
	if primaryDSN == "" || replicaDSN == "" {
		skipReason = "skipping replication-lag tests: PRIMARY_DSN and/or REPLICA_DSN not set " +
			"(run `docker compose up -d` and export both, see README)"
		os.Exit(m.Run())
	}

	ctx := context.Background()

	primary, err := pgxpool.New(ctx, primaryDSN)
	if err != nil {
		fmt.Println("failed to connect to primary:", err)
		os.Exit(1)
	}
	if err := primary.Ping(ctx); err != nil {
		fmt.Println("failed to ping primary:", err)
		os.Exit(1)
	}

	replica, err := pgxpool.New(ctx, replicaDSN)
	if err != nil {
		fmt.Println("failed to connect to replica:", err)
		os.Exit(1)
	}
	if err := replica.Ping(ctx); err != nil {
		fmt.Println("failed to ping replica:", err)
		os.Exit(1)
	}

	primaryPool = primary
	replicaPool = replica

	code := m.Run()

	primary.Close()
	replica.Close()
	os.Exit(code)
}

// requirePools returns the shared primary/replica pools, or skips the
// calling test with a clear reason if the environment does not support
// these tests.
func requirePools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
	return primaryPool, replicaPool
}
