-- +goose Up
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id UUID NOT NULL REFERENCES publishers(id),
    source_id UUID NOT NULL REFERENCES sources(id),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaigns_status_check CHECK (status IN ('draft', 'active', 'paused', 'archived'))
);

CREATE TABLE creatives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    url TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT creatives_status_check CHECK (status IN ('active', 'paused', 'archived'))
);

CREATE INDEX campaigns_source_status_idx ON campaigns(source_id, status);
CREATE INDEX campaigns_publisher_id_idx ON campaigns(publisher_id);
CREATE INDEX creatives_campaign_status_idx ON creatives(campaign_id, status);

-- +goose Down
DROP TABLE creatives;
DROP TABLE campaigns;
