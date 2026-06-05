package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/jackc/pgx/v5"
)

const (
	defaultPostgresURL    = "postgres://push_booster:push_booster@localhost:5432/push_booster?sslmode=disable"
	defaultClickHouseURL  = "http://localhost:8123"
	defaultClickHouseDB   = "push_booster"
	defaultClickHouseUser = "push_booster"
	defaultClickHousePass = "push_booster"
)

type publisherSeed struct {
	ID   string
	Name string
}

type sourceSeed struct {
	ID          string
	PublisherID string
	Name        string
	Domain      string
}

type campaignSeed struct {
	ID            string
	PublisherID   string
	SourceID      string
	SourceIDs     []string
	AudienceScope string
	Name          string
	Status        string
	DailyCap      int
	TotalCap      int
}

type creativeSeed struct {
	ID       string
	Campaign campaignSeed
	Title    string
	Body     string
	URL      string
	Status   string
	Provider bool
}

type launchSeed struct {
	ID       string
	Campaign campaignSeed
	Day      int
}

type subscriberSeed struct {
	ID       string
	Source   sourceSeed
	Endpoint string
	Country  string
	Language string
	Timezone string
	Device   string
	OS       string
	Browser  string
	Day      int
}

const seedSubscriberCount = 120000

var publishers = []publisherSeed{
	{ID: "10000000-0000-0000-0000-000000000001", Name: "Seed Media Network"},
	{ID: "10000000-0000-0000-0000-000000000002", Name: "Seed Utility Apps"},
}

var sources = []sourceSeed{
	{ID: "20000000-0000-0000-0000-000000000001", PublisherID: publishers[0].ID, Name: "Seed News Push", Domain: "news.seed.local"},
	{ID: "20000000-0000-0000-0000-000000000002", PublisherID: publishers[0].ID, Name: "Seed Finance Push", Domain: "finance.seed.local"},
	{ID: "20000000-0000-0000-0000-000000000003", PublisherID: publishers[1].ID, Name: "Seed Tools Push", Domain: "tools.seed.local"},
}

var campaigns = []campaignSeed{
	{ID: "30000000-0000-0000-0000-000000000001", SourceIDs: []string{sources[0].ID, sources[2].ID}, AudienceScope: "selected_sources", Name: "Seed iOS Sweepstakes", Status: "active", DailyCap: 2, TotalCap: 8},
	{ID: "30000000-0000-0000-0000-000000000002", AudienceScope: "all", Name: "Seed Android Antivirus", Status: "active", DailyCap: 1, TotalCap: 5},
	{ID: "30000000-0000-0000-0000-000000000003", SourceIDs: []string{sources[1].ID}, AudienceScope: "selected_sources", Name: "Seed Loan Leadgen", Status: "paused", DailyCap: 2, TotalCap: 10},
	{ID: "30000000-0000-0000-0000-000000000004", SourceIDs: []string{sources[0].ID, sources[1].ID, sources[2].ID}, AudienceScope: "selected_sources", Name: "Seed VPN Trial", Status: "active", DailyCap: 3, TotalCap: 12},
	{ID: "30000000-0000-0000-0000-000000000005", SourceIDs: []string{sources[2].ID}, AudienceScope: "selected_sources", Name: "Seed Cleaner App", Status: "draft", DailyCap: 1, TotalCap: 4},
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	postgresURL := env("POSTGRES_DATABASE_URL", defaultPostgresURL)
	conn, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		fatal("connect postgres", err)
	}
	defer func() {
		if err := conn.Close(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "close postgres: %v\n", err)
		}
	}()

	ch := clickhouse.NewClient(clickhouse.Config{
		URL:      env("CLICKHOUSE_URL", defaultClickHouseURL),
		Database: env("CLICKHOUSE_DATABASE", defaultClickHouseDB),
		Username: env("CLICKHOUSE_USER", defaultClickHouseUser),
		Password: env("CLICKHOUSE_PASSWORD", defaultClickHousePass),
	})

	if err := cleanPostgres(ctx, conn); err != nil {
		fatal("clean postgres seed data", err)
	}
	if err := seedPostgres(ctx, conn); err != nil {
		fatal("seed postgres", err)
	}
	if err := cleanClickHouse(ctx, ch); err != nil {
		fatal("clean clickhouse seed data", err)
	}
	if err := seedClickHouse(ctx, ch); err != nil {
		fatal("seed clickhouse", err)
	}

	fmt.Println("dev seed complete")
	fmt.Printf("publishers=%d sources=%d campaigns=%d creatives=%d subscribers=%d launches=%d days=%d\n",
		len(publishers),
		len(sources),
		len(campaigns),
		len(creatives()),
		len(subscribers()),
		len(launches()),
		seedDayCount(),
	)
}

