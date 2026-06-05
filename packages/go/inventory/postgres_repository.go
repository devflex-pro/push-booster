package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreatePublisher(
	ctx context.Context,
	input CreatePublisherInput,
) (Publisher, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO publishers (name, status)
VALUES ($1, $2)
RETURNING id::text, name, status, created_at, updated_at
`,
		input.Name,
		StatusActive,
	)
	publisher, err := scanPublisher(row)
	if err != nil {
		return Publisher{}, fmt.Errorf("create publisher: %w", err)
	}
	return publisher, nil
}

func (r *PostgresRepository) ListPublishers(ctx context.Context) ([]Publisher, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text, name, status, created_at, updated_at
FROM publishers
ORDER BY created_at DESC
`,
	)
	if err != nil {
		return nil, fmt.Errorf("list publishers: %w", err)
	}
	defer rows.Close()

	publishers := []Publisher{}
	for rows.Next() {
		publisher, err := scanPublisher(rows)
		if err != nil {
			return nil, fmt.Errorf("scan publisher: %w", err)
		}
		publishers = append(publishers, publisher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list publishers rows: %w", err)
	}
	return publishers, nil
}

func (r *PostgresRepository) CreateSource(
	ctx context.Context,
	input CreateSourceInput,
) (Source, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO sources (publisher_id, name, domain, status)
VALUES ($1::uuid, $2, $3, $4)
RETURNING id::text, publisher_id::text, name, domain, status, COALESCE(vapid_key_id::text, ''), created_at, updated_at
`,
		input.PublisherID,
		input.Name,
		input.Domain,
		StatusActive,
	)
	source, err := scanSource(row)
	if err != nil {
		return Source{}, fmt.Errorf("create source: %w", err)
	}
	return source, nil
}

func (r *PostgresRepository) ListSources(ctx context.Context, publisherID string) ([]Source, error) {
	query := `
SELECT id::text, publisher_id::text, name, domain, status, COALESCE(vapid_key_id::text, ''), created_at, updated_at
FROM sources
`
	args := []any{}
	if publisherID != "" {
		query += "WHERE publisher_id = $1::uuid\n"
		args = append(args, publisherID)
	}
	query += "ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	sources := []Source{}
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sources rows: %w", err)
	}
	return sources, nil
}

func (r *PostgresRepository) GetSource(ctx context.Context, id string) (Source, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT id::text, publisher_id::text, name, domain, status, COALESCE(vapid_key_id::text, ''), created_at, updated_at
FROM sources
WHERE id = $1::uuid
LIMIT 1
`,
		id,
	)
	source, err := scanSource(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Source{}, ErrNotFound
		}
		return Source{}, fmt.Errorf("get source: %w", err)
	}
	return source, nil
}

func (r *PostgresRepository) CreateVAPIDKey(
	ctx context.Context,
	key VAPIDKey,
) (VAPIDKey, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO vapid_keys (name, public_key, private_key, status)
VALUES ($1, $2, $3, $4)
RETURNING id::text, name, public_key, private_key, status, created_at, updated_at
`,
		key.Name,
		key.PublicKey,
		key.PrivateKey,
		key.Status,
	)
	vapidKey, err := scanVAPIDKey(row)
	if err != nil {
		return VAPIDKey{}, fmt.Errorf("create vapid key: %w", err)
	}
	return vapidKey, nil
}

func (r *PostgresRepository) ListVAPIDKeys(ctx context.Context) ([]VAPIDKey, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text, name, public_key, '' AS private_key, status, created_at, updated_at
FROM vapid_keys
ORDER BY created_at DESC
`,
	)
	if err != nil {
		return nil, fmt.Errorf("list vapid keys: %w", err)
	}
	defer rows.Close()

	keys := []VAPIDKey{}
	for rows.Next() {
		key, err := scanVAPIDKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan vapid key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list vapid keys rows: %w", err)
	}
	return keys, nil
}

func (r *PostgresRepository) UpdateVAPIDKeyStatus(
	ctx context.Context,
	id string,
	status string,
) (VAPIDKey, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE vapid_keys
SET status = $2, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, name, public_key, '' AS private_key, status, created_at, updated_at
`,
		id,
		status,
	)
	key, err := scanVAPIDKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VAPIDKey{}, ErrNotFound
		}
		return VAPIDKey{}, fmt.Errorf("update vapid key status: %w", err)
	}
	return key, nil
}

func (r *PostgresRepository) AttachVAPIDKeyToSource(
	ctx context.Context,
	input AttachVAPIDKeyInput,
) (Source, error) {
	var keyStatus string
	if err := r.pool.QueryRow(
		ctx,
		`
SELECT status
FROM vapid_keys
WHERE id = $1::uuid
LIMIT 1
`,
		input.VAPIDKeyID,
	).Scan(&keyStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Source{}, ErrNotFound
		}
		return Source{}, fmt.Errorf("get vapid key before attach: %w", err)
	}
	if keyStatus != VAPIDStatusActive {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("vapid key must be active"))
	}

	row := r.pool.QueryRow(
		ctx,
		`
UPDATE sources
SET vapid_key_id = $2::uuid, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, publisher_id::text, name, domain, status, COALESCE(vapid_key_id::text, ''), created_at, updated_at
`,
		input.SourceID,
		input.VAPIDKeyID,
	)
	source, err := scanSource(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Source{}, ErrNotFound
		}
		return Source{}, fmt.Errorf("attach vapid key to source: %w", err)
	}
	return source, nil
}

func (r *PostgresRepository) ActiveVAPIDKeyForSource(ctx context.Context, sourceID string) (VAPIDKey, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT vk.id::text, vk.name, vk.public_key, vk.private_key, vk.status, vk.created_at, vk.updated_at
FROM sources s
JOIN vapid_keys vk ON vk.id = s.vapid_key_id
WHERE s.id = $1::uuid
  AND s.status = $2
  AND vk.status = $3
LIMIT 1
`,
		sourceID,
		StatusActive,
		VAPIDStatusActive,
	)
	key, err := scanVAPIDKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VAPIDKey{}, ErrNotFound
		}
		return VAPIDKey{}, fmt.Errorf("get active vapid key for source: %w", err)
	}
	return key, nil
}

