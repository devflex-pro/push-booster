CREATE DATABASE IF NOT EXISTS push_booster;

CREATE TABLE IF NOT EXISTS push_booster.subscribers (
    source_id UUID,
    subscription_id UUID,
    endpoint String,
    p256dh String,
    auth String,
    user_agent String,
    subscribed_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(subscribed_at)
PARTITION BY toYYYYMM(subscribed_at)
ORDER BY (source_id, subscription_id);
