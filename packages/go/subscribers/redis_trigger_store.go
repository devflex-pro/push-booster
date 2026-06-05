package subscribers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/redis"
)

type RedisTriggerStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisTriggerStore(client *redis.Client, ttl time.Duration) *RedisTriggerStore {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &RedisTriggerStore{client: client, ttl: ttl}
}

func (s *RedisTriggerStore) CreateTrigger(
	ctx context.Context,
	input CreateTriggerInput,
) (DeliveryTrigger, error) {
	now := time.Now().UTC()
	ttl := input.TTL
	if ttl == 0 {
		ttl = s.ttl
	}
	trigger := DeliveryTrigger{
		TriggerID:      strings.TrimSpace(input.TriggerID),
		DeliveryID:     strings.TrimSpace(input.DeliveryID),
		SubscriptionID: strings.TrimSpace(input.SubscriptionID),
		SourceID:       strings.TrimSpace(input.SourceID),
		CampaignID:     strings.TrimSpace(input.CampaignID),
		CreatedAt:      now,
		ExpiresAt:      now.Add(ttl),
	}
	if trigger.TriggerID == "" {
		id, err := generateSubscriptionID()
		if err != nil {
			return DeliveryTrigger{}, err
		}
		trigger.TriggerID = id
	}
	if trigger.DeliveryID == "" {
		id, err := generateSubscriptionID()
		if err != nil {
			return DeliveryTrigger{}, err
		}
		trigger.DeliveryID = id
	}
	payload, err := json.Marshal(trigger)
	if err != nil {
		return DeliveryTrigger{}, fmt.Errorf("marshal delivery trigger: %w", err)
	}
	if err := s.client.SetEX(ctx, triggerKey(trigger.TriggerID), string(payload), ttl); err != nil {
		return DeliveryTrigger{}, fmt.Errorf("store delivery trigger: %w", err)
	}
	return trigger, nil
}

func (s *RedisTriggerStore) ResolveTrigger(ctx context.Context, triggerID string) (DeliveryTrigger, error) {
	raw, err := s.client.Get(ctx, triggerKey(triggerID))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return DeliveryTrigger{}, inventory.ErrNotFound
		}
		return DeliveryTrigger{}, fmt.Errorf("resolve delivery trigger: %w", err)
	}
	var trigger DeliveryTrigger
	if err := json.Unmarshal([]byte(raw), &trigger); err != nil {
		return DeliveryTrigger{}, fmt.Errorf("parse delivery trigger: %w", err)
	}
	return trigger, nil
}

func triggerKey(triggerID string) string {
	return "push_booster:delivery_trigger:" + strings.TrimSpace(triggerID)
}
