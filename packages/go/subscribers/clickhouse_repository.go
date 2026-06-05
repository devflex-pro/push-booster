package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/clickhouse"
	"github.com/devflex-pro/push-booster/packages/go/inventory"
)

type ClickHouseRepository struct {
	client *clickhouse.Client
}

func NewClickHouseRepository(client *clickhouse.Client) *ClickHouseRepository {
	return &ClickHouseRepository{client: client}
}

type subscriberRow struct {
	SourceID          string `json:"source_id"`
	SubscriptionID    string `json:"subscription_id"`
	Endpoint          string `json:"endpoint"`
	P256DH            string `json:"p256dh"`
	Auth              string `json:"auth"`
	UserAgent         string `json:"user_agent"`
	SubID             string `json:"subid"`
	Channel           string `json:"channel"`
	LandingURL        string `json:"landing_url"`
	Referrer          string `json:"referrer"`
	IP                string `json:"ip"`
	Country           string `json:"country"`
	Region            string `json:"region"`
	City              string `json:"city"`
	Timezone          string `json:"timezone"`
	Language          string `json:"language"`
	BrowserName       string `json:"browser_name"`
	BrowserVersion    string `json:"browser_version"`
	OSName            string `json:"os_name"`
	OSVersion         string `json:"os_version"`
	DeviceType        string `json:"device_type"`
	DeviceVendor      string `json:"device_vendor"`
	DeviceModel       string `json:"device_model"`
	UAPlatform        string `json:"ua_platform"`
	UAPlatformVersion string `json:"ua_platform_version"`
	UAMobile          uint8  `json:"ua_mobile"`
	UAFullVersion     string `json:"ua_full_version"`
	UAArch            string `json:"ua_arch"`
	UABitness         string `json:"ua_bitness"`
	SubscribedAt      string `json:"subscribed_at"`
}

type subscriberEventRow struct {
	SourceID       string `json:"source_id"`
	DeliveryID     string `json:"delivery_id"`
	CampaignID     string `json:"campaign_id"`
	CreativeID     string `json:"creative_id"`
	EventID        string `json:"event_id"`
	TargetURL      string `json:"target_url"`
	SubscriptionID string `json:"subscription_id"`
	Endpoint       string `json:"endpoint"`
	EventType      string `json:"event_type"`
	UserAgent      string `json:"user_agent"`
	OccurredAt     string `json:"occurred_at"`
}

type creativeExposureRow struct {
	SourceID       string `json:"source_id"`
	SubscriptionID string `json:"subscription_id"`
	CampaignID     string `json:"campaign_id"`
	CreativeID     string `json:"creative_id"`
	OccurredAt     string `json:"occurred_at"`
}

type payloadDecisionRow struct {
	TriggerID      string `json:"trigger_id"`
	SubscriptionID string `json:"subscription_id"`
	SourceID       string `json:"source_id"`
	CampaignID     string `json:"campaign_id"`
	CreativeID     string `json:"creative_id"`
	Result         string `json:"result"`
	Reason         string `json:"reason"`
	Error          string `json:"error"`
	OccurredAt     string `json:"occurred_at"`
}

