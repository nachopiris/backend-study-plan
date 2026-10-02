#!/bin/bash
# Custom entrypoint for the REPLICA service. This is infrastructure
# plumbing, not part of the exercise — you should not need to change this
# file.
#
# On first start (empty $PGDATA), it waits for the primary to accept
# connections, clones it with pg_basebackup (which also writes standby.signal
# and primary_conninfo via -R), appends a deterministic apply delay so the
# replica is reliably "behind" the primary, then hands off to the normal
# Postgres entrypoint, which starts the server in standby mode.
set -euo pipefail

PRIMARY_HOST="${PRIMARY_HOST:-primary}"
PRIMARY_PORT="${PRIMARY_PORT:-5432}"
REPL_USER="${REPL_USER:-replicator}"
RECOVERY_MIN_APPLY_DELAY="${RECOVERY_MIN_APPLY_DELAY:-1500ms}"
export PGPASSWORD="${REPL_PASSWORD:-replicator_password}"

if [ -z "$(ls -A "$PGDATA" 2>/dev/null)" ]; then
    echo "bootstrap-replica: empty PGDATA, waiting for primary at $PRIMARY_HOST:$PRIMARY_PORT..."
    until pg_isready -h "$PRIMARY_HOST" -p "$PRIMARY_PORT" -U "$REPL_USER" >/dev/null 2>&1; do
        sleep 1
    done

    echo "bootstrap-replica: cloning primary with pg_basebackup..."
    pg_basebackup \
        -h "$PRIMARY_HOST" -p "$PRIMARY_PORT" -U "$REPL_USER" \
        -D "$PGDATA" -Fp -Xs -P -R

    echo "recovery_min_apply_delay = '${RECOVERY_MIN_APPLY_DELAY}'" >> "$PGDATA/postgresql.auto.conf"
    chmod 0700 "$PGDATA"
fi

echo "bootstrap-replica: handing off to the standard postgres entrypoint..."
exec docker-entrypoint.sh postgres
