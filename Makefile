GOFLAGS ?= -buildvcs=false
GOCACHE ?= /tmp/push-booster-go-cache
GOMODCACHE ?= /tmp/push-booster-go-mod-cache

POSTGRES_DATABASE_URL ?= postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable
CLICKHOUSE_URL ?= http://localhost:8123
CLICKHOUSE_DATABASE ?= push_booster
CLICKHOUSE_USER ?= push_booster
CLICKHOUSE_PASSWORD ?= push_booster

.PHONY: infra-up infra-down admin-api public-api payload-api sender scheduler admin-frontend frontend-lint frontend-build migrate-up migrate-status migrate-down migrate-clickhouse dev-seed vapid-keys test build-admin-api build-public-api build-payload-api build-sender build-scheduler build-migrator build-clickhouse-migrator

infra-up:
	docker compose -f deploy/docker-compose.yml up -d

infra-down:
	docker compose -f deploy/docker-compose.yml down

admin-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/admin-api

public-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/public-api

payload-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/payload-api

sender:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/sender

scheduler:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/scheduler

admin-frontend:
	npm --prefix apps/admin-frontend run dev

frontend-lint:
	npm --prefix apps/admin-frontend run lint

frontend-build:
	npm --prefix apps/admin-frontend run build

migrate-up:
	POSTGRES_DATABASE_URL=$(POSTGRES_DATABASE_URL) GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/migrator up

migrate-status:
	POSTGRES_DATABASE_URL=$(POSTGRES_DATABASE_URL) GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/migrator status

migrate-down:
	POSTGRES_DATABASE_URL=$(POSTGRES_DATABASE_URL) GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/migrator down

migrate-clickhouse:
	CLICKHOUSE_URL=$(CLICKHOUSE_URL) CLICKHOUSE_DATABASE=$(CLICKHOUSE_DATABASE) CLICKHOUSE_USER=$(CLICKHOUSE_USER) CLICKHOUSE_PASSWORD=$(CLICKHOUSE_PASSWORD) GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/clickhouse-migrator

dev-seed:
	POSTGRES_DATABASE_URL=$(POSTGRES_DATABASE_URL) CLICKHOUSE_URL=$(CLICKHOUSE_URL) CLICKHOUSE_DATABASE=$(CLICKHOUSE_DATABASE) CLICKHOUSE_USER=$(CLICKHOUSE_USER) CLICKHOUSE_PASSWORD=$(CLICKHOUSE_PASSWORD) GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./apps/dev-seed

vapid-keys:
	scripts/generate-vapid-keys.sh

test:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./apps/admin-api ./apps/public-api ./apps/payload-api ./apps/sender ./apps/scheduler ./apps/migrator ./apps/clickhouse-migrator ./packages/go/...

build-admin-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/admin-api ./apps/admin-api

build-public-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/public-api ./apps/public-api

build-payload-api:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/payload-api ./apps/payload-api

build-sender:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/sender ./apps/sender

build-scheduler:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/scheduler ./apps/scheduler

build-migrator:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/migrator ./apps/migrator

build-clickhouse-migrator:
	GOFLAGS=$(GOFLAGS) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o .bin/clickhouse-migrator ./apps/clickhouse-migrator