func (r *ClickHouseRepository) Save(ctx context.Context, input SubscribeInput) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	row := subscriberRow{
		SourceID:          input.SourceID,
		SubscriptionID:    input.SubscriptionID,
		Endpoint:          input.Endpoint,
		P256DH:            input.Keys.P256DH,
		Auth:              input.Keys.Auth,
		UserAgent:         "",
		SubID:             input.SubID,
		Channel:           input.Channel,
		LandingURL:        input.LandingURL,
		Referrer:          input.Referrer,
		IP:                input.Targeting.IP,
		Country:           input.Targeting.Country,
		Region:            input.Targeting.Region,
		City:              input.Targeting.City,
		Timezone:          input.Targeting.Timezone,
		Language:          input.Targeting.Language,
		BrowserName:       input.Targeting.BrowserName,
		BrowserVersion:    input.Targeting.BrowserVersion,
		OSName:            input.Targeting.OSName,
		OSVersion:         input.Targeting.OSVersion,
		DeviceType:        input.Targeting.DeviceType,
		DeviceVendor:      input.Targeting.DeviceVendor,
		DeviceModel:       input.Targeting.DeviceModel,
		UAPlatform:        input.Targeting.UAPlatform,
		UAPlatformVersion: input.Targeting.UAPlatformVersion,
		UAMobile:          boolUint8(input.Targeting.UAMobile),
		UAFullVersion:     input.Targeting.UAFullVersion,
		UAArch:            input.Targeting.UAArch,
		UABitness:         input.Targeting.UABitness,
		SubscribedAt:      now,
	}
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal subscriber row: %w", err)
	}
	query := "INSERT INTO subscribers FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert subscriber: %w", err)
	}
	event := subscriberEventRow{
		SourceID:       input.SourceID,
		DeliveryID:     "",
		CampaignID:     "",
		CreativeID:     "",
		EventID:        "",
		TargetURL:      "",
		SubscriptionID: input.SubscriptionID,
		Endpoint:       input.Endpoint,
		EventType:      "subscribed",
		UserAgent:      input.UserAgent,
		OccurredAt:     now,
	}
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal subscriber event row: %w", err)
	}
	eventQuery := "INSERT INTO subscriber_events FORMAT JSONEachRow\n" + string(eventPayload)
	if err := r.client.Exec(ctx, eventQuery); err != nil {
		return fmt.Errorf("insert subscriber event: %w", err)
	}
	return nil
}

func boolUint8(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}

func (r *ClickHouseRepository) SaveEvent(ctx context.Context, input ServiceWorkerEventInput) error {
	subscription, err := r.subscriptionDetails(ctx, input.SubscriptionID)
	if err != nil {
		return err
	}
	event := subscriberEventRow{
		SourceID:       subscription.SourceID,
		DeliveryID:     input.DeliveryID,
		CampaignID:     input.CampaignID,
		CreativeID:     input.CreativeID,
		EventID:        input.EventID,
		TargetURL:      input.URL,
		SubscriptionID: input.SubscriptionID,
		Endpoint:       subscription.Endpoint,
		EventType:      input.EventType,
		UserAgent:      input.UserAgent,
		OccurredAt:     time.Now().UTC().Format("2006-01-02 15:04:05.000"),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal subscriber event row: %w", err)
	}
	query := "INSERT INTO subscriber_events FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert subscriber event: %w", err)
	}
	return nil
}

func (r *ClickHouseRepository) SourceIDForSubscription(ctx context.Context, subscriptionID string) (string, error) {
	subscription, err := r.subscriptionDetails(ctx, subscriptionID)
	if err != nil {
		return "", err
	}
	return subscription.SourceID, nil
}

