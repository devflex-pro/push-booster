ALTER TABLE push_booster.subscriber_events
    ADD COLUMN IF NOT EXISTS delivery_id String AFTER source_id,
    ADD COLUMN IF NOT EXISTS campaign_id String AFTER delivery_id,
    ADD COLUMN IF NOT EXISTS creative_id String AFTER campaign_id,
    ADD COLUMN IF NOT EXISTS event_id String AFTER creative_id,
    ADD COLUMN IF NOT EXISTS target_url String AFTER event_id;
