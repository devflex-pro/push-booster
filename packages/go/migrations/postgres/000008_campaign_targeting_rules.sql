-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN targeting_rules JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_targeting_rules_object_check CHECK (jsonb_typeof(targeting_rules) = 'object');

-- +goose Down
ALTER TABLE campaigns
    DROP CONSTRAINT campaigns_targeting_rules_object_check,
    DROP COLUMN targeting_rules;
