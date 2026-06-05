-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN audience_scope TEXT NOT NULL DEFAULT 'selected_sources';

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_audience_scope_check
    CHECK (audience_scope IN ('all', 'selected_sources'));

ALTER TABLE campaigns
    ALTER COLUMN publisher_id DROP NOT NULL,
    ALTER COLUMN source_id DROP NOT NULL;

CREATE TABLE campaign_sources (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES sources(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (campaign_id, source_id)
);

INSERT INTO campaign_sources (campaign_id, source_id)
SELECT id, source_id
FROM campaigns
WHERE source_id IS NOT NULL
ON CONFLICT DO NOTHING;

CREATE INDEX campaign_sources_source_id_idx ON campaign_sources(source_id);
CREATE INDEX campaigns_audience_scope_idx ON campaigns(audience_scope, status);

-- +goose Down
DROP INDEX campaigns_audience_scope_idx;
DROP INDEX campaign_sources_source_id_idx;
DROP TABLE campaign_sources;

ALTER TABLE campaigns
    DROP CONSTRAINT campaigns_audience_scope_check,
    DROP COLUMN audience_scope;

ALTER TABLE campaigns
    ALTER COLUMN publisher_id SET NOT NULL,
    ALTER COLUMN source_id SET NOT NULL;
