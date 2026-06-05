-- +goose Up
CREATE TABLE campaign_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    timezone_mode TEXT NOT NULL DEFAULT 'subscriber_local',
    fallback_timezone TEXT NOT NULL DEFAULT 'UTC',
    grace_minutes INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaign_schedules_status_check CHECK (status IN ('active', 'paused', 'archived')),
    CONSTRAINT campaign_schedules_timezone_mode_check CHECK (timezone_mode IN ('subscriber_local')),
    CONSTRAINT campaign_schedules_grace_minutes_check CHECK (grace_minutes >= 1 AND grace_minutes <= 120)
);

CREATE TABLE campaign_schedule_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES campaign_schedules(id) ON DELETE CASCADE,
    local_time TIME NOT NULL,
    days_of_week INT[] NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaign_schedule_slots_days_check CHECK (
        cardinality(days_of_week) > 0
        AND days_of_week <@ ARRAY[1,2,3,4,5,6,7]
    )
);

CREATE TABLE campaign_schedule_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES campaign_schedules(id) ON DELETE CASCADE,
    slot_id UUID NOT NULL REFERENCES campaign_schedule_slots(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    launch_id UUID REFERENCES campaign_launches(id) ON DELETE SET NULL,
    local_date DATE NOT NULL,
    local_time TIME NOT NULL,
    timezone TEXT NOT NULL,
    scheduled_utc_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    audience_total BIGINT NOT NULL DEFAULT 0,
    enqueued_total BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT campaign_schedule_runs_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

CREATE UNIQUE INDEX campaign_schedule_runs_dedupe_idx
    ON campaign_schedule_runs(schedule_id, slot_id, local_date, timezone);
CREATE INDEX campaign_schedules_campaign_idx ON campaign_schedules(campaign_id, status);
CREATE INDEX campaign_schedule_slots_schedule_idx ON campaign_schedule_slots(schedule_id, position);
CREATE INDEX campaign_schedule_runs_schedule_created_idx ON campaign_schedule_runs(schedule_id, created_at DESC);

-- +goose Down
DROP INDEX campaign_schedule_runs_schedule_created_idx;
DROP INDEX campaign_schedule_slots_schedule_idx;
DROP INDEX campaign_schedules_campaign_idx;
DROP INDEX campaign_schedule_runs_dedupe_idx;
DROP TABLE campaign_schedule_runs;
DROP TABLE campaign_schedule_slots;
DROP TABLE campaign_schedules;
