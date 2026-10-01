#!/bin/bash
# Runs once, automatically, as a docker-entrypoint-initdb.d script on the
# PRIMARY the very first time its data directory is created. It creates the
# replication role the replica uses to stream from this primary, and opens
# pg_hba.conf to replication connections from anywhere on the compose
# network. This is infrastructure plumbing, not part of the exercise — you
# should not need to change this file.
set -euo pipefail

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-'EOSQL'
    CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replicator_password';
EOSQL

echo "host replication replicator all md5" >> "$PGDATA/pg_hba.conf"
