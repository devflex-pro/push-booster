CREATE TABLE IF NOT EXISTS push_booster.push_events (
    delivery_id UUID,
    trigger_id UUID,
    launch_id UUID,
    campaign_id UUID,
    source_id UUID,
    subscription_id UUID,
    event_type LowCardinality(String),
    attempt UInt16,
    error String,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (campaign_id, occurred_at, delivery_id, subscription_id);
