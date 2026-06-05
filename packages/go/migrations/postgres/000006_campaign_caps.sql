-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN daily_cap_per_subscription INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN total_cap_per_subscription INTEGER NOT NULL DEFAULT 0;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_daily_cap_per_subscription_check CHECK (daily_cap_per_subscription >= 0),
    ADD CONSTRAINT campaigns_total_cap_per_subscription_check CHECK (total_cap_per_subscription >= 0);

-- +goose Down
ALTER TABLE campaigns
    DROP CONSTRAINT campaigns_total_cap_per_subscription_check,
    DROP CONSTRAINT campaigns_daily_cap_per_subscription_check,
    DROP COLUMN total_cap_per_subscription,
    DROP COLUMN daily_cap_per_subscription;
