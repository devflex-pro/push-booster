package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
)

type ClickHouseRepository struct {
	client *clickhouse.Client
}

func NewClickHouseRepository(client *clickhouse.Client) *ClickHouseRepository {
	return &ClickHouseRepository{client: client}
}

type PushEventInput struct {
	DeliveryID     string
	TriggerID      string
	LaunchID       string
	CampaignID     string
	SourceID       string
	SubscriptionID string
	EventType      string
	Attempt        int
	Error          string
	OccurredAt     time.Time
}

type pushEventRow struct {
	DeliveryID     string `json:"delivery_id"`
	TriggerID      string `json:"trigger_id"`
	LaunchID       string `json:"launch_id"`
	CampaignID     string `json:"campaign_id"`
	SourceID       string `json:"source_id"`
	SubscriptionID string `json:"subscription_id"`
	EventType      string `json:"event_type"`
	Attempt        int    `json:"attempt"`
	Error          string `json:"error"`
	OccurredAt     string `json:"occurred_at"`
}

func (r *ClickHouseRepository) RecordPushEvent(ctx context.Context, input PushEventInput) error {
	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	row := pushEventRow{
		DeliveryID:     input.DeliveryID,
		TriggerID:      uuidOrZero(input.TriggerID),
		LaunchID:       input.LaunchID,
		CampaignID:     input.CampaignID,
		SourceID:       input.SourceID,
		SubscriptionID: input.SubscriptionID,
		EventType:      input.EventType,
		Attempt:        input.Attempt,
		Error:          input.Error,
		OccurredAt:     occurredAt.UTC().Format("2006-01-02 15:04:05.000"),
	}
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal push event row: %w", err)
	}
	query := "INSERT INTO push_events FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert push event: %w", err)
	}
	return nil
}

func uuidOrZero(value string) string {
	if value == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return value
}