func (r *PostgresRepository) CreateCampaign(
	ctx context.Context,
	input CreateCampaignInput,
) (Campaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, fmt.Errorf("begin create campaign: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var campaignID string
	if input.AudienceScope == CampaignAudienceScopeAll {
		err = tx.QueryRow(
			ctx,
			`
INSERT INTO campaigns (
    publisher_id,
    source_id,
    name,
    status,
    audience_scope,
    targeting_rules,
    daily_cap_per_subscription,
    total_cap_per_subscription
)
VALUES (NULL, NULL, $1, $2, $3, $4::jsonb, $5, $6)
RETURNING id::text
`,
			input.Name,
			CampaignStatusDraft,
			input.AudienceScope,
			targetingRulesJSON(input.TargetingRules),
			input.DailyCapPerSubscription,
			input.TotalCapPerSubscription,
		).Scan(&campaignID)
	} else {
		err = tx.QueryRow(
			ctx,
			`
INSERT INTO campaigns (
    publisher_id,
    source_id,
    name,
    status,
    audience_scope,
    targeting_rules,
    daily_cap_per_subscription,
    total_cap_per_subscription
)
SELECT s.publisher_id, s.id, $2, $3, $4, $5::jsonb, $6, $7
FROM sources s
WHERE s.id = $1::uuid
RETURNING id::text
`,
			input.SourceID,
			input.Name,
			CampaignStatusDraft,
			input.AudienceScope,
			targetingRulesJSON(input.TargetingRules),
			input.DailyCapPerSubscription,
			input.TotalCapPerSubscription,
		).Scan(&campaignID)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Campaign{}, ErrNotFound
		}
		return Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	if input.AudienceScope == CampaignAudienceScopeSelectedSources {
		commandTag, err := tx.Exec(
			ctx,
			`
INSERT INTO campaign_sources (campaign_id, source_id)
SELECT $1::uuid, source_id
FROM unnest($2::uuid[]) AS selected(source_id)
JOIN sources s ON s.id = selected.source_id
ON CONFLICT DO NOTHING
`,
			campaignID,
			input.SourceIDs,
		)
		if err != nil {
			return Campaign{}, fmt.Errorf("create campaign sources: %w", err)
		}
		if int(commandTag.RowsAffected()) != len(input.SourceIDs) {
			_ = tx.Rollback(ctx)
			return Campaign{}, ErrNotFound
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, fmt.Errorf("commit create campaign: %w", err)
	}
	return r.GetCampaign(ctx, campaignID)
}

func (r *PostgresRepository) ListCampaigns(ctx context.Context, sourceID string) ([]Campaign, error) {
	query := `
SELECT ca.id::text,
       COALESCE(ca.publisher_id::text, ''),
       COALESCE(ca.source_id::text, ''),
       COALESCE(array_remove(array_agg(cs.source_id::text ORDER BY cs.created_at), NULL), ARRAY[]::text[]),
       ca.audience_scope,
       ca.name,
       ca.status,
       ca.targeting_rules::text,
       ca.daily_cap_per_subscription,
       ca.total_cap_per_subscription,
       ca.created_at,
       ca.updated_at
FROM campaigns ca
LEFT JOIN campaign_sources cs ON cs.campaign_id = ca.id
`
	args := []any{}
	if sourceID != "" {
		query += `WHERE ca.audience_scope = 'all'
   OR ca.source_id = $1::uuid
   OR EXISTS (
       SELECT 1 FROM campaign_sources selected
       WHERE selected.campaign_id = ca.id AND selected.source_id = $1::uuid
   )
`
		args = append(args, sourceID)
	}
	query += `GROUP BY ca.id
ORDER BY ca.created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := []Campaign{}
	for rows.Next() {
		campaign, err := scanCampaign(rows)
		if err != nil {
			return nil, fmt.Errorf("scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list campaigns rows: %w", err)
	}
	return campaigns, nil
}

func (r *PostgresRepository) GetCampaign(ctx context.Context, id string) (Campaign, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT ca.id::text,
       COALESCE(ca.publisher_id::text, ''),
       COALESCE(ca.source_id::text, ''),
       COALESCE(array_remove(array_agg(cs.source_id::text ORDER BY cs.created_at), NULL), ARRAY[]::text[]),
       ca.audience_scope,
       ca.name,
       ca.status,
       ca.targeting_rules::text,
       ca.daily_cap_per_subscription,
       ca.total_cap_per_subscription,
       ca.created_at,
       ca.updated_at
FROM campaigns ca
LEFT JOIN campaign_sources cs ON cs.campaign_id = ca.id
WHERE ca.id = $1::uuid
GROUP BY ca.id
`,
		id,
	)
	campaign, err := scanCampaign(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Campaign{}, ErrNotFound
		}
		return Campaign{}, fmt.Errorf("get campaign: %w", err)
	}
	return campaign, nil
}

