CREATE TABLE IF NOT EXISTS push_booster.subscriber_events (
    source_id UUID,
    subscription_id UUID,
    endpoint String,
    event_type LowCardinality(String),
    user_agent String,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (source_id, occurred_at, subscription_id);
