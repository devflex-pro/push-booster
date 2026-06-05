package subscribers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/redis"
)

type RedisCapStore struct {
	client *redis.Client
	now    func() time.Time
}

func NewRedisCapStore(client *redis.Client) *RedisCapStore {
	return &RedisCapStore{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *RedisCapStore) Allow(ctx context.Context, input CapCheckInput) (bool, error) {
	if input.DailyCapPerSubscription <= 0 &&
		input.TotalCapPerSubscription <= 0 &&
		input.CreativeDailyCap <= 0 &&
		input.CreativeTotalCap <= 0 {
		return true, nil
	}
	now := s.now()
	checks := []capLimit{
		{
			key: dailyCapKey(
				input.SubscriptionID,
				input.CampaignID,
				now,
			),
			limit: input.DailyCapPerSubscription,
			ttl:   ttlUntilTomorrow(now),
		},
		{
			key: totalCapKey(
				input.SubscriptionID,
				input.CampaignID,
			),
			limit: input.TotalCapPerSubscription,
		},
		{
			key: creativeDailyCapKey(
				input.SubscriptionID,
				input.CampaignID,
				input.CreativeID,
				now,
			),
			limit: input.CreativeDailyCap,
			ttl:   ttlUntilTomorrow(now),
		},
		{
			key: creativeTotalCapKey(
				input.SubscriptionID,
				input.CampaignID,
				input.CreativeID,
			),
			limit: input.CreativeTotalCap,
		},
	}
	for _, check := range checks {
		if check.limit <= 0 {
			continue
		}
		count, err := s.count(ctx, check.key)
		if err != nil {
			return false, err
		}
		if count >= int64(check.limit) {
			return false, nil
		}
	}
	for _, check := range checks {
		if check.limit <= 0 {
			continue
		}
		count, err := s.client.Incr(ctx, check.key)
		if err != nil {
			return false, fmt.Errorf("increment cap: %w", err)
		}
		if count == 1 && check.ttl > 0 {
			if err := s.client.Expire(ctx, check.key, check.ttl); err != nil {
				return false, fmt.Errorf("expire cap: %w", err)
			}
		}
	}
	return true, nil
}

type capLimit struct {
	key   string
	limit int
	ttl   time.Duration
}

func (s *RedisCapStore) count(ctx context.Context, key string) (int64, error) {
	value, err := s.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return 0, nil
		}
		return 0, fmt.Errorf("read cap: %w", err)
	}
	count, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse cap: %w", err)
	}
	return count, nil
}

func dailyCapKey(
	subscriptionID string,
	campaignID string,
	now time.Time,
) string {
	return fmt.Sprintf(
		"push_booster:cap:sub:%s:campaign:%s:day:%s",
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(campaignID),
		now.UTC().Format("20060102"),
	)
}

func totalCapKey(
	subscriptionID string,
	campaignID string,
) string {
	return fmt.Sprintf(
		"push_booster:cap:sub:%s:campaign:%s:total",
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(campaignID),
	)
}

func creativeDailyCapKey(
	subscriptionID string,
	campaignID string,
	creativeID string,
	now time.Time,
) string {
	return fmt.Sprintf(
		"push_booster:cap:sub:%s:campaign:%s:creative:%s:day:%s",
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(campaignID),
		strings.TrimSpace(creativeID),
		now.UTC().Format("20060102"),
	)
}

func creativeTotalCapKey(
	subscriptionID string,
	campaignID string,
	creativeID string,
) string {
	return fmt.Sprintf(
		"push_booster:cap:sub:%s:campaign:%s:creative:%s:total",
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(campaignID),
		strings.TrimSpace(creativeID),
	)
}

func ttlUntilTomorrow(now time.Time) time.Duration {
	now = now.UTC()
	tomorrow := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	return tomorrow.Sub(now)
}