func (r *PostgresRepository) UpdateCampaignStatus(
	ctx context.Context,
	input UpdateCampaignStatusInput,
) (Campaign, error) {
	commandTag, err := r.pool.Exec(
		ctx,
		`
UPDATE campaigns
SET status = $2, updated_at = now()
WHERE id = $1::uuid
`,
		input.ID,
		input.Status,
	)
	if err != nil {
		return Campaign{}, fmt.Errorf("update campaign status: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return Campaign{}, ErrNotFound
	}
	return r.GetCampaign(ctx, input.ID)
}

func (r *PostgresRepository) CreateCampaignLaunch(
	ctx context.Context,
	input CreateCampaignLaunchInput,
) (CampaignLaunch, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO campaign_launches (campaign_id, status)
VALUES ($1::uuid, $2)
RETURNING id::text,
          campaign_id::text,
          status,
          audience_total,
          processed_total,
          error_message,
          enqueue_status,
          enqueued_total,
          enqueue_error,
          created_at,
          updated_at,
          completed_at,
          enqueue_started_at,
          enqueue_completed_at
`,
		input.CampaignID,
		CampaignLaunchStatusBuilding,
	)
	launch, err := scanCampaignLaunch(row)
	if err != nil {
		return CampaignLaunch{}, fmt.Errorf("create campaign launch: %w", err)
	}
	return launch, nil
}

func (r *PostgresRepository) ListCampaignLaunches(
	ctx context.Context,
	campaignID string,
) ([]CampaignLaunch, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       campaign_id::text,
       status,
       audience_total,
       processed_total,
       error_message,
       enqueue_status,
       enqueued_total,
       enqueue_error,
       created_at,
       updated_at,
       completed_at,
       enqueue_started_at,
       enqueue_completed_at
FROM campaign_launches
WHERE campaign_id = $1::uuid
ORDER BY created_at DESC
LIMIT 20
`,
		campaignID,
	)
	if err != nil {
		return nil, fmt.Errorf("list campaign launches: %w", err)
	}
	defer rows.Close()

	launches := []CampaignLaunch{}
	for rows.Next() {
		launch, err := scanCampaignLaunch(rows)
		if err != nil {
			return nil, fmt.Errorf("scan campaign launch: %w", err)
		}
		launches = append(launches, launch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list campaign launches rows: %w", err)
	}
	return launches, nil
}

func (r *PostgresRepository) GetCampaignLaunch(ctx context.Context, id string) (CampaignLaunch, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT id::text,
       campaign_id::text,
       status,
       audience_total,
       processed_total,
       error_message,
       enqueue_status,
       enqueued_total,
       enqueue_error,
       created_at,
       updated_at,
       completed_at,
       enqueue_started_at,
       enqueue_completed_at
FROM campaign_launches
WHERE id = $1::uuid
`,
		id,
	)
	launch, err := scanCampaignLaunch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignLaunch{}, ErrNotFound
		}
		return CampaignLaunch{}, fmt.Errorf("get campaign launch: %w", err)
	}
	return launch, nil
}

func (r *PostgresRepository) UpdateCampaignLaunch(
	ctx context.Context,
	input UpdateCampaignLaunchInput,
) (CampaignLaunch, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE campaign_launches
SET status = $2,
    audience_total = $3,
    processed_total = $4,
    error_message = $5,
    updated_at = now(),
    completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN now() ELSE completed_at END
WHERE id = $1::uuid
RETURNING id::text,
          campaign_id::text,
          status,
          audience_total,
          processed_total,
          error_message,
          enqueue_status,
          enqueued_total,
          enqueue_error,
          created_at,
          updated_at,
          completed_at,
          enqueue_started_at,
          enqueue_completed_at
`,
		input.ID,
		input.Status,
		input.AudienceTotal,
		input.ProcessedTotal,
		input.ErrorMessage,
	)
	launch, err := scanCampaignLaunch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignLaunch{}, ErrNotFound
		}
		return CampaignLaunch{}, fmt.Errorf("update campaign launch: %w", err)
	}
	return launch, nil
}

