-- +goose Up
ALTER TABLE campaign_launches
    ADD COLUMN enqueue_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN enqueued_total BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN enqueue_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN enqueue_started_at TIMESTAMPTZ,
    ADD COLUMN enqueue_completed_at TIMESTAMPTZ;

ALTER TABLE campaign_launches
    ADD CONSTRAINT campaign_launches_enqueue_status_check
    CHECK (enqueue_status IN ('pending', 'enqueuing', 'completed', 'failed'));

CREATE INDEX campaign_launches_enqueue_status_idx ON campaign_launches(enqueue_status);

-- +goose Down
ALTER TABLE campaign_launches DROP CONSTRAINT campaign_launches_enqueue_status_check;
DROP INDEX campaign_launches_enqueue_status_idx;
ALTER TABLE campaign_launches
    DROP COLUMN enqueue_status,
    DROP COLUMN enqueued_total,
    DROP COLUMN enqueue_error,
    DROP COLUMN enqueue_started_at,
    DROP COLUMN enqueue_completed_at;
