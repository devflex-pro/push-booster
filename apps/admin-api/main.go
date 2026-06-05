package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/devflex-pro/push-booster/packages/go/auth"
	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/httpserver"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/logger"
	"github.com/devflex-pro/push-booster/packages/go/postbacks"
	"github.com/devflex-pro/push-booster/packages/go/postgres"
	"github.com/devflex-pro/push-booster/packages/go/redpanda"
	"github.com/devflex-pro/push-booster/packages/go/reports"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

func main() {
	cfg := config.LoadAdminAPI()
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

	authService := auth.NewServiceWithDelivery(
		auth.NewPostgresRepository(db),
		authDelivery(log, cfg.Auth),
		auth.Config{
			AdminEmail:   cfg.Auth.AdminEmail,
			JWTSecret:    cfg.Auth.JWTSecret,
			JWTIssuer:    cfg.Auth.JWTIssuer,
			JWTAudience:  cfg.Auth.JWTAudience,
			OTPTTL:       cfg.Auth.OTPTTL,
			OTPRateLimit: cfg.Auth.OTPRateLimit,
			SessionTTL:   cfg.Auth.SessionTTL,
			DevReturnOTP: cfg.Auth.DevReturnOTP,
		},
	)
	inventoryRepo := inventory.NewPostgresRepository(db)
	clickHouseClient := clickhouse.NewClient(clickhouse.Config{
		URL:      cfg.ClickHouse.URL,
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	subscriberRepo := subscribers.NewClickHouseRepository(clickHouseClient)
	postbackRepo := postbacks.NewPostgresRepository(db)
	postbackEvents := postbacks.NewClickHouseRepository(clickHouseClient)
	postbackService := postbacks.NewService(postbackRepo, postbackEvents)
	reportCostRepo := reports.NewPostgresRepository(db)
	reportMetricRepo := reports.NewClickHouseRepository(clickHouseClient)
	reportService := reports.NewService(reportCostRepo, reportMetricRepo)
	subscriberService := subscribers.NewService(
		subscriberRepo,
		inventoryRepo,
		"",
	)
	inventoryService := inventory.NewService(inventoryRepo, subscriberService, cfg.PublicURL)
	inventoryService.SetCreativeProviderFetcher(inventory.HTTPProviderFetcher{
		Client: &http.Client{Timeout: cfg.CreativeProvider.Timeout},
		Config: inventory.ProviderFetchConfig{
			MaxBodyBytes:          cfg.CreativeProvider.MaxBodyBytes,
			MaxAttempts:           cfg.CreativeProvider.MaxAttempts,
			RetryBackoff:          cfg.CreativeProvider.RetryBackoff,
			AllowPrivateFetchURLs: cfg.CreativeProvider.AllowPrivateFetchURLs,
		},
	})
	deliveryProducer, err := redpanda.NewProducer(
		context.Background(),
		redpanda.Config{Brokers: cfg.Redpanda.Brokers},
	)
	if err != nil {
		log.Error(
			"failed to connect redpanda",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer deliveryProducer.Close()
	inventoryService.SetDeliveryProducer(deliveryProducer)

	server := &http.Server{
		Addr: cfg.Addr,
		Handler: httpserver.NewAdminRouter(httpserver.Options{
			AuthHandler:       auth.NewHandler(log, authService),
			InventoryHandler:  inventory.NewHandler(log, inventoryService),
			PostbackHandler:   postbacks.NewHandler(log, postbackService),
			ReportHandler:     reports.NewHandler(log, reportService),
			SubscriberHandler: subscribers.NewHandler(log, subscriberService),
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
			"admin-api listening",
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
				"admin-api shutdown failed",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(
				"admin-api failed",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}
}

func authDelivery(log *slog.Logger, cfg config.Auth) auth.OTPDelivery {
	if cfg.ResendConfigured() {
		return auth.NewResendDelivery(auth.ResendConfig{
			APIKey:       cfg.ResendAPIKey,
			APIURL:       cfg.ResendAPIURL,
			From:         cfg.EmailFrom,
			MaxAttempts:  cfg.ResendMaxAttempts,
			RetryBackoff: cfg.ResendRetryBackoff,
			Logger:       log,
		})
	}
	if cfg.LocalEnv() {
		return auth.NewLogDelivery(log)
	}
	return nil
}
