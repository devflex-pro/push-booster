-- +goose Up
ALTER TABLE creatives
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN provider_config_id UUID,
    ADD COLUMN provider_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN provider_external_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN raw_provider_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN last_synced_at TIMESTAMPTZ,
    ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'synced',
    ADD CONSTRAINT creatives_source_type_check CHECK (source_type IN ('manual', 'provider_api')),
    ADD CONSTRAINT creatives_sync_status_check CHECK (sync_status IN ('synced', 'invalid', 'stale'));

CREATE TABLE creative_provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    name TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    fetch_url TEXT NOT NULL,
    request_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'active',
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT creative_provider_configs_status_check CHECK (status IN ('active', 'paused', 'archived'))
);

CREATE TABLE creative_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_config_id UUID NOT NULL REFERENCES creative_provider_configs(id),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    status TEXT NOT NULL,
    fetched_total INTEGER NOT NULL DEFAULT 0,
    upserted_total INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT creative_sync_logs_status_check CHECK (status IN ('running', 'completed', 'failed'))
);

ALTER TABLE creatives
    ADD CONSTRAINT creatives_provider_config_fk
    FOREIGN KEY (provider_config_id) REFERENCES creative_provider_configs(id);

CREATE INDEX creative_provider_configs_campaign_idx ON creative_provider_configs(campaign_id, status);
CREATE INDEX creative_sync_logs_config_started_idx ON creative_sync_logs(provider_config_id, started_at DESC);
CREATE UNIQUE INDEX creatives_provider_external_unique_idx
    ON creatives(campaign_id, provider_name, provider_external_id)
    WHERE source_type = 'provider_api' AND provider_external_id <> '';

-- +goose Down
DROP INDEX creatives_provider_external_unique_idx;
DROP INDEX creative_sync_logs_config_started_idx;
DROP INDEX creative_provider_configs_campaign_idx;
ALTER TABLE creatives DROP CONSTRAINT creatives_provider_config_fk;
DROP TABLE creative_sync_logs;
DROP TABLE creative_provider_configs;
ALTER TABLE creatives
    DROP CONSTRAINT creatives_sync_status_check,
    DROP CONSTRAINT creatives_source_type_check,
    DROP COLUMN sync_status,
    DROP COLUMN last_synced_at,
    DROP COLUMN raw_provider_payload,
    DROP COLUMN provider_external_id,
    DROP COLUMN provider_name,
    DROP COLUMN provider_config_id,
    DROP COLUMN source_type;