func cleanPostgres(ctx context.Context, conn *pgx.Conn) error {
	statements := []string{
		"TRUNCATE creative_sync_logs, creative_provider_configs, creatives, campaign_launches, campaign_sources, campaigns, cost_entries, postback_configs, sources, publishers RESTART IDENTITY CASCADE",
		"UPDATE vapid_keys SET status = 'revoked', updated_at = now() WHERE status <> 'revoked'",
	}
	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func seedPostgres(ctx context.Context, conn *pgx.Conn) error {
	for _, publisher := range publishers {
		_, err := conn.Exec(
			ctx,
			`INSERT INTO publishers (id, name, status, created_at, updated_at)
VALUES ($1, $2, 'active', now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, updated_at = now()`,
			publisher.ID,
			publisher.Name,
		)
		if err != nil {
			return fmt.Errorf("insert publisher %s: %w", publisher.ID, err)
		}
	}
	for _, source := range sources {
		_, err := conn.Exec(
			ctx,
			`INSERT INTO sources (id, publisher_id, name, domain, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, 'active', now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, domain = EXCLUDED.domain, status = EXCLUDED.status, updated_at = now()`,
			source.ID,
			source.PublisherID,
			source.Name,
			source.Domain,
		)
		if err != nil {
			return fmt.Errorf("insert source %s: %w", source.ID, err)
		}
	}
	for _, campaign := range campaigns {
		sourceIDs := campaignSourceIDs(campaign)
		sourceID := ""
		publisherID := ""
		if campaign.AudienceScope == "selected_sources" {
			sourceID = sourceIDs[0]
			publisherID = publisherIDForSource(sourceID)
		}
		_, err := conn.Exec(
			ctx,
			`INSERT INTO campaigns (
    id, publisher_id, source_id, audience_scope, name, status,
    daily_cap_per_subscription, total_cap_per_subscription, targeting_rules,
    created_at, updated_at
)
VALUES ($1, nullif($2, '')::uuid, nullif($3, '')::uuid, $4, $5, $6, $7, $8, $9::jsonb, now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET
    publisher_id = EXCLUDED.publisher_id,
    source_id = EXCLUDED.source_id,
    audience_scope = EXCLUDED.audience_scope,
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    daily_cap_per_subscription = EXCLUDED.daily_cap_per_subscription,
    total_cap_per_subscription = EXCLUDED.total_cap_per_subscription,
    targeting_rules = EXCLUDED.targeting_rules,
    updated_at = now()`,
			campaign.ID,
			publisherID,
			sourceID,
			campaign.AudienceScope,
			campaign.Name,
			campaign.Status,
			campaign.DailyCap,
			campaign.TotalCap,
			targetingRules(campaign),
		)
		if err != nil {
			return fmt.Errorf("insert campaign %s: %w", campaign.ID, err)
		}
		if campaign.AudienceScope == "selected_sources" {
			for _, campaignSourceID := range sourceIDs {
				_, err := conn.Exec(
					ctx,
					`INSERT INTO campaign_sources (campaign_id, source_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`,
					campaign.ID,
					campaignSourceID,
				)
				if err != nil {
					return fmt.Errorf("insert campaign source %s/%s: %w", campaign.ID, campaignSourceID, err)
				}
			}
		}
	}
	if err := seedProviderConfigs(ctx, conn); err != nil {
		return err
	}
	for _, creative := range creatives() {
		providerConfigRef := ""
		providerName := ""
		providerExternalID := ""
		sourceType := "manual"
		rawPayload := "{}"
		if creative.Provider {
			providerConfigRef = providerConfigID(creative.Campaign.ID)
			providerName = "seed-provider"
			providerExternalID = "seed-" + creative.ID
			sourceType = "provider_api"
			rawPayload = fmt.Sprintf(`{"seed":true,"external_id":%q}`, providerExternalID)
		}
		_, err := conn.Exec(
			ctx,
			`INSERT INTO creatives (
    id, campaign_id, title, body, url, icon, status,
    daily_cap_per_subscription, total_cap_per_subscription,
    source_type, provider_config_id, provider_name, provider_external_id,
    raw_provider_payload, last_synced_at, sync_status, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 1, 6, $8, nullif($9, '')::uuid, $10, $11, $12::jsonb, now(), 'synced', now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    body = EXCLUDED.body,
    url = EXCLUDED.url,
    status = EXCLUDED.status,
    source_type = EXCLUDED.source_type,
    provider_config_id = EXCLUDED.provider_config_id,
    provider_name = EXCLUDED.provider_name,
    provider_external_id = EXCLUDED.provider_external_id,
    raw_provider_payload = EXCLUDED.raw_provider_payload,
    last_synced_at = EXCLUDED.last_synced_at,
    sync_status = EXCLUDED.sync_status,
    updated_at = now()`,
			creative.ID,
			creative.Campaign.ID,
			creative.Title,
			creative.Body,
			creative.URL,
			"https://example.com/icon.png",
			creative.Status,
			sourceType,
			providerConfigRef,
			providerName,
			providerExternalID,
			rawPayload,
		)
		if err != nil {
			return fmt.Errorf("insert creative %s: %w", creative.ID, err)
		}
	}
	for _, launch := range launches() {
		createdAt := time.Now().UTC().AddDate(0, 0, -launch.Day)
		audience := int64(650 + launch.Day*130)
		_, err := conn.Exec(
			ctx,
			`INSERT INTO campaign_launches (
    id, campaign_id, status, audience_total, processed_total, error_message,
    enqueue_status, enqueued_total, enqueue_error, enqueue_started_at,
    enqueue_completed_at, created_at, updated_at, completed_at
)
VALUES ($1, $2, 'completed', $3, $3, '', 'completed', $3, '', $4, $5, $4, $5, $5)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    audience_total = EXCLUDED.audience_total,
    processed_total = EXCLUDED.processed_total,
    enqueue_status = EXCLUDED.enqueue_status,
    enqueued_total = EXCLUDED.enqueued_total,
    updated_at = EXCLUDED.updated_at,
    completed_at = EXCLUDED.completed_at`,
			launch.ID,
			launch.Campaign.ID,
			audience,
			createdAt,
			createdAt.Add(12*time.Minute),
		)
		if err != nil {
			return fmt.Errorf("insert launch %s: %w", launch.ID, err)
		}
	}
	return seedCostsAndPostbacks(ctx, conn)
}

func seedProviderConfigs(ctx context.Context, conn *pgx.Conn) error {
	for _, campaign := range campaigns {
		id := providerConfigID(campaign.ID)
		_, err := conn.Exec(
			ctx,
			`INSERT INTO creative_provider_configs (
    id, campaign_id, name, provider_name, fetch_url, request_headers,
    status, last_sync_at, created_at, updated_at
)
VALUES ($1, $2, $3, 'seed-provider', $4, '{}'::jsonb, 'active', now(), now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    fetch_url = EXCLUDED.fetch_url,
    status = EXCLUDED.status,
    last_sync_at = EXCLUDED.last_sync_at,
    updated_at = now()`,
			id,
			campaign.ID,
			"Seed provider for "+campaign.Name,
			"https://provider.seed.local/"+campaign.ID+"/creatives.json",
		)
		if err != nil {
			return fmt.Errorf("insert provider config %s: %w", id, err)
		}
		_, err = conn.Exec(
			ctx,
			`INSERT INTO creative_sync_logs (
    id, provider_config_id, campaign_id, status, fetched_total,
    upserted_total, error_message, started_at, completed_at
)
VALUES ($1, $2, $3, 'completed', 2, 2, '', now() - interval '2 hours', now() - interval '115 minutes')
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    fetched_total = EXCLUDED.fetched_total,
    upserted_total = EXCLUDED.upserted_total,
    completed_at = EXCLUDED.completed_at`,
			syncLogID(campaign.ID),
			id,
			campaign.ID,
		)
		if err != nil {
			return fmt.Errorf("insert sync log %s: %w", campaign.ID, err)
		}
	}
	return nil
}

func seedCostsAndPostbacks(ctx context.Context, conn *pgx.Conn) error {
	for _, source := range sources {
		_, err := conn.Exec(
			ctx,
			`INSERT INTO postback_configs (id, name, source_id, token, status, default_currency, created_at, updated_at)
VALUES ($1, $2, $3, 'seed-token', 'active', 'USD', now() - interval '7 days', now())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, updated_at = now()`,
			postbackConfigID(source.ID),
			"Seed postback "+source.Name,
			source.ID,
		)
		if err != nil {
			return fmt.Errorf("insert postback config %s: %w", source.ID, err)
		}
	}
	for day := seedDayCount() - 1; day >= 0; day-- {
		costDate := time.Now().UTC().AddDate(0, 0, -day).Format("2006-01-02")
		for campaignIndex, campaign := range campaigns {
			for _, sourceID := range campaignSourceIDs(campaign) {
				amount := seedCostAmount(day, campaignIndex, sourceID)
				_, err := conn.Exec(
					ctx,
					`INSERT INTO cost_entries (
    cost_date, publisher_id, source_id, campaign_id, amount, currency, note, created_at
)
VALUES ($1, $2, $3, $4, $5, 'USD', $6, now())`,
					costDate,
					publisherIDForSource(sourceID),
					sourceID,
					campaign.ID,
					amount,
					"seed campaign source spend",
				)
				if err != nil {
					return fmt.Errorf("insert cost %s day %d: %w", campaign.ID, day, err)
				}
			}
		}
		for publisherIndex, publisher := range publishers {
			_, err := conn.Exec(
				ctx,
				`INSERT INTO cost_entries (
    cost_date, publisher_id, amount, currency, note, created_at
)
VALUES ($1, $2, $3, 'USD', 'seed publisher spend', now())`,
				costDate,
				publisher.ID,
				seedPublisherCostAmount(day, publisherIndex),
			)
			if err != nil {
				return fmt.Errorf("insert publisher cost %s day %d: %w", publisher.ID, day, err)
			}
		}
	}
	return nil
}

func cleanClickHouse(ctx context.Context, ch *clickhouse.Client) error {
	tables := []string{
		"campaign_audience",
		"creative_exposures",
		"payload_decisions",
		"postback_events",
		"push_events",
		"subscriber_events",
		"subscribers",
	}
	for _, table := range tables {
		if err := ch.Exec(ctx, "TRUNCATE TABLE "+table); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	return nil
}

func seedClickHouse(ctx context.Context, ch *clickhouse.Client) error {
	subs := subscribers()
	if err := insertJSONRows(ctx, ch, "subscribers", subscriberRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "subscriber_events", subscriberEventRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "payload_decisions", payloadDecisionRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "creative_exposures", creativeExposureRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "campaign_audience", audienceRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "push_events", pushEventRows(subs)); err != nil {
		return err
	}
	if err := insertJSONRows(ctx, ch, "postback_events", postbackRows(subs)); err != nil {
		return err
	}
	return nil
}

func insertJSONRows(ctx context.Context, ch *clickhouse.Client, table string, rows []map[string]any) error {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		payload, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshal %s row: %w", table, err)
		}
		lines = append(lines, string(payload))
	}
	if len(lines) == 0 {
		return nil
	}
	query := fmt.Sprintf("INSERT INTO %s FORMAT JSONEachRow\n%s", table, strings.Join(lines, "\n"))
	if err := ch.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}
	return nil
}

