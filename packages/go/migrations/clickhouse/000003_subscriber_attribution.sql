ALTER TABLE push_booster.subscribers
    ADD COLUMN IF NOT EXISTS subid String AFTER user_agent,
    ADD COLUMN IF NOT EXISTS channel String AFTER subid,
    ADD COLUMN IF NOT EXISTS landing_url String AFTER channel,
    ADD COLUMN IF NOT EXISTS referrer String AFTER landing_url;