func (r *ClickHouseRepository) TargetingForSubscription(
	ctx context.Context,
	subscriptionID string,
) (SubscriberTargetingSnapshot, error) {
	escaped := strings.ReplaceAll(subscriptionID, "'", "''")
	query := fmt.Sprintf(
		`SELECT country,
       language,
       device_type,
       os_name,
       browser_name
FROM subscribers
WHERE subscription_id = '%s'
ORDER BY subscribed_at DESC
LIMIT 1
FORMAT JSONEachRow`,
		escaped,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return SubscriberTargetingSnapshot{}, fmt.Errorf("resolve subscription targeting: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SubscriberTargetingSnapshot{}, inventory.ErrNotFound
	}
	var snapshot struct {
		Country     string `json:"country"`
		Language    string `json:"language"`
		DeviceType  string `json:"device_type"`
		OSName      string `json:"os_name"`
		BrowserName string `json:"browser_name"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return SubscriberTargetingSnapshot{}, fmt.Errorf("parse subscription targeting: %w", err)
	}
	return SubscriberTargetingSnapshot{
		Country:     snapshot.Country,
		Language:    snapshot.Language,
		DeviceType:  snapshot.DeviceType,
		OSName:      snapshot.OSName,
		BrowserName: snapshot.BrowserName,
	}, nil
}

type subscriptionDetails struct {
	SourceID string `json:"source_id"`
	Endpoint string `json:"endpoint"`
}

func (r *ClickHouseRepository) subscriptionDetails(ctx context.Context, subscriptionID string) (subscriptionDetails, error) {
	escaped := strings.ReplaceAll(subscriptionID, "'", "''")
	query := fmt.Sprintf(
		"SELECT source_id, endpoint FROM subscribers WHERE subscription_id = '%s' ORDER BY subscribed_at DESC LIMIT 1 FORMAT JSONEachRow",
		escaped,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return subscriptionDetails{}, fmt.Errorf("resolve subscription source: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return subscriptionDetails{}, inventory.ErrNotFound
	}
	var details subscriptionDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return subscriptionDetails{}, fmt.Errorf("parse subscription source: %w", err)
	}
	return details, nil
}

func (r *ClickHouseRepository) PayloadForSubscription(
	ctx context.Context,
	subscriptionID string,
	_ string,
) (PushPayload, error) {
	escaped := strings.ReplaceAll(subscriptionID, "'", "''")
	query := fmt.Sprintf(
		"SELECT source_id FROM subscribers WHERE subscription_id = '%s' ORDER BY subscribed_at DESC LIMIT 1",
		escaped,
	)
	sourceID, err := r.client.QueryText(ctx, query)
	if err != nil {
		return PushPayload{}, fmt.Errorf("resolve push payload subscription: %w", err)
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return PushPayload{}, inventory.ErrNotFound
	}
	return PushPayload{
		Title:    "Push Booster",
		Body:     "Notification payload resolved at display time.",
		URL:      "/",
		SourceID: sourceID,
	}, nil
}

func (r *ClickHouseRepository) SeenCreativeIDsSince(
	ctx context.Context,
	subscriptionID string,
	campaignID string,
	since time.Time,
) (map[string]bool, error) {
	escapedSubscriptionID := strings.ReplaceAll(subscriptionID, "'", "''")
	escapedCampaignID := strings.ReplaceAll(campaignID, "'", "''")
	query := fmt.Sprintf(
		`SELECT creative_id
FROM creative_exposures
WHERE subscription_id = '%s'
  AND campaign_id = '%s'
  AND occurred_at >= toDateTime64('%s', 3, 'UTC')
GROUP BY creative_id
FORMAT JSONEachRow`,
		escapedSubscriptionID,
		escapedCampaignID,
		since.UTC().Format("2006-01-02 15:04:05.000"),
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query creative exposures: %w", err)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			CreativeID string `json:"creative_id"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse creative exposure row: %w", err)
		}
		seen[row.CreativeID] = true
	}
	return seen, nil
}

func (r *ClickHouseRepository) RecordCreativeExposure(ctx context.Context, input CreativeExposureInput) error {
	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	row := creativeExposureRow{
		SourceID:       input.SourceID,
		SubscriptionID: input.SubscriptionID,
		CampaignID:     input.CampaignID,
		CreativeID:     input.CreativeID,
		OccurredAt:     occurredAt.UTC().Format("2006-01-02 15:04:05.000"),
	}
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal creative exposure row: %w", err)
	}
	query := "INSERT INTO creative_exposures FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert creative exposure: %w", err)
	}
	return nil
}

