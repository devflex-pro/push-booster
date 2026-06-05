package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
)

type ClickHouseRepository struct {
	client *clickhouse.Client
}

func NewClickHouseRepository(client *clickhouse.Client) *ClickHouseRepository {
	return &ClickHouseRepository{client: client}
}

type revenueRow struct {
	Key         string `json:"key"`
	Revenue     string `json:"revenue"`
	Conversions string `json:"conversions"`
}

type eventRow struct {
	Key    string `json:"key"`
	Shown  string `json:"shown"`
	Clicks string `json:"clicks"`
	Closed string `json:"closed"`
}

type deliveryRow struct {
	Key  string `json:"key"`
	Sent string `json:"sent"`
}

func (r *ClickHouseRepository) MetricsByGroup(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]MetricRow, error) {
	revenue, err := r.revenueByGroup(ctx, groupBy, dateRange)
	if err != nil {
		return nil, err
	}
	events, err := r.eventsByGroup(ctx, groupBy, dateRange)
	if err != nil {
		return nil, err
	}
	deliveries, err := r.deliveriesByGroup(ctx, groupBy, dateRange)
	if err != nil {
		return nil, err
	}
	rows := map[string]MetricRow{}
	for _, row := range revenue {
		rows[row.Key] = row
	}
	for _, event := range events {
		row := rows[event.Key]
		row.Key = event.Key
		row.Shown = event.Shown
		row.Clicks = event.Clicks
		row.Closed = event.Closed
		rows[event.Key] = row
	}
	for _, delivery := range deliveries {
		row := rows[delivery.Key]
		row.Key = delivery.Key
		row.Sent = delivery.Sent
		rows[delivery.Key] = row
	}
	result := make([]MetricRow, 0, len(rows))
	for _, row := range rows {
		if row.Key == "" {
			continue
		}
		result = append(result, row)
	}
	return result, nil
}

func (r *ClickHouseRepository) deliveriesByGroup(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]MetricRow, error) {
	if groupBy == GroupCreative {
		return r.creativeDeliveriesByGroup(ctx, dateRange)
	}
	field := clickHouseGroupField(groupBy, "occurred_at")
	query := fmt.Sprintf(
		`SELECT %s AS key,
       toString(countIf(event_type = 'sent')) AS sent
FROM push_events
WHERE occurred_at >= toDateTime64('%s', 3, 'UTC')
  AND occurred_at < toDateTime64('%s', 3, 'UTC')
GROUP BY key
FORMAT JSONEachRow`,
		field,
		formatTime(dateRange.From),
		formatTime(dateRange.To.AddDate(0, 0, 1)),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query report deliveries: %w", err)
	}
	rows := []MetricRow{}
	for _, line := range lines(raw) {
		var parsed deliveryRow
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			return nil, fmt.Errorf("parse delivery row: %w", err)
		}
		sent, err := strconv.ParseInt(parsed.Sent, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse sent value: %w", err)
		}
		rows = append(rows, MetricRow{
			Key:  parsed.Key,
			Sent: sent,
		})
	}
	return rows, nil
}

func (r *ClickHouseRepository) creativeDeliveriesByGroup(
	ctx context.Context,
	dateRange DateRange,
) ([]MetricRow, error) {
	query := fmt.Sprintf(
		`SELECT pd.creative_id AS key,
       toString(countIf(pe.event_type = 'sent')) AS sent
FROM push_events pe
INNER JOIN payload_decisions pd ON pd.trigger_id = toString(pe.trigger_id)
WHERE pe.occurred_at >= toDateTime64('%s', 3, 'UTC')
  AND pe.occurred_at < toDateTime64('%s', 3, 'UTC')
GROUP BY key
FORMAT JSONEachRow`,
		formatTime(dateRange.From),
		formatTime(dateRange.To.AddDate(0, 0, 1)),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query report creative deliveries: %w", err)
	}
	rows := []MetricRow{}
	for _, line := range lines(raw) {
		var parsed deliveryRow
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			return nil, fmt.Errorf("parse creative delivery row: %w", err)
		}
		sent, err := strconv.ParseInt(parsed.Sent, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse creative sent value: %w", err)
		}
		rows = append(rows, MetricRow{
			Key:  parsed.Key,
			Sent: sent,
		})
	}
	return rows, nil
}

func (r *ClickHouseRepository) revenueByGroup(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]MetricRow, error) {
	field := clickHouseGroupField(groupBy, "received_at")
	query := fmt.Sprintf(
		`SELECT %s AS key,
       toString(sum(payout)) AS revenue,
       toString(count()) AS conversions
FROM postback_events
WHERE received_at >= toDateTime64('%s', 3, 'UTC')
  AND received_at < toDateTime64('%s', 3, 'UTC')
GROUP BY key
FORMAT JSONEachRow`,
		field,
		formatTime(dateRange.From),
		formatTime(dateRange.To.AddDate(0, 0, 1)),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query report revenue: %w", err)
	}
	rows := []MetricRow{}
	for _, line := range lines(raw) {
		var parsed revenueRow
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			return nil, fmt.Errorf("parse revenue row: %w", err)
		}
		revenue, err := strconv.ParseFloat(parsed.Revenue, 64)
		if err != nil {
			return nil, fmt.Errorf("parse revenue value: %w", err)
		}
		conversions, err := strconv.ParseInt(parsed.Conversions, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse conversions value: %w", err)
		}
		rows = append(rows, MetricRow{
			Key:         parsed.Key,
			Revenue:     revenue,
			Conversions: conversions,
		})
	}
	return rows, nil
}

func (r *ClickHouseRepository) eventsByGroup(
	ctx context.Context,
	groupBy string,
	dateRange DateRange,
) ([]MetricRow, error) {
	field := clickHouseGroupField(groupBy, "occurred_at")
	query := fmt.Sprintf(
		`SELECT %s AS key,
       toString(countIf(event_type = 'notification_shown')) AS shown,
       toString(countIf(event_type = 'notification_click')) AS clicks,
       toString(countIf(event_type = 'notification_close')) AS closed
FROM subscriber_events
WHERE occurred_at >= toDateTime64('%s', 3, 'UTC')
  AND occurred_at < toDateTime64('%s', 3, 'UTC')
GROUP BY key
FORMAT JSONEachRow`,
		field,
		formatTime(dateRange.From),
		formatTime(dateRange.To.AddDate(0, 0, 1)),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query report events: %w", err)
	}
	rows := []MetricRow{}
	for _, line := range lines(raw) {
		var parsed eventRow
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			return nil, fmt.Errorf("parse event row: %w", err)
		}
		shown, err := strconv.ParseInt(parsed.Shown, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse shown value: %w", err)
		}
		clicks, err := strconv.ParseInt(parsed.Clicks, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse clicks value: %w", err)
		}
		closed, err := strconv.ParseInt(parsed.Closed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse closed value: %w", err)
		}
		rows = append(rows, MetricRow{
			Key:    parsed.Key,
			Shown:  shown,
			Clicks: clicks,
			Closed: closed,
		})
	}
	return rows, nil
}

func clickHouseGroupField(groupBy string, timeField string) string {
	switch groupBy {
	case GroupDate:
		return fmt.Sprintf("toString(toDate(%s))", timeField)
	case GroupCampaign:
		return "campaign_id"
	case GroupCreative:
		return "creative_id"
	default:
		return "source_id"
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

func lines(raw string) []string {
	result := []string{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
