package reports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCostEntry(
	ctx context.Context,
	input CreateCostEntryInput,
) (CostEntry, error) {
	row := r.pool.QueryRow(
		ctx,
		`
INSERT INTO cost_entries (
    cost_date,
    publisher_id,
    source_id,
    campaign_id,
    creative_id,
    amount,
    currency,
    note
)
VALUES (
    $1::date,
    NULLIF($2, '')::uuid,
    NULLIF($3, '')::uuid,
    NULLIF($4, '')::uuid,
    NULLIF($5, '')::uuid,
    $6,
    $7,
    $8
)
RETURNING id::text,
          cost_date::text,
          COALESCE(publisher_id::text, ''),
          '',
          COALESCE(source_id::text, ''),
          '',
          COALESCE(campaign_id::text, ''),
          '',
          COALESCE(creative_id::text, ''),
          '',
          amount,
          currency,
          note,
          created_at
`,
		input.Date,
		input.PublisherID,
		input.SourceID,
		input.CampaignID,
		input.CreativeID,
		input.Amount,
		input.Currency,
		input.Note,
	)
	entry, err := scanCostEntry(row)
	if err != nil {
		return CostEntry{}, fmt.Errorf("create cost entry: %w", err)
	}
	return entry, nil
}

func (r *PostgresRepository) ListCostEntries(
	ctx context.Context,
	input ListCostEntriesInput,
) (CostEntryPage, error) {
	var total int
	if err := r.pool.QueryRow(
		ctx,
		`SELECT count(*) FROM cost_entries`,
	).Scan(&total); err != nil {
		return CostEntryPage{}, fmt.Errorf("count cost entries: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       cost_date::text,
       COALESCE(publisher_id::text, ''),
       COALESCE((SELECT name FROM publishers WHERE id = cost_entries.publisher_id), ''),
       COALESCE(source_id::text, ''),
       COALESCE((SELECT name FROM sources WHERE id = cost_entries.source_id), ''),
       COALESCE(campaign_id::text, ''),
       COALESCE((SELECT name FROM campaigns WHERE id = cost_entries.campaign_id), ''),
       COALESCE(creative_id::text, ''),
       COALESCE((SELECT title FROM creatives WHERE id = cost_entries.creative_id), ''),
       amount,
       currency,
       note,
       created_at
FROM cost_entries
ORDER BY cost_date DESC, created_at DESC
LIMIT $1
OFFSET $2
`,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		return CostEntryPage{}, fmt.Errorf("list cost entries: %w", err)
	}
	defer rows.Close()

	entries := []CostEntry{}
	for rows.Next() {
		entry, err := scanCostEntry(rows)
		if err != nil {
			return CostEntryPage{}, fmt.Errorf("scan cost entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return CostEntryPage{}, fmt.Errorf("list cost entries rows: %w", err)
	}
	return CostEntryPage{Items: entries, Total: total}, nil
}

func (r *PostgresRepository) CostAllocations(
	ctx context.Context,
	dateRange DateRange,
) ([]CostAllocationRow, error) {
	rows, err := r.pool.Query(
		ctx,
		`
SELECT COALESCE(publisher_id::text, '') AS publisher_id,
       COALESCE(source_id::text, '') AS source_id,
       COALESCE(sum(amount), 0) AS spend
FROM cost_entries
WHERE cost_date >= $1::date
  AND cost_date <= $2::date
GROUP BY publisher_id, source_id
`,
		dateRange.From.Format("2006-01-02"),
		dateRange.To.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("query cost allocations: %w", err)
	}
	defer rows.Close()

	costs := []CostAllocationRow{}
	for rows.Next() {
		var row CostAllocationRow
		if err := rows.Scan(&row.PublisherID, &row.SourceID, &row.Spend); err != nil {
			return nil, fmt.Errorf("scan cost allocation: %w", err)
		}
		costs = append(costs, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cost allocation rows: %w", err)
	}
	return costs, nil
}

func (r *PostgresRepository) CostsByGroup(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]CostRow, error) {
	field := costGroupField(groupBy)
	query := fmt.Sprintf(
		`
SELECT COALESCE(%s::text, '') AS key,
       COALESCE(sum(amount), 0) AS spend
FROM cost_entries
WHERE cost_date >= $1::date
  AND cost_date <= $2::date
GROUP BY key
`,
		field,
	)
	rows, err := r.pool.Query(
		ctx,
		query,
		dateRange.From.Format("2006-01-02"),
		dateRange.To.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("query cost entries by group: %w", err)
	}
	defer rows.Close()

	costs := []CostRow{}
	for rows.Next() {
		var row CostRow
		if err := rows.Scan(&row.Key, &row.Spend); err != nil {
			return nil, fmt.Errorf("scan cost group: %w", err)
		}
		costs = append(costs, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cost group rows: %w", err)
	}
	return costs, nil
}

func (r *PostgresRepository) DimensionNames(
	ctx context.Context,
	groupBy string,
	keys []string,
) (map[string]string, error) {
	if groupBy == GroupDate {
		return map[string]string{}, nil
	}
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	table, nameField := dimensionTable(groupBy)
	query := fmt.Sprintf(
		`
SELECT id::text,
       %s
FROM %s
WHERE id::text = ANY($1)
`,
		nameField,
		table,
	)
	rows, err := r.pool.Query(ctx, query, keys)
	if err != nil {
		return nil, fmt.Errorf("query dimension names: %w", err)
	}
	defer rows.Close()

	names := map[string]string{}
	for rows.Next() {
		var key string
		var name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, fmt.Errorf("scan dimension name: %w", err)
		}
		names[key] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dimension name rows: %w", err)
	}
	return names, nil
}

func (r *PostgresRepository) PublisherIDsBySource(
	ctx context.Context,
	sourceIDs []string,
) (map[string]string, error) {
	if len(sourceIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       publisher_id::text
FROM sources
WHERE id::text = ANY($1)
`,
		sourceIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query source publishers: %w", err)
	}
	defer rows.Close()

	publishers := map[string]string{}
	for rows.Next() {
		var sourceID string
		var publisherID string
		if err := rows.Scan(&sourceID, &publisherID); err != nil {
			return nil, fmt.Errorf("scan source publisher: %w", err)
		}
		publishers[sourceID] = publisherID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("source publisher rows: %w", err)
	}
	return publishers, nil
}

func (r *PostgresRepository) SourceIDsByCampaign(
	ctx context.Context,
	campaignIDs []string,
) (map[string]string, error) {
	if len(campaignIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       COALESCE(source_id::text, '')
FROM campaigns
WHERE id::text = ANY($1)
`,
		campaignIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query campaign sources: %w", err)
	}
	defer rows.Close()

	sources := map[string]string{}
	for rows.Next() {
		var campaignID string
		var sourceID string
		if err := rows.Scan(&campaignID, &sourceID); err != nil {
			return nil, fmt.Errorf("scan campaign source: %w", err)
		}
		sources[campaignID] = sourceID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("campaign source rows: %w", err)
	}
	return sources, nil
}

func (r *PostgresRepository) SourceIDsByCreative(
	ctx context.Context,
	creativeIDs []string,
) (map[string]string, error) {
	if len(creativeIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`
SELECT cr.id::text,
       COALESCE(ca.source_id::text, '')
FROM creatives cr
JOIN campaigns ca ON ca.id = cr.campaign_id
WHERE cr.id::text = ANY($1)
`,
		creativeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query creative sources: %w", err)
	}
	defer rows.Close()

	sources := map[string]string{}
	for rows.Next() {
		var creativeID string
		var sourceID string
		if err := rows.Scan(&creativeID, &sourceID); err != nil {
			return nil, fmt.Errorf("scan creative source: %w", err)
		}
		sources[creativeID] = sourceID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("creative source rows: %w", err)
	}
	return sources, nil
}

func (r *PostgresRepository) CampaignIDsByCreative(
	ctx context.Context,
	creativeIDs []string,
) (map[string]string, error) {
	if len(creativeIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`
SELECT id::text,
       campaign_id::text
FROM creatives
WHERE id::text = ANY($1)
`,
		creativeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query creative campaigns: %w", err)
	}
	defer rows.Close()

	campaigns := map[string]string{}
	for rows.Next() {
		var creativeID string
		var campaignID string
		if err := rows.Scan(&creativeID, &campaignID); err != nil {
			return nil, fmt.Errorf("scan creative campaign: %w", err)
		}
		campaigns[creativeID] = campaignID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("creative campaign rows: %w", err)
	}
	return campaigns, nil
}

type costScanner interface {
	Scan(dest ...any) error
}

func scanCostEntry(row costScanner) (CostEntry, error) {
	var entry CostEntry
	if err := row.Scan(
		&entry.ID,
		&entry.Date,
		&entry.PublisherID,
		&entry.PublisherName,
		&entry.SourceID,
		&entry.SourceName,
		&entry.CampaignID,
		&entry.CampaignName,
		&entry.CreativeID,
		&entry.CreativeName,
		&entry.Amount,
		&entry.Currency,
		&entry.Note,
		&entry.CreatedAt,
	); err != nil {
		return CostEntry{}, err
	}
	return entry, nil
}

func costGroupField(groupBy string) string {
	switch groupBy {
	case GroupDate:
		return "cost_date"
	case GroupPublisher:
		return "publisher_id"
	case GroupCampaign:
		return "campaign_id"
	case GroupCreative:
		return "creative_id"
	default:
		return "source_id"
	}
}

func dimensionTable(groupBy string) (string, string) {
	switch groupBy {
	case GroupPublisher:
		return "publishers", "name"
	case GroupCampaign:
		return "campaigns", "name"
	case GroupCreative:
		return "creatives", "title"
	default:
		return "sources", "name"
	}
}
