ALTER TABLE push_booster.subscribers
    ADD COLUMN IF NOT EXISTS subscription_id UUID AFTER source_id;

ALTER TABLE push_booster.subscriber_events
    ADD COLUMN IF NOT EXISTS subscription_id UUID AFTER source_id;
