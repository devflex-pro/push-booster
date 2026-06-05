CREATE TABLE IF NOT EXISTS push_booster.creative_exposures (
    source_id UUID,
    subscription_id UUID,
    campaign_id UUID,
    creative_id UUID,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (campaign_id, subscription_id, occurred_at, creative_id);
