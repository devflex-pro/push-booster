-- +goose Up
CREATE TABLE vapid_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    public_key TEXT NOT NULL UNIQUE,
    private_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vapid_keys_status_check CHECK (status IN ('active', 'deprecated', 'revoked'))
);

ALTER TABLE sources
    ADD COLUMN vapid_key_id UUID REFERENCES vapid_keys(id);

CREATE INDEX vapid_keys_status_idx ON vapid_keys(status);
CREATE INDEX sources_vapid_key_id_idx ON sources(vapid_key_id);

-- +goose Down
DROP INDEX sources_vapid_key_id_idx;
DROP INDEX vapid_keys_status_idx;

ALTER TABLE sources
    DROP COLUMN vapid_key_id;

DROP TABLE vapid_keys;
