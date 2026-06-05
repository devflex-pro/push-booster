package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type AdminAPI struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	PostgresURL       string
	ClickHouse        ClickHouse
	Redpanda          Redpanda
	PublicURL         string
	Auth              Auth
	CreativeProvider  CreativeProviderSync
	LogLevel          string
}

type PublicAPI struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	PostgresURL       string
	ClickHouse        ClickHouse
	Redis             Redis
	VAPIDPublicKey    string
	PublicURL         string
	PayloadURL        string
	EventIDTTL        time.Duration
	LogLevel          string
}

type PayloadAPI struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	PostgresURL       string
	ClickHouse        ClickHouse
	Redis             Redis
	TriggerTTL        time.Duration
	LogLevel          string
}

type Sender struct {
	Redpanda          Redpanda
	PostgresURL       string
	ClickHouse        ClickHouse
	Redis             Redis
	ConsumerGroup     string
	TriggerTTL        time.Duration
	WebPushSubject    string
	WebPushTTLSeconds int
	MaxAttempts       int
	Concurrency       int
	ProviderRateLimit int
	LogLevel          string
}

type Scheduler struct {
	PostgresURL  string
	ClickHouse   ClickHouse
	Redpanda     Redpanda
	TickInterval time.Duration
	LogLevel     string
}

type Auth struct {
	AdminEmail         string
	JWTSecret          string
	JWTIssuer          string
	JWTAudience        string
	OTPTTL             time.Duration
	OTPRateLimit       time.Duration
	SessionTTL         time.Duration
	Env                string
	DevReturnOTP       bool
	EmailFrom          string
	ResendAPIKey       string
	ResendAPIURL       string
	ResendMaxAttempts  int
	ResendRetryBackoff time.Duration
}

type CreativeProviderSync struct {
	Timeout               time.Duration
	MaxBodyBytes          int64
	MaxAttempts           int
	RetryBackoff          time.Duration
	AllowPrivateFetchURLs bool
}

type PostgresMigrations struct {
	DatabaseURL   string
	MigrationsDir string
	LogLevel      string
}

type ClickHouse struct {
	URL      string
	Database string
	Username string
	Password string
}

type Redis struct {
	Addr     string
	Password string
	DB       int
	Timeout  time.Duration
}

type Redpanda struct {
	Brokers []string
}

type ClickHouseMigrations struct {
	ClickHouse    ClickHouse
	MigrationsDir string
	LogLevel      string
}

func LoadAdminAPI() AdminAPI {
	return AdminAPI{
		Addr:              String("ADMIN_API_ADDR", ":8080"),
		ReadHeaderTimeout: Duration("ADMIN_API_READ_HEADER_TIMEOUT", 5*time.Second),
		ShutdownTimeout:   Duration("ADMIN_API_SHUTDOWN_TIMEOUT", 10*time.Second),
		PostgresURL:       String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		ClickHouse:        LoadClickHouse(),
		Redpanda:          LoadRedpanda(),
		PublicURL:         String("PUBLIC_API_BASE_URL", ""),
		Auth:              LoadAuth(),
		CreativeProvider:  LoadCreativeProviderSync(),
		LogLevel:          String("LOG_LEVEL", "info"),
	}
}

func LoadPublicAPI() PublicAPI {
	return PublicAPI{
		Addr:              String("PUBLIC_API_ADDR", ":8082"),
		ReadHeaderTimeout: Duration("PUBLIC_API_READ_HEADER_TIMEOUT", 5*time.Second),
		ShutdownTimeout:   Duration("PUBLIC_API_SHUTDOWN_TIMEOUT", 10*time.Second),
		PostgresURL:       String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		ClickHouse:        LoadClickHouse(),
		Redis:             LoadRedis(),
		VAPIDPublicKey:    String("VAPID_PUBLIC_KEY", ""),
		PublicURL:         String("PUBLIC_API_BASE_URL", ""),
		PayloadURL:        String("PAYLOAD_API_BASE_URL", ""),
		EventIDTTL:        Duration("PUBLIC_EVENT_ID_TTL", 24*time.Hour),
		LogLevel:          String("LOG_LEVEL", "info"),
	}
}

func LoadPayloadAPI() PayloadAPI {
	return PayloadAPI{
		Addr:              String("PAYLOAD_API_ADDR", ":8083"),
		ReadHeaderTimeout: Duration("PAYLOAD_API_READ_HEADER_TIMEOUT", 5*time.Second),
		ShutdownTimeout:   Duration("PAYLOAD_API_SHUTDOWN_TIMEOUT", 10*time.Second),
		PostgresURL:       String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		ClickHouse:        LoadClickHouse(),
		Redis:             LoadRedis(),
		TriggerTTL:        Duration("PAYLOAD_TRIGGER_TTL", 5*time.Minute),
		LogLevel:          String("LOG_LEVEL", "info"),
	}
}

