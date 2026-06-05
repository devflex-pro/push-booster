package subscribers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/redis"
)

type RedisEventIDStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisEventIDStore(
	client *redis.Client,
	ttl time.Duration,
) *RedisEventIDStore {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &RedisEventIDStore{client: client, ttl: ttl}
}

func (s *RedisEventIDStore) AllowEvent(
	ctx context.Context,
	input ServiceWorkerEventInput,
) (bool, error) {
	if s == nil || s.client == nil {
		return true, nil
	}
	key := eventIDKey(input)
	if key == "" {
		return true, nil
	}
	count, err := s.client.Incr(ctx, key)
	if err != nil {
		return false, fmt.Errorf("increment event idempotency key: %w", err)
	}
	if count == 1 {
		if err := s.client.Expire(
			ctx,
			key,
			s.ttl,
		); err != nil {
			return false, fmt.Errorf("expire event idempotency key: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func eventIDKey(input ServiceWorkerEventInput) string {
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		if strings.TrimSpace(input.DeliveryID) == "" {
			return ""
		}
		eventID = strings.TrimSpace(input.DeliveryID) + ":" + strings.TrimSpace(input.EventType)
	}
	return "push_booster:sw_event:" + eventID
}
