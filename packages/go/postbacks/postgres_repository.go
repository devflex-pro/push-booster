package postbacks

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateConfig(
	ctx context.Context,
	input CreateConfigInput,
) (Config, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO postback_configs (
    name,
    source_id,
    token,
    status,
    click_id_param,
    delivery_id_param,
    subscription_id_param,
    external_id_param,
    payout_param,
    currency_param,
    status_param,
    default_currency
)
VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id::text,
          name,
          COALESCE(source_id::text, ''),
          token,
          status,
          click_id_param,
          delivery_id_param,
          subscription_id_param,
          external_id_param,
          payout_param,
          currency_param,
          status_param,
          default_currency,
          created_at,
          updated_at
`,
		input.Name,
		input.SourceID,
		input.Token,
		StatusActive,
		input.ClickIDParam,
		input.DeliveryIDParam,
		input.SubscriptionIDParam,
		input.ExternalIDParam,
		input.PayoutParam,
		input.CurrencyParam,
		input.StatusParam,
		input.DefaultCurrency,
	)
	cfg, err := scanConfig(row)
	if err != nil {
		return Config{}, fmt.Errorf("create postback config: %w", err)
	}
	return cfg, nil
}

func (r *PostgresRepository) ListConfigs(ctx context.Context) ([]Config, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       name,
       COALESCE(source_id::text, ''),
       token,
       status,
       click_id_param,
       delivery_id_param,
       subscription_id_param,
       external_id_param,
       payout_param,
       currency_param,
       status_param,
       default_currency,
       created_at,
       updated_at
FROM postback_configs
ORDER BY created_at DESC
`,
	)
	if err != nil {
		return nil, fmt.Errorf("list postback configs: %w", err)
	}
	defer rows.Close()

	configs := []Config{}
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("scan postback config: %w", err)
		}
		configs = append(configs, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list postback configs rows: %w", err)
	}
	return configs, nil
}

func (r *PostgresRepository) GetConfig(ctx context.Context, id string) (Config, error) {
	row := r.pool.QueryRow(
		ctx,
		`
SELECT id::text,
       name,
       COALESCE(source_id::text, ''),
       token,
       status,
       click_id_param,
       delivery_id_param,
       subscription_id_param,
       external_id_param,
       payout_param,
       currency_param,
       status_param,
       default_currency,
       created_at,
       updated_at
FROM postback_configs
WHERE id = $1::uuid
LIMIT 1
`,
		id,
	)
	cfg, err := scanConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("get postback config: %w", err)
	}
	return cfg, nil
}

func (r *PostgresRepository) UpdateConfigStatus(
	ctx context.Context,
	input UpdateConfigStatusInput,
) (Config, error) {
	row := r.pool.QueryRow(
		ctx,
		`
UPDATE postback_configs
SET status = $2, updated_at = now()
WHERE id = $1::uuid
RETURNING id::text,
          name,
          COALESCE(source_id::text, ''),
          token,
          status,
          click_id_param,
          delivery_id_param,
          subscription_id_param,
          external_id_param,
          payout_param,
          currency_param,
          status_param,
          default_currency,
          created_at,
          updated_at
`,
		input.ID,
		input.Status,
	)
	cfg, err := scanConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("update postback config status: %w", err)
	}
	return cfg, nil
}

type configScanner interface {
	Scan(dest ...any) error
}

func scanConfig(row configScanner) (Config, error) {
	var cfg Config
	if err := row.Scan(
		&cfg.ID,
		&cfg.Name,
		&cfg.SourceID,
		&cfg.Token,
		&cfg.Status,
		&cfg.ClickIDParam,
		&cfg.DeliveryIDParam,
		&cfg.SubscriptionIDParam,
		&cfg.ExternalIDParam,
		&cfg.PayoutParam,
		&cfg.CurrencyParam,
		&cfg.StatusParam,
		&cfg.DefaultCurrency,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