func (r *PostgresRepository) UpdateCampaignLaunchEnqueue(
	ctx context.Context,
	input UpdateCampaignLaunchEnqueueInput,
) (CampaignLaunch, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE campaign_launches
SET enqueue_status = $2,
    enqueued_total = $3,
    enqueue_error = $4,
    updated_at = now(),
    enqueue_started_at = CASE WHEN $2 = 'enqueuing' THEN now() ELSE enqueue_started_at END,
    enqueue_completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN now() ELSE enqueue_completed_at END
WHERE id = $1::uuid
RETURNING id::text,
          campaign_id::text,
          status,
          audience_total,
          processed_total,
          error_message,
          enqueue_status,
          enqueued_total,
          enqueue_error,
          created_at,
          updated_at,
          completed_at,
          enqueue_started_at,
          enqueue_completed_at
`,
		input.ID,
		input.EnqueueStatus,
		input.EnqueuedTotal,
		input.EnqueueError,
	)
	launch, err := scanCampaignLaunch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignLaunch{}, ErrNotFound
		}
		return CampaignLaunch{}, fmt.Errorf("update campaign launch enqueue: %w", err)
	}
	return launch, nil
}

func (r *PostgresRepository) ListCampaignSchedules(
	ctx context.Context,
	campaignID string,
) ([]CampaignSchedule, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text, campaign_id::text, status, timezone_mode, fallback_timezone,
       grace_minutes, created_at, updated_at
FROM campaign_schedules
WHERE campaign_id = $1::uuid
ORDER BY created_at DESC
`,
		campaignID,
	)
	if err != nil {
		return nil, fmt.Errorf("list campaign schedules: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(ctx, rows)
}

func (r *PostgresRepository) ListActiveCampaignSchedules(ctx context.Context) ([]CampaignSchedule, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text, campaign_id::text, status, timezone_mode, fallback_timezone,
       grace_minutes, created_at, updated_at
FROM campaign_schedules
WHERE status = $1
ORDER BY created_at ASC
`,
		ScheduleStatusActive,
	)
	if err != nil {
		return nil, fmt.Errorf("list active campaign schedules: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(ctx, rows)
}

func (r *PostgresRepository) CreateCampaignSchedule(
	ctx context.Context,
	input CreateCampaignScheduleInput,
) (CampaignSchedule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return CampaignSchedule{}, fmt.Errorf("begin create campaign schedule: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	var id string
	err = tx.QueryRow(
		ctx,
		`
INSERT INTO campaign_schedules (
    campaign_id, status, timezone_mode, fallback_timezone, grace_minutes
)
VALUES ($1::uuid, $2, 'subscriber_local', $3, $4)
RETURNING id::text
`,
		input.CampaignID,
		input.Status,
		input.FallbackTimezone,
		input.GraceMinutes,
	).Scan(&id)
	if err != nil {
		return CampaignSchedule{}, fmt.Errorf("create campaign schedule: %w", err)
	}
	for _, slot := range input.Slots {
		if _, err = tx.Exec(
			ctx,
			`
INSERT INTO campaign_schedule_slots (schedule_id, local_time, days_of_week, position)
VALUES ($1::uuid, $2::time, $3, $4)
`,
			id,
			slot.LocalTime,
			slot.DaysOfWeek,
			slot.Position,
		); err != nil {
			return CampaignSchedule{}, fmt.Errorf("create campaign schedule slot: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return CampaignSchedule{}, fmt.Errorf("commit create campaign schedule: %w", err)
	}
	schedules, err := r.ListCampaignSchedules(ctx, input.CampaignID)
	if err != nil {
		return CampaignSchedule{}, err
	}
	for _, schedule := range schedules {
		if schedule.ID == id {
			return schedule, nil
		}
	}
	return CampaignSchedule{}, ErrNotFound
}

func (r *PostgresRepository) UpdateCampaignScheduleStatus(
	ctx context.Context,
	input UpdateCampaignScheduleStatusInput,
) (CampaignSchedule, error) {
	var campaignID string
	err := r.pool.QueryRow(
		ctx,
		`
UPDATE campaign_schedules
SET status = $2, updated_at = now()
WHERE id = $1::uuid
RETURNING campaign_id::text
`,
		input.ID,
		input.Status,
	).Scan(&campaignID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignSchedule{}, ErrNotFound
		}
		return CampaignSchedule{}, fmt.Errorf("update campaign schedule status: %w", err)
	}
	schedules, err := r.ListCampaignSchedules(ctx, campaignID)
	if err != nil {
		return CampaignSchedule{}, err
	}
	for _, schedule := range schedules {
		if schedule.ID == input.ID {
			return schedule, nil
		}
	}
	return CampaignSchedule{}, ErrNotFound
}

func (r *PostgresRepository) CreateScheduleRun(
	ctx context.Context,
	input CreateScheduleRunInput,
) (CampaignScheduleRun, bool, error) {
	var id string
	err := r.pool.QueryRow(
		ctx,
		`
INSERT INTO campaign_schedule_runs (
    schedule_id, slot_id, campaign_id, local_date, local_time,
    timezone, scheduled_utc_at, status
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::date, $5::time, $6, $7, $8)
ON CONFLICT (schedule_id, slot_id, local_date, timezone) DO NOTHING
RETURNING id::text
`,
		input.ScheduleID,
		input.SlotID,
		input.CampaignID,
		input.LocalDate,
		input.LocalTime,
		input.Timezone,
		input.ScheduledUTCAt,
		ScheduleRunStatusPending,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignScheduleRun{}, false, nil
		}
		return CampaignScheduleRun{}, false, fmt.Errorf("create schedule run: %w", err)
	}
	run, err := r.getScheduleRun(ctx, id)
	return run, true, err
}

func (r *PostgresRepository) CompleteScheduleRun(
	ctx context.Context,
	input CompleteScheduleRunInput,
) (CampaignScheduleRun, error) {
	var id string
	err := r.pool.QueryRow(
		ctx,
		`
UPDATE campaign_schedule_runs
SET launch_id = NULLIF($2, '')::uuid,
    status = $3,
    audience_total = $4,
    enqueued_total = $5,
    error_message = $6,
    updated_at = now(),
    completed_at = now()
WHERE id = $1::uuid
RETURNING id::text
`,
		input.ID,
		input.LaunchID,
		input.Status,
		input.AudienceTotal,
		input.EnqueuedTotal,
		input.ErrorMessage,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignScheduleRun{}, ErrNotFound
		}
		return CampaignScheduleRun{}, fmt.Errorf("complete schedule run: %w", err)
	}
	return r.getScheduleRun(ctx, id)
}

func (r *PostgresRepository) ListCampaignScheduleRuns(
	ctx context.Context,
	campaignID string,
) ([]CampaignScheduleRun, error) {
	rows, err := r.pool.Query(
		ctx,
		scheduleRunSelectSQL()+`
WHERE campaign_id = $1::uuid
ORDER BY scheduled_utc_at DESC
LIMIT 50
`,
		campaignID,
	)
	if err != nil {
		return nil, fmt.Errorf("list campaign schedule runs: %w", err)
	}
	defer rows.Close()
	return scanScheduleRuns(rows)
}

func (r *PostgresRepository) CreateCreative(
	ctx context.Context,
	input CreateCreativeInput,
) (Creative, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO creatives (
    campaign_id,
    title,
    body,
    url,
    icon,
    status,
    daily_cap_per_subscription,
    total_cap_per_subscription
)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
RETURNING id::text,
          campaign_id::text,
          title,
          body,
          url,
          icon,
          status,
          source_type,
          COALESCE(provider_config_id::text, ''),
          provider_name,
          provider_external_id,
          last_synced_at,
          sync_status,
          daily_cap_per_subscription,
          total_cap_per_subscription,
          0 AS campaign_daily_cap_per_subscription,
          0 AS campaign_total_cap_per_subscription,
          '{}' AS campaign_targeting_rules,
          created_at,
          updated_at
`,
		input.CampaignID,
		input.Title,
		input.Body,
		input.URL,
		input.Icon,
		CreativeStatusActive,
		input.DailyCapPerSubscription,
		input.TotalCapPerSubscription,
	)
	creative, err := scanCreative(row)
	if err != nil {
		return Creative{}, fmt.Errorf("create creative: %w", err)
	}
	return creative, nil
}

func (r *PostgresRepository) ListCreatives(ctx context.Context, campaignID string) ([]Creative, error) {
	query := `
SELECT id::text,
       campaign_id::text,
       title,
       body,
       url,
       icon,
       status,
       source_type,
       COALESCE(provider_config_id::text, ''),
       provider_name,
       provider_external_id,
       last_synced_at,
       sync_status,
       daily_cap_per_subscription,
       total_cap_per_subscription,
       0 AS campaign_daily_cap_per_subscription,
       0 AS campaign_total_cap_per_subscription,
       '{}' AS campaign_targeting_rules,
       created_at,
       updated_at
FROM creatives
`
	args := []any{}
	if campaignID != "" {
		query += "WHERE campaign_id = $1::uuid\n"
		args = append(args, campaignID)
	}
	query += "ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list creatives: %w", err)
	}
	defer rows.Close()

	creatives := []Creative{}
	for rows.Next() {
		creative, err := scanCreative(rows)
		if err != nil {
			return nil, fmt.Errorf("scan creative: %w", err)
		}
		creatives = append(creatives, creative)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list creatives rows: %w", err)
	}
	return creatives, nil
}

