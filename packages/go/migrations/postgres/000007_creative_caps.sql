-- +goose Up
ALTER TABLE creatives
    ADD COLUMN daily_cap_per_subscription INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN total_cap_per_subscription INTEGER NOT NULL DEFAULT 0;

ALTER TABLE creatives
    ADD CONSTRAINT creatives_daily_cap_per_subscription_check CHECK (daily_cap_per_subscription >= 0),
    ADD CONSTRAINT creatives_total_cap_per_subscription_check CHECK (total_cap_per_subscription >= 0);

-- +goose Down
ALTER TABLE creatives
    DROP CONSTRAINT creatives_total_cap_per_subscription_check,
    DROP CONSTRAINT creatives_daily_cap_per_subscription_check,
    DROP COLUMN total_cap_per_subscription,
    DROP COLUMN daily_cap_per_subscription;