func (r *ClickHouseRepository) RecordPayloadDecision(ctx context.Context, input PayloadDecisionInput) error {
	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	row := payloadDecisionRow{
		TriggerID:      input.TriggerID,
		SubscriptionID: input.SubscriptionID,
		SourceID:       input.SourceID,
		CampaignID:     input.CampaignID,
		CreativeID:     input.CreativeID,
		Result:         input.Result,
		Reason:         input.Reason,
		Error:          input.Error,
		OccurredAt:     occurredAt.UTC().Format("2006-01-02 15:04:05.000"),
	}
	payload, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal payload decision row: %w", err)
	}
	query := "INSERT INTO payload_decisions FORMAT JSONEachRow\n" + string(payload)
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("insert payload decision: %w", err)
	}
	return nil
}

func (r *ClickHouseRepository) CampaignReport(ctx context.Context, campaignID string) (CampaignReport, error) {
	decisionsByResult, err := r.countPayloadDecisions(ctx, campaignID, "result")
	if err != nil {
		return CampaignReport{}, err
	}
	decisionsByReason, err := r.countPayloadDecisions(ctx, campaignID, "reason")
	if err != nil {
		return CampaignReport{}, err
	}
	exposures, err := r.countCreativeExposures(ctx, campaignID)
	if err != nil {
		return CampaignReport{}, err
	}
	eventsByType, err := r.countTrackingEvents(ctx, campaignID)
	if err != nil {
		return CampaignReport{}, err
	}
	report := CampaignReport{
		CampaignID:        campaignID,
		DecisionsByResult: decisionsByResult,
		DecisionsByReason: decisionsByReason,
		CreativeExposures: exposures,
		EventsByType:      eventsByType,
	}
	for _, count := range decisionsByResult {
		report.DecisionsTotal += count
	}
	for _, count := range eventsByType {
		report.TrackedEvents += count
	}
	report.Selected = decisionsByResult[payloadDecisionSelected]
	report.Suppressed = decisionsByResult[payloadDecisionSuppressed]
	report.NotFound = decisionsByResult[payloadDecisionNotFound]
	report.Errors = decisionsByResult[payloadDecisionError]
	report.Shown = eventsByType["notification_shown"]
	report.Clicks = eventsByType["notification_click"]
	report.Closed = eventsByType["notification_close"]
	report.Health = campaignReportHealth(report)
	return report, nil
}

func (r *ClickHouseRepository) EstimateAudience(
	ctx context.Context,
	sourceIDs []string,
	rules inventory.TargetingRules,
) (int64, error) {
	query := campaignAudienceSelect(
		"",
		"",
		sourceIDs,
		rules,
		"",
		true,
	)
	return r.count(ctx, query, "estimate campaign audience")
}

func (r *ClickHouseRepository) BuildAudience(
	ctx context.Context,
	input inventory.BuildAudienceInput,
) (int64, error) {
	insertQuery := fmt.Sprintf(
		`INSERT INTO campaign_audience
%s`,
		campaignAudienceSelect(
			input.LaunchID,
			input.CampaignID,
			input.SourceIDs,
			input.TargetingRules,
			input.Timezone,
			false,
		),
	)
	if err := r.client.Exec(ctx, insertQuery); err != nil {
		return 0, fmt.Errorf("build campaign audience: %w", err)
	}
	return r.countAudienceLaunch(ctx, input.LaunchID)
}

