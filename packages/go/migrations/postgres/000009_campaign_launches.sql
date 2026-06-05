-- +goose Up
CREATE TABLE campaign_launches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    status TEXT NOT NULL DEFAULT 'building',
    audience_total BIGINT NOT NULL DEFAULT 0,
    processed_total BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT campaign_launches_status_check CHECK (status IN ('building', 'completed', 'failed'))
);

CREATE INDEX campaign_launches_campaign_created_idx ON campaign_launches(campaign_id, created_at DESC);
CREATE INDEX campaign_launches_status_idx ON campaign_launches(status);

-- +goose Down
DROP TABLE campaign_launches;
