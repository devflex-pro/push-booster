-- +goose Up
CREATE TABLE postback_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    source_id UUID REFERENCES sources(id),
    token TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    click_id_param TEXT NOT NULL DEFAULT 'click_id',
    delivery_id_param TEXT NOT NULL DEFAULT 'delivery_id',
    subscription_id_param TEXT NOT NULL DEFAULT 'subscription_id',
    external_id_param TEXT NOT NULL DEFAULT 'external_id',
    payout_param TEXT NOT NULL DEFAULT 'payout',
    currency_param TEXT NOT NULL DEFAULT 'currency',
    status_param TEXT NOT NULL DEFAULT 'status',
    default_currency TEXT NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT postback_configs_status_check CHECK (status IN ('active', 'paused', 'archived'))
);

CREATE INDEX postback_configs_source_id_idx ON postback_configs(source_id);
CREATE INDEX postback_configs_status_idx ON postback_configs(status);

-- +goose Down
DROP TABLE postback_configs;
