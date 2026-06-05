package sender

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/redis"
)

type RedisProviderThrottler struct {
	client *redis.Client
	limit  int
}

func NewRedisProviderThrottler(
	client *redis.Client,
	limit int,
) *RedisProviderThrottler {
	return &RedisProviderThrottler{
		client: client,
		limit:  limit,
	}
}

func (t *RedisProviderThrottler) Wait(ctx context.Context, endpoint string) error {
	if t == nil || t.client == nil || t.limit <= 0 {
		return nil
	}
	host, err := endpointHost(endpoint)
	if err != nil {
		return err
	}
	for {
		now := time.Now().UTC()
		key := providerThrottleKey(host, now)
		count, err := t.client.Incr(ctx, key)
		if err != nil {
			return fmt.Errorf("increment provider throttle: %w", err)
		}
		if count == 1 {
			if err := t.client.Expire(
				ctx,
				key,
				2*time.Second,
			); err != nil {
				return fmt.Errorf("expire provider throttle: %w", err)
			}
		}
		if count <= int64(t.limit) {
			return nil
		}
		nextWindow := now.Truncate(time.Second).Add(time.Second)
		wait := time.Until(nextWindow)
		if wait <= 0 {
			continue
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func endpointHost(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse web push endpoint: %w", err)
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("web push endpoint host is required")
	}
	return strings.ToLower(host), nil
}

func providerThrottleKey(host string, now time.Time) string {
	return "push_booster:sender:provider:" +
		host +
		":" +
		strconv.FormatInt(now.Unix(), 10)
}