func (r *PostgresRepository) UpdateCreativeStatus(
	ctx context.Context,
	input UpdateCreativeStatusInput,
) (Creative, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE creatives
SET status = $2, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text,
          campaign_id::text,
          title,
          body,
          url,
          icon,
          status,
          source_type,
          COALESCE(provider_config_id::text, ''),
          provider_name,
          provider_external_id,
          last_synced_at,
          sync_status,
          daily_cap_per_subscription,
          total_cap_per_subscription,
          0 AS campaign_daily_cap_per_subscription,
          0 AS campaign_total_cap_per_subscription,
          '{}' AS campaign_targeting_rules,
          created_at,
          updated_at
`,
		input.ID,
		input.Status,
	)
	creative, err := scanCreative(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Creative{}, ErrNotFound
		}
		return Creative{}, fmt.Errorf("update creative status: %w", err)
	}
	return creative, nil
}

func (r *PostgresRepository) CreateCreativeProviderConfig(
	ctx context.Context,
	input CreateCreativeProviderConfigInput,
) (CreativeProviderConfig, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO creative_provider_configs (
    campaign_id,
    name,
    provider_name,
    fetch_url,
    request_headers,
    status
)
VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6)
RETURNING id::text,
          campaign_id::text,
          name,
          provider_name,
          fetch_url,
          request_headers::text,
          status,
          last_sync_at,
          created_at,
          updated_at
`,
		input.CampaignID,
		input.Name,
		input.ProviderName,
		input.FetchURL,
		headersJSON(input.RequestHeaders),
		ProviderConfigStatusActive,
	)
	config, err := scanCreativeProviderConfig(row)
	if err != nil {
		return CreativeProviderConfig{}, fmt.Errorf("create creative provider config: %w", err)
	}
	return config, nil
}

func (r *PostgresRepository) ListCreativeProviderConfigs(
	ctx context.Context,
	campaignID string,
) ([]CreativeProviderConfig, error) {
	query := `
SELECT id::text,
       campaign_id::text,
       name,
       provider_name,
       fetch_url,
       request_headers::text,
       status,
       last_sync_at,
       created_at,
       updated_at
FROM creative_provider_configs
`
	args := []any{}
	if campaignID != "" {
		query += "WHERE campaign_id = $1::uuid\n"
		args = append(args, campaignID)
	}
	query += "ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list creative provider configs: %w", err)
	}
	defer rows.Close()

	configs := []CreativeProviderConfig{}
	for rows.Next() {
		config, err := scanCreativeProviderConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("scan creative provider config: %w", err)
		}
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("creative provider config rows: %w", err)
	}
	return configs, nil
}

func (r *PostgresRepository) GetCreativeProviderConfig(
	ctx context.Context,
	id string,
) (CreativeProviderConfig, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT id::text,
       campaign_id::text,
       name,
       provider_name,
       fetch_url,
       request_headers::text,
       status,
       last_sync_at,
       created_at,
       updated_at
FROM creative_provider_configs
WHERE id = $1::uuid
`,
		id,
	)
	config, err := scanCreativeProviderConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CreativeProviderConfig{}, ErrNotFound
		}
		return CreativeProviderConfig{}, fmt.Errorf("get creative provider config: %w", err)
	}
	return config, nil
}

func (r *PostgresRepository) UpdateCreativeProviderConfigStatus(
	ctx context.Context,
	input UpdateCreativeProviderConfigStatusInput,
) (CreativeProviderConfig, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE creative_provider_configs
SET status = $2, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text,
          campaign_id::text,
          name,
          provider_name,
          fetch_url,
          request_headers::text,
          status,
          last_sync_at,
          created_at,
          updated_at
`,
		input.ID,
		input.Status,
	)
	config, err := scanCreativeProviderConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CreativeProviderConfig{}, ErrNotFound
		}
		return CreativeProviderConfig{}, fmt.Errorf("update creative provider config status: %w", err)
	}
	return config, nil
}

