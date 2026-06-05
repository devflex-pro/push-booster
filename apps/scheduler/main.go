package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/logger"
	"github.com/devflex-pro/push-booster/packages/go/postgres"
	"github.com/devflex-pro/push-booster/packages/go/redpanda"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

func main() {
	cfg := config.LoadScheduler()
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := postgres.Connect(ctx, cfg.PostgresURL)
	if err != nil {
		log.Error("failed to connect postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	clickHouseClient := clickhouse.NewClient(clickhouse.Config{
		URL:      cfg.ClickHouse.URL,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	subscriberRepo := subscribers.NewClickHouseRepository(clickHouseClient)
	subscriberService := subscribers.NewService(
		subscriberRepo,
		inventory.NewPostgresRepository(db),
		"",
	)
	inventoryService := inventory.NewService(
		inventory.NewPostgresRepository(db),
		subscriberService,
		"",
	)
	producer, err := redpanda.NewProducer(ctx, redpanda.Config{Brokers: cfg.Redpanda.Brokers})
	if err != nil {
		log.Error("failed to connect redpanda", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer producer.Close()
	inventoryService.SetDeliveryProducer(producer)

	interval := cfg.TickInterval
	if interval <= 0 {
		interval = time.Minute
	}
	log.Info("scheduler started", slog.Duration("tick_interval", interval))
	runTick(ctx, log, inventoryService)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("scheduler stopped")
			return
		case <-ticker.C:
			runTick(ctx, log, inventoryService)
		}
	}
}

func runTick(
	ctx context.Context,
	log *slog.Logger,
	service *inventory.Service,
) {
	result, err := service.ProcessDueSchedules(ctx, time.Now().UTC())
	if err != nil {
		log.Error("schedule tick failed", slog.String("error", err.Error()))
		return
	}
	if result.CreatedRuns == 0 && result.Completed == 0 && result.Failed == 0 {
		return
	}
	log.Info(
		"schedule tick completed",
		slog.Int("created_runs", result.CreatedRuns),
		slog.Int("completed", result.Completed),
		slog.Int("failed", result.Failed),
	)
}