func subscribers() []subscriberSeed {
	rng := rand.New(rand.NewSource(42))
	dayWeights := seedDayWeights(seedDayCount())
	countries := []string{"US", "GB", "CA", "DE", "BR", "IN"}
	languages := []string{"en", "en", "en", "de", "pt", "hi"}
	timezones := []string{"America/New_York", "Europe/London", "America/Toronto", "Europe/Berlin", "America/Sao_Paulo", "Asia/Kolkata"}
	devices := []string{"mobile", "desktop", "mobile", "tablet"}
	osNames := []string{"Android", "iOS", "Windows", "macOS"}
	browsers := []string{"Chrome", "Safari", "Edge", "Firefox"}
	result := make([]subscriberSeed, 0, seedSubscriberCount)
	for i := range seedSubscriberCount {
		source := sources[i%len(sources)]
		countryIndex := rng.Intn(len(countries))
		result = append(result, subscriberSeed{
			ID:       fmt.Sprintf("50000000-0000-0000-0000-%012d", i+1),
			Source:   source,
			Endpoint: fmt.Sprintf("https://push.seed.local/sub/%d", i+1),
			Country:  countries[countryIndex],
			Language: languages[countryIndex],
			Timezone: timezones[countryIndex],
			Device:   devices[rng.Intn(len(devices))],
			OS:       osNames[rng.Intn(len(osNames))],
			Browser:  browsers[rng.Intn(len(browsers))],
			Day:      weightedDay(i, rng, dayWeights),
		})
	}
	return result
}

