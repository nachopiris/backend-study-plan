-- Pre-existing production schema and data, as they are BEFORE this practice's
-- migration starts. This file is given to you as-is — do not edit it; it
-- represents a system that is already live, with real rows already written
-- by app v1 (template/appv1), which is the only version that exists so far.
CREATE TABLE customers (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Rows written by app v1 before anyone thought about renaming `name` to
-- `full_name`. These are exactly the rows Backfill() is responsible for
-- catching up once 002_expand.sql exists: they predate the sync trigger, so
-- the trigger never ran for them and their eventual full_name column stays
-- NULL until Backfill() fixes it.
INSERT INTO customers (name)
SELECT 'Customer ' || i
FROM generate_series(1, 500) AS i;
