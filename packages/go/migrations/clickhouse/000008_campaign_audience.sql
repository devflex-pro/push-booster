CREATE TABLE IF NOT EXISTS push_booster.campaign_audience (
    launch_id UUID,
    campaign_id UUID,
    source_id UUID,
    subscription_id UUID,
    endpoint String,
    p256dh String,
    auth String,
    shard UInt16,
    selected_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(selected_at)
PARTITION BY toYYYYMM(selected_at)
ORDER BY (campaign_id, launch_id, shard, subscription_id);