func subscriberRows(subs []subscriberSeed) []map[string]any {
	rows := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		at := nowMinus(sub.Day, time.Duration(sub.Day+1)*time.Hour)
		rows = append(rows, map[string]any{
			"source_id":           sub.Source.ID,
			"subscription_id":     sub.ID,
			"endpoint":            sub.Endpoint,
			"p256dh":              "seed-p256dh",
			"auth":                "seed-auth",
			"user_agent":          userAgent(sub),
			"subid":               "seed-subid",
			"channel":             "seed-channel",
			"landing_url":         "https://" + sub.Source.Domain + "/landing",
			"referrer":            "https://traffic.seed.local",
			"ip":                  "127.0.0.1",
			"country":             sub.Country,
			"region":              "Seed Region",
			"city":                "Seed City",
			"timezone":            sub.Timezone,
			"language":            sub.Language,
			"browser_name":        sub.Browser,
			"browser_version":     "120",
			"os_name":             sub.OS,
			"os_version":          "17",
			"device_type":         sub.Device,
			"device_vendor":       "Seed",
			"device_model":        "Seed Model",
			"ua_platform":         sub.OS,
			"ua_platform_version": "17",
			"ua_mobile":           boolToInt(sub.Device == "mobile"),
			"ua_full_version":     "120.0.0",
			"ua_arch":             "arm64",
			"ua_bitness":          "64",
			"subscribed_at":       chTime(at),
		})
	}
	return rows
}

