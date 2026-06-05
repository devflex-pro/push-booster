package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/logger"
	"github.com/devflex-pro/push-booster/packages/go/postgres"
	"github.com/devflex-pro/push-booster/packages/go/redis"
	"github.com/devflex-pro/push-booster/packages/go/redpanda"
	"github.com/devflex-pro/push-booster/packages/go/sender"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	cfg := config.LoadSender()
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Error(
			"failed to connect postgres",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer db.Close()

	clickHouseClient := clickhouse.NewClient(clickhouse.Config{
		URL:      cfg.ClickHouse.URL,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	redisClient := redis.NewClient(redis.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Timeout:  cfg.Redis.Timeout,
	})
	triggerStore := subscribers.NewRedisTriggerStore(redisClient, cfg.TriggerTTL)
	repository := sender.NewClickHouseRepository(clickHouseClient)
	producer, err := redpanda.NewProducer(ctx, redpanda.Config{Brokers: cfg.Redpanda.Brokers})
	if err != nil {
		log.Error(
			"failed to connect redpanda producer",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer producer.Close()

	service := sender.NewService(
		inventory.NewPostgresRepository(db),
		triggerStore,
		repository,
		producer,
		sender.NewWebPushClient(cfg.WebPushSubject, cfg.WebPushTTLSeconds),
		sender.NewRedisProviderThrottler(redisClient, cfg.ProviderRateLimit),
		sender.Config{
			TriggerTTL:  cfg.TriggerTTL,
			MaxAttempts: cfg.MaxAttempts,
			Concurrency: cfg.Concurrency,
		},
	)
	consumer, err := redpanda.NewConsumer(
		redpanda.Config{Brokers: cfg.Redpanda.Brokers},
		cfg.ConsumerGroup,
		redpanda.DeliveryTopics[:4],
	)
	if err != nil {
		log.Error(
			"failed to create redpanda consumer",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer consumer.Close()

	log.Info(
		"sender started",
		slog.String("consumer_group", cfg.ConsumerGroup),
		slog.Int("concurrency", service.Concurrency()),
		slog.Int("provider_rate_limit_per_second", cfg.ProviderRateLimit),
		slog.Int("max_attempts", cfg.MaxAttempts),
	)
	limiter := make(chan struct{}, service.Concurrency())
	var workers sync.WaitGroup
	defer workers.Wait()
	if err := consumer.Run(ctx, func(
		ctx context.Context,
		record *kgo.Record,
	) error {
		task, err := sender.DecodeTask(record.Value)
		if err != nil {
			log.Error(
				"failed to decode delivery task",
				slog.String("error", err.Error()),
			)
			return nil
		}
		select {
		case limiter <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer func() {
				<-limiter
			}()
			if err := service.ProcessTask(ctx, task); err != nil {
				log.Error(
					"failed to process delivery task",
					slog.String("delivery_id", task.DeliveryID),
					slog.String("error", err.Error()),
				)
			}
		}()
		return nil
	}); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(
			"sender stopped with error",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