func (r *ClickHouseRepository) AudienceBatch(
	ctx context.Context,
	input inventory.AudienceBatchInput,
) ([]inventory.AudienceRow, error) {
	escapedLaunchID := strings.ReplaceAll(input.LaunchID, "'", "''")
	cursorSQL := ""
	if input.AfterSubscriptionID != "" {
		cursorSQL = fmt.Sprintf(
			`  AND (shard > %d OR (shard = %d AND subscription_id > '%s'))
`,
			input.AfterShard,
			input.AfterShard,
			strings.ReplaceAll(input.AfterSubscriptionID, "'", "''"),
		)
	}
	query := fmt.Sprintf(
		`SELECT launch_id,
       campaign_id,
       source_id,
       subscription_id,
       endpoint,
       p256dh,
       auth,
       shard
FROM campaign_audience
WHERE launch_id = '%s'
%sORDER BY shard ASC, subscription_id ASC
LIMIT %d
FORMAT JSONEachRow`,
		escapedLaunchID,
		cursorSQL,
		input.Limit,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query campaign audience batch: %w", err)
	}
	rows := []inventory.AudienceRow{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row inventory.AudienceRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse campaign audience row: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (r *ClickHouseRepository) Timezones(ctx context.Context) ([]string, error) {
	raw, err := r.client.QueryText(
		ctx,
		`SELECT timezone
FROM (
    SELECT argMax(timezone, subscribed_at) AS timezone
    FROM subscribers
    GROUP BY subscription_id
)
WHERE timezone != ''
GROUP BY timezone
ORDER BY timezone
FORMAT JSONEachRow`,
	)
	if err != nil {
		return nil, fmt.Errorf("query subscriber timezones: %w", err)
	}
	timezones := []string{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Timezone string `json:"timezone"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse subscriber timezone: %w", err)
		}
		if strings.TrimSpace(row.Timezone) != "" {
			timezones = append(timezones, row.Timezone)
		}
	}
	return timezones, nil
}

func (r *ClickHouseRepository) countPayloadDecisions(
	ctx context.Context,
	campaignID string,
	field string,
) (map[string]int64, error) {
	escapedCampaignID := strings.ReplaceAll(campaignID, "'", "''")
	query := fmt.Sprintf(
		`SELECT %s AS key, count() AS count
FROM payload_decisions
WHERE campaign_id = '%s'
GROUP BY key
FORMAT JSONEachRow`,
		field,
		escapedCampaignID,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count payload decisions by %s: %w", field, err)
	}
	return parseCountRows(raw, "payload decision count")
}

func (r *ClickHouseRepository) countCreativeExposures(
	ctx context.Context,
	campaignID string,
) ([]CreativeExposureCount, error) {
	escapedCampaignID := strings.ReplaceAll(campaignID, "'", "''")
	query := fmt.Sprintf(
		`SELECT creative_id, count() AS count
FROM creative_exposures
WHERE campaign_id = '%s'
GROUP BY creative_id
ORDER BY count DESC
FORMAT JSONEachRow`,
		escapedCampaignID,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count creative exposures: %w", err)
	}
	rows := []CreativeExposureCount{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			CreativeID string `json:"creative_id"`
			Count      string `json:"count"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse creative exposure count: %w", err)
		}
		count, err := strconv.ParseInt(row.Count, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse creative exposure count value: %w", err)
		}
		rows = append(rows, CreativeExposureCount{
			CreativeID: row.CreativeID,
			Count:      count,
		})
	}
	return rows, nil
}

func (r *ClickHouseRepository) countTrackingEvents(
	ctx context.Context,
	campaignID string,
) (map[string]int64, error) {
	escapedCampaignID := strings.ReplaceAll(campaignID, "'", "''")
	query := fmt.Sprintf(
		`SELECT event_type AS key, count() AS count
FROM subscriber_events
WHERE campaign_id = '%s'
GROUP BY key
FORMAT JSONEachRow`,
		escapedCampaignID,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count tracking events: %w", err)
	}
	return parseCountRows(raw, "tracking event count")
}

func (r *ClickHouseRepository) countAudienceLaunch(ctx context.Context, launchID string) (int64, error) {
	escapedLaunchID := strings.ReplaceAll(launchID, "'", "''")
	query := fmt.Sprintf(
		"SELECT count() FROM campaign_audience WHERE launch_id = '%s'",
		escapedLaunchID,
	)
	return r.count(ctx, query, "count campaign audience")
}

func campaignAudienceSelect(
	launchID string,
	campaignID string,
	sourceIDs []string,
	rules inventory.TargetingRules,
	timezone string,
	countOnly bool,
) string {
	sourceFilter := campaignAudienceSourceFilter(sourceIDs)
	timezoneFilter := campaignAudienceTimezoneFilter(timezone)
	selectClause := "SELECT count()"
	if !countOnly {
		selectClause = fmt.Sprintf(
			`SELECT toUUID('%s') AS launch_id,
       toUUID('%s') AS campaign_id,
       source_id,
       subscription_id,
       endpoint,
       p256dh,
       auth,
       toUInt16(cityHash64(subscription_id) %% 256) AS shard,
       now64(3, 'UTC') AS selected_at`,
			strings.ReplaceAll(launchID, "'", "''"),
			strings.ReplaceAll(campaignID, "'", "''"),
		)
	}
	return fmt.Sprintf(
		`%s
FROM (
    SELECT source_id,
           subscription_id,
           argMax(endpoint, subscribed_at) AS endpoint,
           argMax(p256dh, subscribed_at) AS p256dh,
           argMax(auth, subscribed_at) AS auth,
           argMax(timezone, subscribed_at) AS timezone,
           lower(argMax(country, subscribed_at)) AS country,
           lower(argMax(language, subscribed_at)) AS language,
           lower(argMax(device_type, subscribed_at)) AS device_type,
           lower(argMax(os_name, subscribed_at)) AS os_name,
           lower(argMax(browser_name, subscribed_at)) AS browser_name
    FROM subscribers
%s
    GROUP BY source_id, subscription_id
)
WHERE endpoint != ''
  AND p256dh != ''
  AND auth != ''
%s
%s`,
		selectClause,
		sourceFilter,
		timezoneFilter,
		audienceTargetingSQL(rules),
	)
}

func campaignAudienceSourceFilter(sourceIDs []string) string {
	if len(sourceIDs) == 0 {
		return ""
	}
	return fmt.Sprintf("    WHERE source_id IN (%s)", sqlStringList(sourceIDs))
}

func campaignAudienceTimezoneFilter(timezone string) string {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return ""
	}
	return fmt.Sprintf("  AND timezone = '%s'\n", strings.ReplaceAll(timezone, "'", "''"))
}