func subscriberEventRows(subs []subscriberSeed) []map[string]any {
	clickedIndexes := seedClickedIndexes(subs)
	rows := make([]map[string]any, 0, len(subs)*2)
	for index, sub := range subs {
		campaign := campaignForSource(sub.Source.ID, index)
		creative := creativesForCampaign(campaign.ID)[index%3]
		base := nowMinus(sub.Day, time.Duration(index%20)*time.Minute)
		rows = append(rows, subscriberEvent(sub, campaign, creative, "", "subscribed", base.Add(-3*time.Hour), index))
		if !seedHasImpression(index) {
			continue
		}
		rows = append(rows, subscriberEvent(sub, campaign, creative, deliveryID(index), "notification_shown", base, index))
		if clickedIndexes[index] {
			rows = append(rows, subscriberEvent(sub, campaign, creative, deliveryID(index), "notification_click", base.Add(2*time.Minute), index))
		}
		if index%5 == 0 {
			rows = append(rows, subscriberEvent(sub, campaign, creative, deliveryID(index), "notification_close", base.Add(90*time.Second), index))
		}
	}
	return rows
}

func subscriberEvent(
	sub subscriberSeed,
	campaign campaignSeed,
	creative creativeSeed,
	deliveryIDValue string,
	eventType string,
	occurredAt time.Time,
	index int,
) map[string]any {
	return map[string]any{
		"source_id":       sub.Source.ID,
		"delivery_id":     deliveryIDValue,
		"campaign_id":     campaign.ID,
		"creative_id":     creative.ID,
		"event_id":        fmt.Sprintf("seed-event-%s-%d", eventType, index),
		"target_url":      creative.URL,
		"subscription_id": sub.ID,
		"endpoint":        sub.Endpoint,
		"event_type":      eventType,
		"user_agent":      userAgent(sub),
		"occurred_at":     chTime(occurredAt),
	}
}

func payloadDecisionRows(subs []subscriberSeed) []map[string]any {
	rows := make([]map[string]any, 0, len(subs))
	for index, sub := range subs {
		campaign := campaignForSource(sub.Source.ID, index)
		creative := creativesForCampaign(campaign.ID)[index%3]
		result := "selected"
		reason := "matched"
		if index%41 == 0 {
			result = "suppressed"
			reason = "campaign_cap_exceeded"
		}
		rows = append(rows, map[string]any{
			"trigger_id":      triggerID(index),
			"subscription_id": sub.ID,
			"source_id":       sub.Source.ID,
			"campaign_id":     campaign.ID,
			"creative_id":     creative.ID,
			"result":          result,
			"reason":          reason,
			"error":           "",
			"occurred_at":     chTime(nowMinus(sub.Day, time.Duration(index%30)*time.Minute)),
		})
	}
	return rows
}

func creativeExposureRows(subs []subscriberSeed) []map[string]any {
	rows := make([]map[string]any, 0, len(subs))
	for index, sub := range subs {
		campaign := campaignForSource(sub.Source.ID, index)
		creative := creativesForCampaign(campaign.ID)[index%3]
		rows = append(rows, map[string]any{
			"source_id":       sub.Source.ID,
			"subscription_id": sub.ID,
			"campaign_id":     campaign.ID,
			"creative_id":     creative.ID,
			"occurred_at":     chTime(nowMinus(sub.Day, time.Duration(index%45)*time.Minute)),
		})
	}
	return rows
}

