package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/logger"
)

func main() {
	cfg := config.LoadPostgresMigrations()
	log := logger.New(cfg.LogLevel)

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Error("open postgres failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("close postgres failed", slog.String("error", err.Error()))
		}
	}()

	if err := db.Ping(); err != nil {
		log.Error("ping postgres failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Error("set migration dialect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := goose.RunContext(
		context.Background(),
		command,
		db,
		cfg.MigrationsDir,
	); err != nil {
		log.Error(
			"migration command failed",
			slog.String("command", command),
			slog.String("dir", cfg.MigrationsDir),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	fmt.Printf("migrations %s complete\n", command)
}