func audienceTargetingSQL(rules inventory.TargetingRules) string {
	parts := []string{}
	parts = appendTargetingIn(parts, "country", rules.Countries)
	parts = appendTargetingLanguage(parts, rules.Languages)
	parts = appendTargetingIn(parts, "device_type", rules.DeviceTypes)
	parts = appendTargetingIn(parts, "os_name", rules.OSNames)
	parts = appendTargetingIn(parts, "browser_name", rules.BrowserNames)
	if len(parts) == 0 {
		return ""
	}
	return "\n  AND " + strings.Join(parts, "\n  AND ")
}

func appendTargetingIn(parts []string, field string, values []string) []string {
	normalized := sqlStringList(values)
	if normalized == "" {
		return parts
	}
	return append(parts, fmt.Sprintf("%s IN (%s)", field, normalized))
}

func appendTargetingLanguage(parts []string, values []string) []string {
	normalizedValues := normalizedTargetingValues(values)
	if len(normalizedValues) == 0 {
		return parts
	}
	conditions := []string{}
	for _, value := range normalizedValues {
		escaped := strings.ReplaceAll(value, "'", "''")
		conditions = append(
			conditions,
			fmt.Sprintf("(language = '%s' OR startsWith(language, '%s-'))", escaped, escaped),
		)
	}
	return append(parts, "("+strings.Join(conditions, " OR ")+")")
}

func sqlStringList(values []string) string {
	normalizedValues := normalizedTargetingValues(values)
	if len(normalizedValues) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(normalizedValues))
	for _, value := range normalizedValues {
		quoted = append(quoted, "'"+strings.ReplaceAll(value, "'", "''")+"'")
	}
	return strings.Join(quoted, ", ")
}

func normalizedTargetingValues(values []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized
}