func audienceRows(subs []subscriberSeed) []map[string]any {
	rows := make([]map[string]any, 0, len(subs))
	launchesByCampaign := launches()
	for index, sub := range subs {
		campaign := campaignForSource(sub.Source.ID, index)
		launch := launchesByCampaign[index%len(launchesByCampaign)]
		if launch.Campaign.ID != campaign.ID {
			launch = launchForCampaign(campaign.ID, index)
		}
		rows = append(rows, map[string]any{
			"launch_id":       launch.ID,
			"campaign_id":     campaign.ID,
			"source_id":       sub.Source.ID,
			"subscription_id": sub.ID,
			"endpoint":        sub.Endpoint,
			"p256dh":          "seed-p256dh",
			"auth":            "seed-auth",
			"shard":           index % 16,
			"selected_at":     chTime(nowMinus(launch.Day, time.Duration(index%10)*time.Minute)),
		})
	}
	return rows
}

func pushEventRows(subs []subscriberSeed) []map[string]any {
	rows := make([]map[string]any, 0, len(subs)*2)
	for index, sub := range subs {
		campaign := campaignForSource(sub.Source.ID, index)
		launch := launchForCampaign(campaign.ID, index)
		base := nowMinus(sub.Day, time.Duration(index%35)*time.Minute)
		rows = append(rows, pushEvent(sub, campaign, launch, index, "sent", "", base))
		if index%11 == 0 {
			rows = append(rows, pushEvent(sub, campaign, launch, index, "retry_enqueued", "temporary_webpush_failure", base.Add(time.Minute)))
		}
		if index%29 == 0 {
			rows = append(rows, pushEvent(sub, campaign, launch, index, "invalid_endpoint", "gone", base.Add(2*time.Minute)))
		}
	}
	return rows
}

func pushEvent(
	sub subscriberSeed,
	campaign campaignSeed,
	launch launchSeed,
	index int,
	eventType string,
	errText string,
	occurredAt time.Time,
) map[string]any {
	return map[string]any{
		"delivery_id":     deliveryID(index),
		"trigger_id":      triggerID(index),
		"launch_id":       launch.ID,
		"campaign_id":     campaign.ID,
		"source_id":       sub.Source.ID,
		"subscription_id": sub.ID,
		"event_type":      eventType,
		"attempt":         1,
		"error":           errText,
		"occurred_at":     chTime(occurredAt),
	}
}

func postbackRows(subs []subscriberSeed) []map[string]any {
	clickedIndexes := seedClickedIndexes(subs)
	clicksByDay := seedClickCountsByDay(subs, clickedIndexes)
	quotasByDay := seedConversionQuotasByDay(clicksByDay)
	seenByDay := make(map[int]int, len(clicksByDay))
	rows := make([]map[string]any, 0, len(subs)/25)
	for index, sub := range subs {
		if !clickedIndexes[index] {
			continue
		}
		seen := seenByDay[sub.Day]
		seenByDay[sub.Day] = seen + 1
		clicks := clicksByDay[sub.Day]
		quota := quotasByDay[sub.Day]
		if quota == 0 || (seen+1)*quota/clicks == seen*quota/clicks {
			continue
		}
		campaign := campaignForSource(sub.Source.ID, index)
		creative := creativesForCampaign(campaign.ID)[index%3]
		payout := seedPayout(index, sub.Day)
		rows = append(rows, map[string]any{
			"postback_config_id": postbackConfigID(sub.Source.ID),
			"dedupe_key":         fmt.Sprintf("seed-conv-%d", index),
			"external_id":        fmt.Sprintf("ext-%d", index),
			"click_id":           deliveryID(index),
			"delivery_id":        deliveryID(index),
			"subscription_id":    sub.ID,
			"source_id":          sub.Source.ID,
			"campaign_id":        campaign.ID,
			"creative_id":        creative.ID,
			"payout":             payout,
			"currency":           "USD",
			"status":             "approved",
			"attribution_status": "matched",
			"raw_payload":        fmt.Sprintf(`{"seed":true,"index":%d}`, index),
			"received_at":        chTime(nowMinus(sub.Day, time.Duration(index%50)*time.Minute)),
		})
	}
	return rows
}

func seedHasImpression(index int) bool {
	return index%17 != 0 && index%23 != 0
}

