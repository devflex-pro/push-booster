package postbacks

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

type eventRow struct {
	PostbackConfigID string  `json:"postback_config_id"`
	DedupeKey        string  `json:"dedupe_key"`
	ExternalID       string  `json:"external_id"`
	ClickID          string  `json:"click_id"`
	DeliveryID       string  `json:"delivery_id"`
	SubscriptionID   string  `json:"subscription_id"`
	SourceID         string  `json:"source_id"`
	CampaignID       string  `json:"campaign_id"`
	CreativeID       string  `json:"creative_id"`
	Payout           float64 `json:"payout"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	Attribution      string  `json:"attribution_status"`
	RawPayload       string  `json:"raw_payload"`
	ReceivedAt       string  `json:"received_at"`
}

type attributionRow struct {
	DeliveryID     string `json:"delivery_id"`
	SubscriptionID string `json:"subscription_id"`
	SourceID       string `json:"source_id"`
	CampaignID     string `json:"campaign_id"`
	CreativeID     string `json:"creative_id"`
}

func (r *ClickHouseRepository) RecordEvent(ctx context.Context, event Event) error {
	receivedAt := event.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	row := eventRow{
		PostbackConfigID: event.PostbackConfigID,
		DedupeKey:        event.DedupeKey,
		ExternalID:       event.ExternalID,
		ClickID:          event.ClickID,
		DeliveryID:       event.DeliveryID,
		SubscriptionID:   event.SubscriptionID,
		SourceID:         event.SourceID,
		CampaignID:       event.CampaignID,
		CreativeID:       event.CreativeID,
		Payout:           event.Payout,
		Currency:         event.Currency,
		Status:           event.Status,
		Attribution:      event.Attribution,
		RawPayload:       event.RawPayload,
		ReceivedAt:       receivedAt.UTC().Format("2006-01-02 15:04:05.000"),
	}
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal postback event row: %w", err)
	}
	query := "INSERT INTO postback_events FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert postback event: %w", err)
	}
	return nil
}

func (r *ClickHouseRepository) EventExists(
	ctx context.Context,
	configID string,
	dedupeKey string,
) (bool, error) {
	query := fmt.Sprintf(
		`SELECT count()
FROM postback_events
WHERE postback_config_id = '%s'
  AND dedupe_key = '%s'`,
		escape(configID),
		escape(dedupeKey),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return false, fmt.Errorf("check postback dedupe: %w", err)
	}
	count, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return false, fmt.Errorf("parse postback dedupe count: %w", err)
	}
	return count > 0, nil
}

func (r *ClickHouseRepository) RecentEvents(
	ctx context.Context,
	input RecentEventsInput,
) ([]Event, error) {
	where := ""
	if input.PostbackConfigID != "" {
		where = fmt.Sprintf(
			"\nWHERE postback_config_id = '%s'",
			escape(input.PostbackConfigID),
		)
	}
	query := fmt.Sprintf(
		`SELECT postback_config_id,
       dedupe_key,
       external_id,
       click_id,
       delivery_id,
       subscription_id,
       source_id,
       campaign_id,
       creative_id,
       payout,
       currency,
       status,
       attribution_status,
       raw_payload,
       toString(received_at) AS received_at
FROM postback_events%s
ORDER BY received_at DESC
LIMIT %d
FORMAT JSONEachRow`,
		where,
		input.Limit,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list recent postbacks: %w", err)
	}
	events := []Event{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row eventRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse postback event row: %w", err)
		}
		event, err := eventFromRow(row)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *ClickHouseRepository) ResolveAttribution(
	ctx context.Context,
	input Attribution,
) (Attribution, error) {
	if input.DeliveryID != "" {
		attribution, err := r.resolveByDeliveryID(ctx, input.DeliveryID)
		if err != nil || attribution.SourceID != "" {
			return attribution, err
		}
	}
	if input.SubscriptionID != "" {
		return r.resolveBySubscriptionID(ctx, input.SubscriptionID)
	}
	return input, nil
}

func (r *ClickHouseRepository) resolveByDeliveryID(
	ctx context.Context,
	deliveryID string,
) (Attribution, error) {
	query := fmt.Sprintf(
		`SELECT delivery_id,
       subscription_id,
       source_id,
       campaign_id,
       creative_id
FROM subscriber_events
WHERE delivery_id = '%s'
ORDER BY occurred_at DESC
LIMIT 1
FORMAT JSONEachRow`,
		escape(deliveryID),
	)
	return r.resolveOne(ctx, query, "resolve postback delivery")
}

func (r *ClickHouseRepository) resolveBySubscriptionID(
	ctx context.Context,
	subscriptionID string,
) (Attribution, error) {
	query := fmt.Sprintf(
		`SELECT '' AS delivery_id,
       subscription_id,
       source_id,
       '' AS campaign_id,
       '' AS creative_id
FROM subscribers
WHERE subscription_id = '%s'
ORDER BY subscribed_at DESC
LIMIT 1
FORMAT JSONEachRow`,
		escape(subscriptionID),
	)
	return r.resolveOne(ctx, query, "resolve postback subscription")
}

func (r *ClickHouseRepository) resolveOne(
	ctx context.Context,
	query string,
	operation string,
) (Attribution, error) {
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return Attribution{}, fmt.Errorf("%s: %w", operation, err)
	}
	if strings.TrimSpace(raw) == "" {
		return Attribution{}, nil
	}
	var row attributionRow
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		return Attribution{}, fmt.Errorf("parse %s: %w", operation, err)
	}
	return Attribution(row), nil
}

func eventFromRow(row eventRow) (Event, error) {
	receivedAt, err := time.Parse("2006-01-02 15:04:05.000", row.ReceivedAt)
	if err != nil {
		return Event{}, fmt.Errorf("parse postback received_at: %w", err)
	}
	return Event{
		PostbackConfigID: row.PostbackConfigID,
		DedupeKey:        row.DedupeKey,
		ExternalID:       row.ExternalID,
		ClickID:          row.ClickID,
		DeliveryID:       row.DeliveryID,
		SubscriptionID:   row.SubscriptionID,
		SourceID:         row.SourceID,
		CampaignID:       row.CampaignID,
		CreativeID:       row.CreativeID,
		Payout:           row.Payout,
		Currency:         row.Currency,
		Status:           row.Status,
		Attribution:      row.Attribution,
		RawPayload:       row.RawPayload,
		ReceivedAt:       receivedAt,
	}, nil
}

func escape(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