func (r *PostgresRepository) CreateCreativeSyncLog(
	ctx context.Context,
	providerConfigID string,
	campaignID string,
) (CreativeSyncLog, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO creative_sync_logs (provider_config_id, campaign_id, status)
VALUES ($1::uuid, $2::uuid, $3)
RETURNING id::text,
          provider_config_id::text,
          campaign_id::text,
          status,
          fetched_total,
          upserted_total,
          error_message,
          started_at,
          completed_at
`,
		providerConfigID,
		campaignID,
		CreativeSyncLogStatusRunning,
	)
	log, err := scanCreativeSyncLog(row)
	if err != nil {
		return CreativeSyncLog{}, fmt.Errorf("create creative sync log: %w", err)
	}
	return log, nil
}

func (r *PostgresRepository) ListCreativeSyncLogs(
	ctx context.Context,
	providerConfigID string,
	campaignID string,
) ([]CreativeSyncLog, error) {
	query := `
SELECT id::text,
       provider_config_id::text,
       campaign_id::text,
       status,
       fetched_total,
       upserted_total,
       error_message,
       started_at,
       completed_at
FROM creative_sync_logs
`
	args := []any{}
	conditions := []string{}
	if providerConfigID != "" {
		args = append(args, providerConfigID)
		conditions = append(conditions, fmt.Sprintf("provider_config_id = $%d::uuid", len(args)))
	}
	if campaignID != "" {
		args = append(args, campaignID)
		conditions = append(conditions, fmt.Sprintf("campaign_id = $%d::uuid", len(args)))
	}
	if len(conditions) > 0 {
		query += "WHERE " + strings.Join(conditions, " AND ") + "\n"
	}
	query += "ORDER BY started_at DESC LIMIT 20"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list creative sync logs: %w", err)
	}
	defer rows.Close()

	logs := []CreativeSyncLog{}
	for rows.Next() {
		log, err := scanCreativeSyncLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan creative sync log: %w", err)
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("creative sync log rows: %w", err)
	}
	return logs, nil
}

func (r *PostgresRepository) CompleteCreativeSyncLog(
	ctx context.Context,
	input CompleteCreativeSyncLogInput,
) (CreativeSyncLog, error) {
	row := r.pool.QueryRow(
		ctx,
		`
WITH updated_log AS (
    UPDATE creative_sync_logs
    SET status = $2,
        fetched_total = $3,
        upserted_total = $4,
        error_message = $5,
        completed_at = now()
    WHERE id = $1::uuid
    RETURNING id,
              provider_config_id,
              campaign_id,
              status,
              fetched_total,
              upserted_total,
              error_message,
              started_at,
              completed_at
),
updated_config AS (
    UPDATE creative_provider_configs c
    SET last_sync_at = CASE WHEN $2 = 'completed' THEN now() ELSE c.last_sync_at END,
        updated_at = now()
    FROM updated_log l
    WHERE c.id = l.provider_config_id
    RETURNING c.id
)
SELECT id::text,
       provider_config_id::text,
       campaign_id::text,
       status,
       fetched_total,
       upserted_total,
       error_message,
       started_at,
       completed_at
FROM updated_log
`,
		input.ID,
		input.Status,
		input.FetchedTotal,
		input.UpsertedTotal,
		input.ErrorMessage,
	)
	log, err := scanCreativeSyncLog(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CreativeSyncLog{}, ErrNotFound
		}
		return CreativeSyncLog{}, fmt.Errorf("complete creative sync log: %w", err)
	}
	return log, nil
}

func (r *PostgresRepository) UpsertProviderCreative(
	ctx context.Context,
	input UpsertProviderCreativeInput,
) (Creative, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO creatives (
    campaign_id,
    title,
    body,
    url,
    icon,
    status,
    source_type,
    provider_config_id,
    provider_name,
    provider_external_id,
    raw_provider_payload,
    last_synced_at,
    sync_status,
    daily_cap_per_subscription,
    total_cap_per_subscription
)
VALUES (
    $1::uuid,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8::uuid,
    $9,
    $10,
    $11::jsonb,
    now(),
    $12,
    $13,
    $14
)
ON CONFLICT (campaign_id, provider_name, provider_external_id)
WHERE source_type = 'provider_api' AND provider_external_id <> ''
DO UPDATE SET
    provider_config_id = EXCLUDED.provider_config_id,
    title = EXCLUDED.title,
    body = EXCLUDED.body,
    url = EXCLUDED.url,
    icon = EXCLUDED.icon,
    status = EXCLUDED.status,
    raw_provider_payload = EXCLUDED.raw_provider_payload,
    last_synced_at = now(),
    sync_status = EXCLUDED.sync_status,
    daily_cap_per_subscription = EXCLUDED.daily_cap_per_subscription,
    total_cap_per_subscription = EXCLUDED.total_cap_per_subscription,
    updated_at = now()
RETURNING id::text,
          campaign_id::text,
          title,
          body,
          url,
          icon,
          status,
          source_type,
          COALESCE(provider_config_id::text, ''),
          provider_name,
          provider_external_id,
          last_synced_at,
          sync_status,
          daily_cap_per_subscription,
          total_cap_per_subscription,
          0 AS campaign_daily_cap_per_subscription,
          0 AS campaign_total_cap_per_subscription,
          '{}' AS campaign_targeting_rules,
          created_at,
          updated_at
`,
		input.CampaignID,
		input.Title,
		input.Body,
		input.URL,
		input.Icon,
		input.Status,
		CreativeSourceProviderAPI,
		input.ProviderConfigID,
		input.ProviderName,
		input.ProviderExternalID,
		input.RawProviderPayload,
		CreativeSyncStatusSynced,
		input.DailyCapPerSubscription,
		input.TotalCapPerSubscription,
	)
	creative, err := scanCreative(row)
	if err != nil {
		return Creative{}, fmt.Errorf("upsert provider creative: %w", err)
	}
	return creative, nil
}

func (r *PostgresRepository) MarkMissingProviderCreativesStale(
	ctx context.Context,
	providerConfigID string,
	externalIDs []string,
) (int64, error) {
	commandTag, err := r.pool.Exec(
		ctx,
		`
UPDATE creatives
SET sync_status = $2, updated_at = now()
WHERE provider_config_id = $1::uuid
  AND source_type = 'provider_api'
  AND NOT (provider_external_id = ANY($3::text[]))
`,
		providerConfigID,
		CreativeSyncStatusStale,
		externalIDs,
	)
	if err != nil {
		return 0, fmt.Errorf("mark missing provider creatives stale: %w", err)
	}
	return commandTag.RowsAffected(), nil
}

func (r *PostgresRepository) TryAcquireCreativeProviderSyncLock(
	ctx context.Context,
	providerConfigID string,
) (bool, error) {
	var locked bool
	if err := r.pool.QueryRow(
		ctx,
		`SELECT pg_try_advisory_lock(hashtext('creative_provider_sync'), hashtext($1))`,
		providerConfigID,
	).Scan(&locked); err != nil {
		return false, fmt.Errorf("acquire creative provider sync lock: %w", err)
	}
	return locked, nil
}