func seedClickedIndexes(subs []subscriberSeed) map[int]bool {
	impressionsByDay := seedImpressionCountsByDay(subs)
	quotasByDay := seedClickQuotasByDay(impressionsByDay)
	seenByDay := make(map[int]int, len(impressionsByDay))
	clicked := make(map[int]bool, len(subs)/60)
	for index, sub := range subs {
		if !seedHasImpression(index) {
			continue
		}
		seen := seenByDay[sub.Day]
		seenByDay[sub.Day] = seen + 1
		impressions := impressionsByDay[sub.Day]
		quota := quotasByDay[sub.Day]
		if quota > 0 && (seen+1)*quota/impressions != seen*quota/impressions {
			clicked[index] = true
		}
	}
	return clicked
}

func seedImpressionCountsByDay(subs []subscriberSeed) map[int]int {
	counts := make(map[int]int)
	for index, sub := range subs {
		if seedHasImpression(index) {
			counts[sub.Day]++
		}
	}
	return counts
}

func seedClickQuotasByDay(impressionsByDay map[int]int) map[int]int {
	quotas := make(map[int]int, len(impressionsByDay))
	for day, impressions := range impressionsByDay {
		quotas[day] = (impressions*seedClickRatePermille(day) + 500) / 1000
	}
	return quotas
}

func seedClickRatePermille(day int) int {
	if day%13 == 4 || day%17 == 9 {
		return 11
	}
	if day%7 == 2 {
		return 13
	}
	if day%6 == 0 {
		return 19
	}
	return 16
}

func seedClickCountsByDay(subs []subscriberSeed, clickedIndexes map[int]bool) map[int]int {
	counts := make(map[int]int)
	for index, sub := range subs {
		if clickedIndexes[index] {
			counts[sub.Day]++
		}
	}
	return counts
}

func seedConversionQuotasByDay(clicksByDay map[int]int) map[int]int {
	quotas := make(map[int]int, len(clicksByDay))
	for day, clicks := range clicksByDay {
		quota := (clicks*seedConversionRatePermille(day) + 500) / 1000
		capValue := clicks * 6 / 100
		if quota > capValue {
			quota = capValue
		}
		quotas[day] = quota
	}
	return quotas
}

func seedConversionRatePermille(day int) int {
	if day%13 == 4 || day%17 == 9 {
		return 14
	}
	if day%7 == 2 {
		return 26
	}
	if day%6 == 0 {
		return 58
	}
	if day%5 == 1 {
		return 36
	}
	return 44
}

func seedPayout(index int, day int) float64 {
	return (85.0 + float64(index%11)*8.5) * seedRevenueMultiplier(day)
}

func creatives() []creativeSeed {
	result := make([]creativeSeed, 0, len(campaigns)*3)
	for campaignIndex, campaign := range campaigns {
		for creativeIndex := range 3 {
			id := fmt.Sprintf("40000000-0000-0000-0000-%012d", campaignIndex*3+creativeIndex+1)
			provider := creativeIndex > 0
			status := "active"
			if campaign.Status == "draft" && creativeIndex == 2 {
				status = "paused"
			}
			result = append(result, creativeSeed{
				ID:       id,
				Campaign: campaign,
				Title:    fmt.Sprintf("%s Creative %d", campaign.Name, creativeIndex+1),
				Body:     "Seed body for admin testing",
				URL:      fmt.Sprintf("https://offers.seed.local/%s/%d", campaign.ID, creativeIndex+1),
				Status:   status,
				Provider: provider,
			})
		}
	}
	return result
}

func launches() []launchSeed {
	result := make([]launchSeed, 0, len(campaigns)*2)
	for campaignIndex, campaign := range campaigns {
		for launchIndex := range 2 {
			result = append(result, launchSeed{
				ID:       fmt.Sprintf("60000000-0000-0000-0000-%012d", campaignIndex*2+launchIndex+1),
				Campaign: campaign,
				Day:      campaignIndex + launchIndex + 1,
			})
		}
	}
	return result
}

func creativesForCampaign(campaignID string) []creativeSeed {
	result := []creativeSeed{}
	for _, creative := range creatives() {
		if creative.Campaign.ID == campaignID {
			result = append(result, creative)
		}
	}
	return result
}

func campaignForSource(sourceID string, index int) campaignSeed {
	matches := []campaignSeed{}
	for _, campaign := range campaigns {
		if campaign.AudienceScope == "all" || containsString(campaignSourceIDs(campaign), sourceID) {
			matches = append(matches, campaign)
		}
	}
	if len(matches) == 0 {
		return campaigns[index%len(campaigns)]
	}
	return matches[index%len(matches)]
}

