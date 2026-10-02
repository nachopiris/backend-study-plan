-- Given schema for this practice.
--
-- orders / outbox: PlaceOrder must insert into both in the SAME transaction
-- (that is the whole point of the transactional outbox pattern: the business
-- write and the "I need to publish this" record either both happen or
-- neither does).
--
-- processed_messages / dead_letters: used by the idempotent consumer side
-- (Consumer.Process), independent of the relay.
CREATE TABLE IF NOT EXISTS orders (
    id            TEXT PRIMARY KEY,
    customer_id   TEXT NOT NULL,
    amount_cents  BIGINT NOT NULL CHECK (amount_cents >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- sent_at IS NULL means "not yet published"; Relay.RunOnce claims rows where
-- this is the case. aggregate_id is the order id; event_type and payload
-- describe the domain event ("OrderPlaced", a JSON blob with the order's
-- data) that downstream consumers care about.
CREATE TABLE IF NOT EXISTS outbox (
    id            BIGSERIAL PRIMARY KEY,
    aggregate_id  TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_unsent_idx ON outbox (id) WHERE sent_at IS NULL;

-- One row per message_id that was ever successfully processed. The
-- `INSERT ... ON CONFLICT DO NOTHING` into this table, executed in the SAME
-- transaction as the message's side effect, is what makes Consumer.Process
-- idempotent: a redelivered message_id finds its row already here and skips
-- the side effect entirely.
CREATE TABLE IF NOT EXISTS processed_messages (
    message_id    TEXT PRIMARY KEY,
    processed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tracks how many times a message_id has failed processing, so Consumer can
-- decide when to give up and dead-letter it instead of retrying forever.
CREATE TABLE IF NOT EXISTS delivery_attempts (
    message_id  TEXT PRIMARY KEY,
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT
);

-- A message that failed processing maxAttempts times ends up here instead of
-- being retried again.
CREATE TABLE IF NOT EXISTS dead_letters (
    id              BIGSERIAL PRIMARY KEY,
    message_id      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    failure_reason  TEXT NOT NULL,
    attempts        INT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