func LoadSender() Sender {
	return Sender{
		Redpanda:          LoadRedpanda(),
		PostgresURL:       String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		ClickHouse:        LoadClickHouse(),
		Redis:             LoadRedis(),
		ConsumerGroup:     String("SENDER_CONSUMER_GROUP", "push_booster-sender"),
		TriggerTTL:        Duration("SENDER_TRIGGER_TTL", 5*time.Minute),
		WebPushSubject:    String("SENDER_WEB_PUSH_SUBJECT", "mailto:admin@example.com"),
		WebPushTTLSeconds: Int("SENDER_WEB_PUSH_TTL_SECONDS", 60),
		MaxAttempts:       Int("SENDER_MAX_ATTEMPTS", 3),
		Concurrency:       Int("SENDER_CONCURRENCY", 8),
		ProviderRateLimit: Int("SENDER_PROVIDER_RATE_LIMIT_PER_SECOND", 50),
		LogLevel:          String("LOG_LEVEL", "info"),
	}
}

func LoadScheduler() Scheduler {
	return Scheduler{
		PostgresURL:  String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		ClickHouse:   LoadClickHouse(),
		Redpanda:     LoadRedpanda(),
		TickInterval: Duration("SCHEDULER_TICK_INTERVAL", time.Minute),
		LogLevel:     String("LOG_LEVEL", "info"),
	}
}

func LoadClickHouse() ClickHouse {
	return ClickHouse{
		URL:      String("CLICKHOUSE_URL", "http://localhost:8123"),
		Database: String("CLICKHOUSE_DATABASE", "push_booster"),
		Username: String("CLICKHOUSE_USER", "push_booster"),
		Password: String("CLICKHOUSE_PASSWORD", "push_booster"),
	}
}

func LoadRedis() Redis {
	return Redis{
		Addr:     String("REDIS_ADDR", "localhost:6379"),
		Password: String("REDIS_PASSWORD", ""),
		DB:       Int("REDIS_DB", 0),
		Timeout:  Duration("REDIS_TIMEOUT", 5*time.Second),
	}
}

func LoadRedpanda() Redpanda {
	return Redpanda{
		Brokers: CSV("REDPANDA_BROKERS", []string{"localhost:19092", "localhost:19093", "localhost:19094"}),
	}
}

func LoadAuth() Auth {
	return Auth{
		AdminEmail:         String("AUTH_ADMIN_EMAIL", "admin@example.com"),
		JWTSecret:          String("AUTH_JWT_SECRET", "change-me-local-dev-secret"),
		JWTIssuer:          String("AUTH_JWT_ISSUER", "push_booster-admin-api"),
		JWTAudience:        String("AUTH_JWT_AUDIENCE", "push_booster-admin"),
		OTPTTL:             DurationUnit("AUTH_OTP_TTL_SECONDS", time.Second, 10*time.Minute),
		OTPRateLimit:       DurationUnit("AUTH_OTP_RATE_LIMIT_MINUTES", time.Minute, time.Minute),
		SessionTTL:         DurationUnit("AUTH_SESSION_TTL_HOURS", time.Hour, 24*time.Hour),
		Env:                String("AUTH_ENV", "local"),
		DevReturnOTP:       Bool("AUTH_DEV_RETURN_OTP", false),
		EmailFrom:          String("AUTH_EMAIL_FROM", "noreply@example.com"),
		ResendAPIKey:       String("AUTH_RESEND_API_KEY", ""),
		ResendAPIURL:       String("AUTH_RESEND_API_URL", "https://api.resend.com/emails"),
		ResendMaxAttempts:  Int("AUTH_RESEND_MAX_ATTEMPTS", 3),
		ResendRetryBackoff: DurationUnit("AUTH_RESEND_RETRY_BACKOFF_MS", time.Millisecond, 500*time.Millisecond),
	}
}

func LoadCreativeProviderSync() CreativeProviderSync {
	return CreativeProviderSync{
		Timeout:               Duration("CREATIVE_PROVIDER_FETCH_TIMEOUT", 15*time.Second),
		MaxBodyBytes:          Int64("CREATIVE_PROVIDER_MAX_BODY_BYTES", 2*1024*1024),
		MaxAttempts:           Int("CREATIVE_PROVIDER_MAX_ATTEMPTS", 2),
		RetryBackoff:          DurationUnit("CREATIVE_PROVIDER_RETRY_BACKOFF_MS", time.Millisecond, 500*time.Millisecond),
		AllowPrivateFetchURLs: Bool("CREATIVE_PROVIDER_ALLOW_PRIVATE_FETCH_URLS", false),
	}
}

func (a Auth) LocalEnv() bool {
	env := strings.ToLower(strings.TrimSpace(a.Env))
	return env == "local" || env == "dev" || env == "test"
}

func (a Auth) ResendConfigured() bool {
	return strings.TrimSpace(a.ResendAPIKey) != ""
}

func LoadPostgresMigrations() PostgresMigrations {
	return PostgresMigrations{
		DatabaseURL:   String("POSTGRES_DATABASE_URL", "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"),
		MigrationsDir: String("POSTGRES_MIGRATIONS_DIR", "packages/go/migrations/postgres"),
		LogLevel:      String("LOG_LEVEL", "info"),
	}
}

func LoadClickHouseMigrations() ClickHouseMigrations {
	return ClickHouseMigrations{
		ClickHouse:    LoadClickHouse(),
		MigrationsDir: String("CLICKHOUSE_MIGRATIONS_DIR", "packages/go/migrations/clickhouse"),
		LogLevel:      String("LOG_LEVEL", "info"),
	}
}

func Int(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func Int64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func Bool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}

func String(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func CSV(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	items := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func Duration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}

func DurationUnit(key string, unit time.Duration, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}

	amount, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return time.Duration(amount) * unit
}
