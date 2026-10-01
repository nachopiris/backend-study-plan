-- Given schema for this practice: a single table, replicated from primary to
-- replica by Postgres's own streaming replication (not by anything in this
-- repository). Router and NaiveRouter only read/write this table.
CREATE TABLE IF NOT EXISTS notes (
    id          BIGSERIAL PRIMARY KEY,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
