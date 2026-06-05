-- +goose Up
CREATE TABLE cost_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cost_date DATE NOT NULL,
    publisher_id UUID REFERENCES publishers(id),
    source_id UUID REFERENCES sources(id),
    campaign_id UUID REFERENCES campaigns(id),
    creative_id UUID REFERENCES creatives(id),
    amount DOUBLE PRECISION NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cost_entries_date_idx ON cost_entries(cost_date);
CREATE INDEX cost_entries_source_date_idx ON cost_entries(source_id, cost_date);
CREATE INDEX cost_entries_campaign_date_idx ON cost_entries(campaign_id, cost_date);
CREATE INDEX cost_entries_creative_date_idx ON cost_entries(creative_id, cost_date);

-- +goose Down
DROP TABLE cost_entries;
