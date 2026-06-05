CREATE TABLE IF NOT EXISTS push_booster.postback_events (
    postback_config_id UUID,
    dedupe_key String,
    external_id String,
    click_id String,
    delivery_id String,
    subscription_id String,
    source_id String,
    campaign_id String,
    creative_id String,
    payout Float64,
    currency LowCardinality(String),
    status LowCardinality(String),
    attribution_status LowCardinality(String),
    raw_payload String,
    received_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(received_at)
ORDER BY (postback_config_id, received_at, dedupe_key);
