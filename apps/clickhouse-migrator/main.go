package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/config"
	"github.com/devflex-pro/push-booster/packages/go/logger"
)

func main() {
	cfg := config.LoadClickHouseMigrations()
	log := logger.New(cfg.LogLevel)

	files, err := filepath.Glob(filepath.Join(cfg.MigrationsDir, "*.sql"))
	if err != nil {
		log.Error("list clickhouse migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	sort.Strings(files)

	client := clickhouse.NewClient(clickhouse.Config{
		URL:      cfg.ClickHouse.URL,
		Database: "",
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})

	ctx := context.Background()
	for _, file := range files {
		query, err := os.ReadFile(file)
		if err != nil {
			log.Error("read clickhouse migration failed", slog.String("file", file), slog.String("error", err.Error()))
			os.Exit(1)
		}
		for _, statement := range splitStatements(string(query)) {
			if err := client.Exec(ctx, statement); err != nil {
				log.Error("run clickhouse migration failed", slog.String("file", file), slog.String("error", err.Error()))
				os.Exit(1)
			}
		}
	}

	fmt.Printf("clickhouse migrations complete: %d files\n", len(files))
}

func splitStatements(query string) []string {
	parts := strings.Split(query, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