func (r *PostgresRepository) ReleaseCreativeProviderSyncLock(
	ctx context.Context,
	providerConfigID string,
) error {
	var unlocked bool
	if err := r.pool.QueryRow(
		ctx,
		`SELECT pg_advisory_unlock(hashtext('creative_provider_sync'), hashtext($1))`,
		providerConfigID,
	).Scan(&unlocked); err != nil {
		return fmt.Errorf("release creative provider sync lock: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ActiveCreativeForCampaign(ctx context.Context, campaignID string) (Creative, error) {
	creatives, err := r.ActiveCreativesForCampaign(ctx, campaignID)
	if err != nil {
		return Creative{}, err
	}
	if len(creatives) == 0 {
		return Creative{}, ErrNotFound
	}
	return creatives[0], nil
}

func (r *PostgresRepository) ActiveCreativesForCampaign(ctx context.Context, campaignID string) ([]Creative, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT cr.id::text,
       cr.campaign_id::text,
       cr.title,
       cr.body,
       cr.url,
       cr.icon,
       cr.status,
       cr.source_type,
       COALESCE(cr.provider_config_id::text, ''),
       cr.provider_name,
       cr.provider_external_id,
       cr.last_synced_at,
       cr.sync_status,
       cr.daily_cap_per_subscription,
       cr.total_cap_per_subscription,
       ca.daily_cap_per_subscription,
       ca.total_cap_per_subscription,
       ca.targeting_rules::text,
       cr.created_at,
       cr.updated_at
FROM campaigns ca
JOIN creatives cr ON cr.campaign_id = ca.id
WHERE ca.id = $1::uuid
  AND ca.status = $2
  AND cr.status = $3
ORDER BY cr.created_at ASC
`,
		campaignID,
		CampaignStatusActive,
		CreativeStatusActive,
	)
	if err != nil {
		return nil, fmt.Errorf("active creatives for campaign: %w", err)
	}
	defer rows.Close()
	creatives, err := scanCreatives(rows)
	if err != nil {
		return nil, err
	}
	if len(creatives) == 0 {
		return nil, ErrNotFound
	}
	return creatives, nil
}

func (r *PostgresRepository) ActiveCreativeForSource(ctx context.Context, sourceID string) (Creative, error) {
	creatives, err := r.ActiveCreativesForSource(ctx, sourceID)
	if err != nil {
		return Creative{}, err
	}
	if len(creatives) == 0 {
		return Creative{}, ErrNotFound
	}
	return creatives[0], nil
}

func (r *PostgresRepository) ActiveCreativesForSource(ctx context.Context, sourceID string) ([]Creative, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT cr.id::text,
       cr.campaign_id::text,
       cr.title,
       cr.body,
       cr.url,
       cr.icon,
       cr.status,
       cr.source_type,
       COALESCE(cr.provider_config_id::text, ''),
       cr.provider_name,
       cr.provider_external_id,
       cr.last_synced_at,
       cr.sync_status,
       cr.daily_cap_per_subscription,
       cr.total_cap_per_subscription,
       ca.daily_cap_per_subscription,
       ca.total_cap_per_subscription,
       ca.targeting_rules::text,
       cr.created_at,
       cr.updated_at
FROM campaigns ca
JOIN creatives cr ON cr.campaign_id = ca.id
WHERE (
      ca.audience_scope = 'all'
      OR ca.source_id = $1::uuid
      OR EXISTS (
          SELECT 1 FROM campaign_sources cs
          WHERE cs.campaign_id = ca.id AND cs.source_id = $1::uuid
      )
  )
  AND ca.status = $2
  AND cr.status = $3
ORDER BY ca.created_at ASC, cr.created_at ASC
`,
		sourceID,
		CampaignStatusActive,
		CreativeStatusActive,
	)
	if err != nil {
		return nil, fmt.Errorf("active creatives for source: %w", err)
	}
	defer rows.Close()
	creatives, err := scanCreatives(rows)
	if err != nil {
		return nil, err
	}
	if len(creatives) == 0 {
		return nil, ErrNotFound
	}
	return creatives, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPublisher(row rowScanner) (Publisher, error) {
	var publisher Publisher
	if err := row.Scan(
		&publisher.ID,
		&publisher.Name,
		&publisher.Status,
		&publisher.CreatedAt,
		&publisher.UpdatedAt,
	); err != nil {
		return Publisher{}, err
	}
	return publisher, nil
}

func scanSource(row rowScanner) (Source, error) {
	var source Source
	if err := row.Scan(
		&source.ID,
		&source.PublisherID,
		&source.Name,
		&source.Domain,
		&source.Status,
		&source.VAPIDKeyID,
		&source.CreatedAt,
		&source.UpdatedAt,
	); err != nil {
		return Source{}, err
	}
	return source, nil
}

func scanVAPIDKey(row rowScanner) (VAPIDKey, error) {
	var key VAPIDKey
	if err := row.Scan(
		&key.ID,
		&key.Name,
		&key.PublicKey,
		&key.PrivateKey,
		&key.Status,
		&key.CreatedAt,
		&key.UpdatedAt,
	); err != nil {
		return VAPIDKey{}, err
	}
	return key, nil
}

func scanCampaign(row rowScanner) (Campaign, error) {
	var campaign Campaign
	var targetingRulesRaw string
	var publisherID sql.NullString
	var sourceID sql.NullString
	if err := row.Scan(
		&campaign.ID,
		&publisherID,
		&sourceID,
		&campaign.SourceIDs,
		&campaign.AudienceScope,
		&campaign.Name,
		&campaign.Status,
		&targetingRulesRaw,
		&campaign.DailyCapPerSubscription,
		&campaign.TotalCapPerSubscription,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	); err != nil {
		return Campaign{}, err
	}
	rules, err := parseTargetingRules(targetingRulesRaw)
	if err != nil {
		return Campaign{}, err
	}
	campaign.PublisherID = publisherID.String
	campaign.SourceID = sourceID.String
	if campaign.AudienceScope == "" {
		campaign.AudienceScope = CampaignAudienceScopeSelectedSources
	}
	if len(campaign.SourceIDs) == 0 && campaign.SourceID != "" {
		campaign.SourceIDs = []string{campaign.SourceID}
	}
	campaign.TargetingRules = rules
	return campaign, nil
}

func scanCampaignLaunch(row rowScanner) (CampaignLaunch, error) {
	var launch CampaignLaunch
	var completedAt *time.Time
	var enqueueStartedAt *time.Time
	var enqueueCompletedAt *time.Time
	if err := row.Scan(
		&launch.ID,
		&launch.CampaignID,
		&launch.Status,
		&launch.AudienceTotal,
		&launch.ProcessedTotal,
		&launch.ErrorMessage,
		&launch.EnqueueStatus,
		&launch.EnqueuedTotal,
		&launch.EnqueueError,
		&launch.CreatedAt,
		&launch.UpdatedAt,
		&completedAt,
		&enqueueStartedAt,
		&enqueueCompletedAt,
	); err != nil {
		return CampaignLaunch{}, err
	}
	launch.CompletedAt = completedAt
	launch.EnqueueStartedAt = enqueueStartedAt
	launch.EnqueueCompletedAt = enqueueCompletedAt
	return launch, nil
}

func (r *PostgresRepository) scanSchedules(ctx context.Context, rows pgx.Rows) ([]CampaignSchedule, error) {
	schedules := []CampaignSchedule{}
	ids := []string{}
	for rows.Next() {
		var schedule CampaignSchedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.CampaignID,
			&schedule.Status,
			&schedule.TimezoneMode,
			&schedule.FallbackTimezone,
			&schedule.GraceMinutes,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan campaign schedule: %w", err)
		}
		ids = append(ids, schedule.ID)
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaign schedule rows: %w", err)
	}
	slots, err := r.scheduleSlots(ctx, ids)
	if err != nil {
		return nil, err
	}
	for index := range schedules {
		schedules[index].Slots = slots[schedules[index].ID]
	}
	return schedules, nil
}

func (r *PostgresRepository) scheduleSlots(
	ctx context.Context,
	scheduleIDs []string,
) (map[string][]CampaignScheduleSlot, error) {
	result := map[string][]CampaignScheduleSlot{}
	if len(scheduleIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       schedule_id::text,
       to_char(local_time, 'HH24:MI'),
       days_of_week,
       position,
       created_at
FROM campaign_schedule_slots
WHERE schedule_id::text = ANY($1)
ORDER BY schedule_id, position, local_time
`,
		scheduleIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query campaign schedule slots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slot CampaignScheduleSlot
		if err := rows.Scan(
			&slot.ID,
			&slot.ScheduleID,
			&slot.LocalTime,
			&slot.DaysOfWeek,
			&slot.Position,
			&slot.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan campaign schedule slot: %w", err)
		}
		result[slot.ScheduleID] = append(result[slot.ScheduleID], slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaign schedule slot rows: %w", err)
	}
	return result, nil
}

func (r *PostgresRepository) getScheduleRun(ctx context.Context, id string) (CampaignScheduleRun, error) {
	row := r.pool.QueryRow(
		ctx,
		scheduleRunSelectSQL()+`
WHERE id = $1::uuid
`,
		id,
	)
	run, err := scanScheduleRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CampaignScheduleRun{}, ErrNotFound
		}
		return CampaignScheduleRun{}, fmt.Errorf("get campaign schedule run: %w", err)
	}
	return run, nil
}

func scheduleRunSelectSQL() string {
	return `
SELECT id::text,
       schedule_id::text,
       slot_id::text,
       campaign_id::text,
       COALESCE(launch_id::text, ''),
       local_date::text,
       to_char(local_time, 'HH24:MI'),
       timezone,
       scheduled_utc_at,
       status,
       audience_total,
       enqueued_total,
       error_message,
       created_at,
       updated_at,
       completed_at
FROM campaign_schedule_runs
`
}

func scanScheduleRuns(rows pgx.Rows) ([]CampaignScheduleRun, error) {
	runs := []CampaignScheduleRun{}
	for rows.Next() {
		run, err := scanScheduleRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan campaign schedule run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaign schedule run rows: %w", err)
	}
	return runs, nil
}

func scanScheduleRun(row rowScanner) (CampaignScheduleRun, error) {
	var run CampaignScheduleRun
	var completedAt *time.Time
	if err := row.Scan(
		&run.ID,
		&run.ScheduleID,
		&run.SlotID,
		&run.CampaignID,
		&run.LaunchID,
		&run.LocalDate,
		&run.LocalTime,
		&run.Timezone,
		&run.ScheduledUTCAt,
		&run.Status,
		&run.AudienceTotal,
		&run.EnqueuedTotal,
		&run.ErrorMessage,
		&run.CreatedAt,
		&run.UpdatedAt,
		&completedAt,
	); err != nil {
		return CampaignScheduleRun{}, err
	}
	run.CompletedAt = completedAt
	return run, nil
}

func scanCreative(row rowScanner) (Creative, error) {
	var creative Creative
	var campaignTargetingRulesRaw string
	var lastSyncedAt *time.Time
	if err := row.Scan(
		&creative.ID,
		&creative.CampaignID,
		&creative.Title,
		&creative.Body,
		&creative.URL,
		&creative.Icon,
		&creative.Status,
		&creative.SourceType,
		&creative.ProviderConfigID,
		&creative.ProviderName,
		&creative.ProviderExternalID,
		&lastSyncedAt,
		&creative.SyncStatus,
		&creative.DailyCapPerSubscription,
		&creative.TotalCapPerSubscription,
		&creative.CampaignDailyCapPerSubscription,
		&creative.CampaignTotalCapPerSubscription,
		&campaignTargetingRulesRaw,
		&creative.CreatedAt,
		&creative.UpdatedAt,
	); err != nil {
		return Creative{}, err
	}
	creative.LastSyncedAt = lastSyncedAt
	rules, err := parseTargetingRules(campaignTargetingRulesRaw)
	if err != nil {
		return Creative{}, err
	}
	creative.CampaignTargetingRules = rules
	return creative, nil
}

func scanCreativeProviderConfig(row rowScanner) (CreativeProviderConfig, error) {
	var config CreativeProviderConfig
	var headersRaw string
	var lastSyncAt *time.Time
	if err := row.Scan(
		&config.ID,
		&config.CampaignID,
		&config.Name,
		&config.ProviderName,
		&config.FetchURL,
		&headersRaw,
		&config.Status,
		&lastSyncAt,
		&config.CreatedAt,
		&config.UpdatedAt,
	); err != nil {
		return CreativeProviderConfig{}, err
	}
	headers, err := parseHeaders(headersRaw)
	if err != nil {
		return CreativeProviderConfig{}, err
	}
	config.RequestHeaders = headers
	config.LastSyncAt = lastSyncAt
	return config, nil
}

func scanCreativeSyncLog(row rowScanner) (CreativeSyncLog, error) {
	var log CreativeSyncLog
	var completedAt *time.Time
	if err := row.Scan(
		&log.ID,
		&log.ProviderConfigID,
		&log.CampaignID,
		&log.Status,
		&log.FetchedTotal,
		&log.UpsertedTotal,
		&log.ErrorMessage,
		&log.StartedAt,
		&completedAt,
	); err != nil {
		return CreativeSyncLog{}, err
	}
	log.CompletedAt = completedAt
	return log, nil
}

func targetingRulesJSON(rules TargetingRules) string {
	payload, err := json.Marshal(rules)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func headersJSON(headers map[string]string) string {
	payload, err := json.Marshal(headers)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func parseHeaders(raw string) (map[string]string, error) {
	if raw == "" {
		return map[string]string{}, nil
	}
	headers := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return nil, fmt.Errorf("parse request headers: %w", err)
	}
	return headers, nil
}

func parseTargetingRules(raw string) (TargetingRules, error) {
	if raw == "" {
		return TargetingRules{}, nil
	}
	var rules TargetingRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return TargetingRules{}, fmt.Errorf("parse targeting rules: %w", err)
	}
	return normalizeTargetingRules(rules), nil
}

func scanCreatives(rows pgx.Rows) ([]Creative, error) {
	creatives := []Creative{}
	for rows.Next() {
		creative, err := scanCreative(rows)
		if err != nil {
			return nil, fmt.Errorf("scan creative: %w", err)
		}
		creatives = append(creatives, creative)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("creative rows: %w", err)
	}
	return creatives, nil
}
