-- Given schema for this practice: a single SKU's stock, with a version
-- column for optimistic concurrency control, and an orders table recording
-- successful purchases.
--
-- The CHECK constraint is a structural safety net, not a substitute for
-- correct application logic: your implementation should never rely on
-- Postgres rejecting a negative stock update, but it is good practice to
-- have the database refuse to store an impossible state regardless.
CREATE TABLE IF NOT EXISTS inventory (
    sku      TEXT PRIMARY KEY,
    stock    INT NOT NULL CHECK (stock >= 0),
    version  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS orders (
    id          BIGSERIAL PRIMARY KEY,
    sku         TEXT NOT NULL REFERENCES inventory(sku),
    buyer_id    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