func parseCountRows(raw string, operation string) (map[string]int64, error) {
	counts := map[string]int64{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Key   string `json:"key"`
			Count string `json:"count"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("parse %s row: %w", operation, err)
		}
		count, err := strconv.ParseInt(row.Count, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse %s value: %w", operation, err)
		}
		counts[row.Key] = count
	}
	return counts, nil
}

func campaignReportHealth(report CampaignReport) ReportHealth {
	issues := []string{}
	if report.DecisionsTotal == 0 {
		issues = append(issues, "no payload decisions recorded")
	}
	if report.Errors > 0 {
		issues = append(issues, "payload decision errors detected")
	}
	if report.Selected == 0 && report.DecisionsTotal > 0 {
		issues = append(issues, "no selected payloads")
	}
	status := "ok"
	if len(issues) > 0 {
		status = "attention"
	}
	return ReportHealth{Status: status, Issues: issues}
}

func (r *ClickHouseRepository) CountBySource(ctx context.Context, sourceID string) (int64, error) {
	escaped := strings.ReplaceAll(sourceID, "'", "''")
	query := fmt.Sprintf("SELECT countDistinct(subscription_id) FROM subscribers WHERE source_id = '%s'", escaped)
	return r.count(ctx, query, "count subscribers")
}

func (r *ClickHouseRepository) CountBySourceToday(ctx context.Context, sourceID string) (int64, error) {
	escaped := strings.ReplaceAll(sourceID, "'", "''")
	query := fmt.Sprintf(
		"SELECT countDistinct(subscription_id) FROM subscribers WHERE source_id = '%s' AND subscribed_at >= toStartOfDay(now('UTC'))",
		escaped,
	)
	return r.count(ctx, query, "count subscribers today")
}

func (r *ClickHouseRepository) CountEventsBySourceToday(ctx context.Context, sourceID string) (int64, error) {
	escaped := strings.ReplaceAll(sourceID, "'", "''")
	query := fmt.Sprintf(
		"SELECT count() FROM subscriber_events WHERE source_id = '%s' AND occurred_at >= toStartOfDay(now('UTC'))",
		escaped,
	)
	return r.count(ctx, query, "count subscriber events today")
}

func (r *ClickHouseRepository) CountEventsBySourceTodayByType(
	ctx context.Context,
	sourceID string,
) (map[string]int64, error) {
	escaped := strings.ReplaceAll(sourceID, "'", "''")
	query := fmt.Sprintf(
		"SELECT event_type, count() AS count FROM subscriber_events WHERE source_id = '%s' AND occurred_at >= toStartOfDay(now('UTC')) GROUP BY event_type FORMAT JSONEachRow",
		escaped,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count subscriber events today by type: %w", err)
	}
	rows := strings.Split(strings.TrimSpace(raw), "\n")
	counts := map[string]int64{}
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		var event struct {
			EventType string `json:"event_type"`
			Count     string `json:"count"`
		}
		if err := json.Unmarshal([]byte(row), &event); err != nil {
			return nil, fmt.Errorf("parse subscriber event count: %w", err)
		}
		count, err := strconv.ParseInt(event.Count, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse subscriber event count value: %w", err)
		}
		counts[event.EventType] = count
	}
	return counts, nil
}

func (r *ClickHouseRepository) LastEventAtBySource(ctx context.Context, sourceID string) (string, error) {
	escaped := strings.ReplaceAll(sourceID, "'", "''")
	query := fmt.Sprintf(
		"SELECT ifNull(toString(max(occurred_at)), '') FROM subscriber_events WHERE source_id = '%s'",
		escaped,
	)
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return "", fmt.Errorf("last subscriber event: %w", err)
	}
	return strings.TrimSpace(raw), nil
}

func (r *ClickHouseRepository) count(
	ctx context.Context,
	query string,
	operation string,
) (int64, error) {
	raw, err := r.client.QueryText(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", operation, err)
	}
	count, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", operation, err)
	}
	return count, nil
}
