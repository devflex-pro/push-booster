CREATE TABLE IF NOT EXISTS push_booster.payload_decisions (
    trigger_id String,
    subscription_id String,
    source_id String,
    campaign_id String,
    creative_id String,
    result LowCardinality(String),
    reason LowCardinality(String),
    error String,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (occurred_at, source_id, campaign_id, subscription_id, trigger_id);