func launchForCampaign(campaignID string, index int) launchSeed {
	matches := []launchSeed{}
	for _, launch := range launches() {
		if launch.Campaign.ID == campaignID {
			matches = append(matches, launch)
		}
	}
	return matches[index%len(matches)]
}

func targetingRules(campaign campaignSeed) string {
	if strings.Contains(campaign.Name, "iOS") {
		return `{"os_names":["iOS"],"device_types":["mobile"]}`
	}
	if strings.Contains(campaign.Name, "Android") {
		return `{"os_names":["Android"],"device_types":["mobile"]}`
	}
	if strings.Contains(campaign.Name, "VPN") {
		return `{"countries":["US","GB","CA"]}`
	}
	return `{}`
}

func campaignSourceIDs(campaign campaignSeed) []string {
	if campaign.AudienceScope == "all" {
		return sourceIDs()
	}
	if len(campaign.SourceIDs) > 0 {
		return campaign.SourceIDs
	}
	if campaign.SourceID != "" {
		return []string{campaign.SourceID}
	}
	return []string{sources[0].ID}
}

func publisherIDForSource(sourceID string) string {
	for _, source := range sources {
		if source.ID == sourceID {
			return source.PublisherID
		}
	}
	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func weightedDay(index int, rng *rand.Rand, weights []int) int {
	total := 0
	for _, weight := range weights {
		total += weight
	}
	pick := (index*17 + rng.Intn(total)) % total
	for day, weight := range weights {
		if pick < weight {
			return day
		}
		pick -= weight
	}
	return len(weights) - 1
}

func seedCostAmount(day int, campaignIndex int, sourceID string) float64 {
	sourceShape := map[string]float64{
		sources[0].ID: 1.25,
		sources[1].ID: 0.75,
		sources[2].ID: 1.55,
	}
	return (2.4 + float64(campaignIndex+1)*0.7) * seedDayShape(day) * sourceShape[sourceID]
}

func seedPublisherCostAmount(day int, publisherIndex int) float64 {
	return (0.8 + float64(publisherIndex+1)*0.45) * seedDayShape(day)
}

func seedDayCount() int {
	return time.Now().UTC().Day()
}

func seedDayWeights(days int) []int {
	weights := make([]int, 0, days)
	for day := range days {
		weight := 9 + ((day + 3) * (day + 11) % 31)
		if day%6 == 0 {
			weight += 28
		}
		if day%9 == 3 {
			weight += 18
		}
		if day%5 == 1 {
			weight /= 2
		}
		if weight < 3 {
			weight = 3
		}
		weights = append(weights, weight)
	}
	return weights
}

func seedDayShape(day int) float64 {
	weight := seedDayWeights(seedDayCount())[day]
	return 0.35 + float64(weight)/24.0
}

func seedRevenueMultiplier(day int) float64 {
	if day%13 == 4 || day%17 == 9 {
		return 0.18
	}
	if day%7 == 2 {
		return 0.65
	}
	if day%6 == 0 {
		return 1.35
	}
	return 1.0
}

func publisherIDs() []string {
	ids := make([]string, 0, len(publishers))
	for _, publisher := range publishers {
		ids = append(ids, publisher.ID)
	}
	return ids
}

func sourceIDs() []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.ID)
	}
	return ids
}

func campaignIDs() []string {
	ids := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		ids = append(ids, campaign.ID)
	}
	return ids
}

func providerConfigID(campaignID string) string {
	return strings.Replace(campaignID, "30000000", "70000000", 1)
}

func syncLogID(campaignID string) string {
	return strings.Replace(campaignID, "30000000", "71000000", 1)
}

func postbackConfigID(sourceID string) string {
	return strings.Replace(sourceID, "20000000", "80000000", 1)
}

func deliveryID(index int) string {
	return fmt.Sprintf("90000000-0000-0000-0000-%012d", index+1)
}

func triggerID(index int) string {
	return fmt.Sprintf("91000000-0000-0000-0000-%012d", index+1)
}

func quotedList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "'"+value+"'")
	}
	return strings.Join(quoted, ",")
}

func nowMinus(days int, offset time.Duration) time.Time {
	return time.Now().UTC().AddDate(0, 0, -days).Add(-offset)
}

func chTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

func userAgent(sub subscriberSeed) string {
	return fmt.Sprintf("SeedAdminTest/1.0 (%s; %s; %s)", sub.Device, sub.OS, sub.Browser)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func fatal(action string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
