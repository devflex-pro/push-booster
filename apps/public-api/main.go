package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/httpserver"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/logger"
	"github.com/devflex-pro/push-booster/packages/go/postbacks"
	"github.com/devflex-pro/push-booster/packages/go/postgres"
	"github.com/devflex-pro/push-booster/packages/go/redis"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

func main() {
	cfg := config.LoadPublicAPI()
	log := logger.New(cfg.LogLevel)

	db, err := postgres.Connect(context.Background(), cfg.PostgresURL)
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
	inventoryRepo := inventory.NewPostgresRepository(db)
	subscriberRepo := subscribers.NewClickHouseRepository(clickHouseClient)
	postbackRepo := postbacks.NewPostgresRepository(db)
	postbackEvents := postbacks.NewClickHouseRepository(clickHouseClient)
	postbackService := postbacks.NewService(postbackRepo, postbackEvents)
	redisClient := redis.NewClient(redis.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		Timeout:  cfg.Redis.Timeout,
	})
	subscriberService := subscribers.NewServiceWithConfig(subscriberRepo, inventoryRepo, subscribers.Config{
		FallbackVAPIDPublicKey: cfg.VAPIDPublicKey,
		PublicURL:              cfg.PublicURL,
		PayloadURL:             cfg.PayloadURL,
		EventIDs:               subscribers.NewRedisEventIDStore(redisClient, cfg.EventIDTTL),
	})

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: httpserver.NewPublicRouter(httpserver.PublicOptions{
			SubscriberHandler: subscribers.NewHandler(log, subscriberService),
			PostbackHandler:   postbacks.NewHandler(log, postbackService),
		}),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info(
			"public-api listening",
			slog.String("addr", cfg.Addr),
		)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error(
				"public-api shutdown failed",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(
				"public-api failed",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}
}
