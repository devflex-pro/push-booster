package subscribers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
)

var (
	ErrInvalidInput         = errors.New("invalid subscriber input")
	ErrForbiddenOrigin      = errors.New("source origin is not allowed")
	ErrVAPIDNotReady        = errors.New("vapid public key is not configured")
	ErrTriggerStoreNotReady = errors.New("delivery trigger store is not configured")
	ErrPayloadSuppressed    = errors.New("push payload suppressed")
)

const (
	payloadDecisionSelected   = "selected"
	payloadDecisionSuppressed = "suppressed"
	payloadDecisionNotFound   = "not_found"
	payloadDecisionError      = "error"

	payloadDecisionReasonSelected           = "selected"
	payloadDecisionReasonNoActiveCreative   = "no_active_creative"
	payloadDecisionReasonCapsExceeded       = "all_eligible_creatives_capped"
	payloadDecisionReasonSourceLookupFailed = "source_lookup_failed"
	payloadDecisionReasonCreativeLookup     = "creative_lookup_failed"
	payloadDecisionReasonExposureLookup     = "exposure_lookup_failed"
	payloadDecisionReasonCapCheck           = "cap_check_failed"
	payloadDecisionReasonExposureRecord     = "exposure_record_failed"
	payloadDecisionReasonTargetingLookup    = "targeting_lookup_failed"
	payloadDecisionReasonCampaignTargeting  = "campaign_targeting_mismatch"
)

type Keys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type SubscribeInput struct {
	SubscriptionID  string `json:"-"`
	SourceID        string `json:"source_id"`
	Endpoint        string `json:"endpoint"`
	Keys            Keys   `json:"keys"`
	SubID           string `json:"subid"`
	Channel         string `json:"channel"`
	LandingURL      string `json:"landing_url"`
	Referrer        string `json:"referrer"`
	Timezone        string `json:"timezone"`
	Targeting       Targeting
	UserAgent       string `json:"-"`
	RequestOrigin   string `json:"-"`
	RequestReferrer string `json:"-"`
}

type SubscribeResult struct {
	SubscriptionID string `json:"subscription_id"`
	Status         string `json:"status"`
}

type Targeting struct {
	IP                string
	Country           string
	Region            string
	City              string
	Timezone          string
	Language          string
	BrowserName       string
	BrowserVersion    string
	OSName            string
	OSVersion         string
	DeviceType        string
	DeviceVendor      string
	DeviceModel       string
	UAPlatform        string
	UAPlatformVersion string
	UAMobile          bool
	UAFullVersion     string
	UAArch            string
	UABitness         string
}

type ServiceWorkerEventInput struct {
	SubscriptionID  string `json:"subscription_id"`
	SourceID        string `json:"source_id,omitempty"`
	DeliveryID      string `json:"delivery_id,omitempty"`
	CampaignID      string `json:"campaign_id,omitempty"`
	CreativeID      string `json:"creative_id,omitempty"`
	EventID         string `json:"event_id,omitempty"`
	URL             string `json:"url,omitempty"`
	Endpoint        string `json:"-"`
	EventType       string `json:"event_type"`
	UserAgent       string `json:"-"`
	RequestOrigin   string `json:"-"`
	RequestReferrer string `json:"-"`
}

type PushPayloadRequest struct {
	TriggerID string
	UserAgent string
}

type CreateTriggerInput struct {
	TriggerID      string
	DeliveryID     string
	SubscriptionID string
	SourceID       string
	CampaignID     string
	TTL            time.Duration
}

type DeliveryTrigger struct {
	TriggerID      string    `json:"trigger_id"`
	DeliveryID     string    `json:"delivery_id"`
	SubscriptionID string    `json:"subscription_id"`
	SourceID       string    `json:"source_id"`
	CampaignID     string    `json:"campaign_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type CapCheckInput struct {
	SubscriptionID          string
	CampaignID              string
	CreativeID              string
	DailyCapPerSubscription int
	TotalCapPerSubscription int
	CreativeDailyCap        int
	CreativeTotalCap        int
}

type PushPayload struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	Icon           string `json:"icon,omitempty"`
	URL            string `json:"url"`
	ClickURL       string `json:"click_url,omitempty"`
	DeliveryID     string `json:"delivery_id,omitempty"`
	TriggerID      string `json:"trigger_id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	SourceID       string `json:"source_id,omitempty"`
	CampaignID     string `json:"campaign_id,omitempty"`
	CreativeID     string `json:"creative_id,omitempty"`
}

type CampaignReport struct {
	CampaignID        string                  `json:"campaign_id"`
	DecisionsTotal    int64                   `json:"decisions_total"`
	Selected          int64                   `json:"selected"`
	Suppressed        int64                   `json:"suppressed"`
	NotFound          int64                   `json:"not_found"`
	Errors            int64                   `json:"errors"`
	TrackedEvents     int64                   `json:"tracked_events"`
	Shown             int64                   `json:"shown"`
	Clicks            int64                   `json:"clicks"`
	Closed            int64                   `json:"closed"`
	EventsByType      map[string]int64        `json:"events_by_type"`
	DecisionsByResult map[string]int64        `json:"decisions_by_result"`
	DecisionsByReason map[string]int64        `json:"decisions_by_reason"`
	CreativeExposures []CreativeExposureCount `json:"creative_exposures"`
	Health            ReportHealth            `json:"health"`
}

type CreativeExposureCount struct {
	CreativeID string `json:"creative_id"`
	Count      int64  `json:"count"`
}

type ReportHealth struct {
	Status string   `json:"status"`
	Issues []string `json:"issues"`
}

type SubscriberTargetingSnapshot struct {
	Country     string
	Language    string
	DeviceType  string
	OSName      string
	BrowserName string
}

type SDKConfig struct {
	SourceID            string `json:"source_id"`
	VAPIDPublicKey      string `json:"vapid_public_key"`
	SubscribeEndpoint   string `json:"subscribe_endpoint"`
	PushPayloadEndpoint string `json:"push_payload_endpoint"`
	EventsEndpoint      string `json:"events_endpoint"`
	ServiceWorkerURL    string `json:"service_worker_url"`
}

type SDKConfigInput struct {
	SourceID        string
	RequestOrigin   string
	RequestReferrer string
}

type SDKScriptInput struct {
	SourceID        string
	RequestOrigin   string
	RequestReferrer string
}

type Store interface {
	Save(ctx context.Context, input SubscribeInput) error
	SaveEvent(ctx context.Context, input ServiceWorkerEventInput) error
	SourceIDForSubscription(ctx context.Context, subscriptionID string) (string, error)
	TargetingForSubscription(ctx context.Context, subscriptionID string) (SubscriberTargetingSnapshot, error)
	PayloadForSubscription(ctx context.Context, subscriptionID string, userAgent string) (PushPayload, error)
	CountBySource(ctx context.Context, sourceID string) (int64, error)
	CountBySourceToday(ctx context.Context, sourceID string) (int64, error)
	CountEventsBySourceToday(ctx context.Context, sourceID string) (int64, error)
	CountEventsBySourceTodayByType(ctx context.Context, sourceID string) (map[string]int64, error)
	LastEventAtBySource(ctx context.Context, sourceID string) (string, error)
	CampaignReport(ctx context.Context, campaignID string) (CampaignReport, error)
	EstimateAudience(
		ctx context.Context,
		sourceIDs []string,
		rules inventory.TargetingRules,
	) (int64, error)
	BuildAudience(ctx context.Context, input inventory.BuildAudienceInput) (int64, error)
	AudienceBatch(ctx context.Context, input inventory.AudienceBatchInput) ([]inventory.AudienceRow, error)
	Timezones(ctx context.Context) ([]string, error)
}

type TriggerStore interface {
	CreateTrigger(ctx context.Context, input CreateTriggerInput) (DeliveryTrigger, error)
	ResolveTrigger(ctx context.Context, triggerID string) (DeliveryTrigger, error)
}

type CreativeSelector interface {
	ActiveCreativesForCampaign(ctx context.Context, campaignID string) ([]inventory.Creative, error)
	ActiveCreativesForSource(ctx context.Context, sourceID string) ([]inventory.Creative, error)
}

type CapStore interface {
	Allow(ctx context.Context, input CapCheckInput) (bool, error)
}

type CreativeExposureInput struct {
	SubscriptionID string
	SourceID       string
	CampaignID     string
	CreativeID     string
	OccurredAt     time.Time
}

type CreativeExposureStore interface {
	SeenCreativeIDsSince(
		ctx context.Context,
		subscriptionID string,
		campaignID string,
		since time.Time,
	) (map[string]bool, error)
	RecordCreativeExposure(ctx context.Context, input CreativeExposureInput) error
}

type PayloadDecisionInput struct {
	TriggerID      string
	SubscriptionID string
	SourceID       string
	CampaignID     string
	CreativeID     string
	Result         string
	Reason         string
	Error          string
	OccurredAt     time.Time
}

type PayloadDecisionStore interface {
	RecordPayloadDecision(ctx context.Context, input PayloadDecisionInput) error
}

type EventIDStore interface {
	AllowEvent(ctx context.Context, input ServiceWorkerEventInput) (bool, error)
}

type SourceStore interface {
	GetSource(ctx context.Context, id string) (inventory.Source, error)
	ActiveVAPIDKeyForSource(ctx context.Context, sourceID string) (inventory.VAPIDKey, error)
}

type Config struct {
	FallbackVAPIDPublicKey string
	PublicURL              string
	PayloadURL             string
	TriggerStore           TriggerStore
	TriggerTTL             time.Duration
	Creatives              CreativeSelector
	Caps                   CapStore
	Exposures              CreativeExposureStore
	ExposureWindow         time.Duration
	Decisions              PayloadDecisionStore
	EventIDs               EventIDStore
}

type Service struct {
	store          Store
	sources        SourceStore
	vapidPublicKey string
	publicURL      string
	payloadURL     string
	triggers       TriggerStore
	triggerTTL     time.Duration
	creatives      CreativeSelector
	caps           CapStore
	exposures      CreativeExposureStore
	exposureWindow time.Duration
	decisions      PayloadDecisionStore
	eventIDs       EventIDStore
}

func NewService(
	store Store,
	sources SourceStore,
	vapidPublicKey string,
) *Service {
	return NewServiceWithConfig(store, sources, Config{FallbackVAPIDPublicKey: vapidPublicKey})
}

func NewServiceWithConfig(
	store Store,
	sources SourceStore,
	cfg Config,
) *Service {
	exposureWindow := cfg.ExposureWindow
	if exposureWindow == 0 {
		exposureWindow = 24 * time.Hour
	}
	return &Service{
		store:          store,
		sources:        sources,
		vapidPublicKey: strings.TrimSpace(cfg.FallbackVAPIDPublicKey),
		publicURL:      strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/"),
		payloadURL:     strings.TrimRight(strings.TrimSpace(cfg.PayloadURL), "/"),
		triggers:       cfg.TriggerStore,
		triggerTTL:     cfg.TriggerTTL,
		creatives:      cfg.Creatives,
		caps:           cfg.Caps,
		exposures:      cfg.Exposures,
		exposureWindow: exposureWindow,
		decisions:      cfg.Decisions,
		eventIDs:       cfg.EventIDs,
	}
}

func (s *Service) VAPIDPublicKey() (string, error) {
	if s.vapidPublicKey == "" {
		return "", ErrVAPIDNotReady
	}
	return s.vapidPublicKey, nil
}

func (s *Service) SDKConfig(ctx context.Context, input SDKConfigInput) (SDKConfig, error) {
	input.SourceID = strings.TrimSpace(input.SourceID)
	if input.SourceID == "" {
		return SDKConfig{}, errors.Join(ErrInvalidInput, errors.New("source_id is required"))
	}
	source, err := s.sources.GetSource(ctx, input.SourceID)
	if err != nil {
		return SDKConfig{}, err
	}
	if err := validateSourceOrigin(source, input.RequestOrigin, input.RequestReferrer); err != nil {
		return SDKConfig{}, err
	}
	key, err := s.sources.ActiveVAPIDKeyForSource(ctx, source.ID)
	if err != nil {
		return SDKConfig{}, err
	}
	return SDKConfig{
		SourceID:            source.ID,
		VAPIDPublicKey:      key.PublicKey,
		SubscribeEndpoint:   s.endpoint(s.publicURL, "/api/subscribe"),
		PushPayloadEndpoint: s.endpoint(s.payloadURL, "/api/push/payload"),
		EventsEndpoint:      s.endpoint(s.publicURL, "/api/sw/events"),
		ServiceWorkerURL:    "/push-booster-sw.js",
	}, nil
}

func (s *Service) SDKScript(ctx context.Context, input SDKScriptInput) (string, error) {
	cfg, err := s.SDKConfig(ctx, SDKConfigInput(input))
	if err != nil {
		return "", err
	}
	configURL := s.endpoint(
		s.publicURL,
		"/api/sdk/config?source_id="+url.QueryEscape(cfg.SourceID),
	)
	return fmt.Sprintf(`(function () {
  var script = document.currentScript;
  (async function () {
    if (!("serviceWorker" in navigator) || !("PushManager" in window)) return;
    var configResponse = await fetch(%q);
    if (!configResponse.ok) return;
    var config = await configResponse.json();
    var registration = await navigator.serviceWorker.register(config.service_worker_url);
    var readyRegistration = await navigator.serviceWorker.ready;
    if (readyRegistration.active) {
      readyRegistration.active.postMessage({type: "push_booster_config", config: config});
    }
    var params = new URLSearchParams(window.location.search);
    var permission = await Notification.requestPermission();
    if (permission !== "granted") return;
    var key = config.vapid_public_key.replace(/-/g, "+").replace(/_/g, "/");
    var raw = atob(key + "=".repeat((4 - key.length %% 4) %% 4));
    var bytes = new Uint8Array(raw.length);
    for (var i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
    var subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: bytes
    });
    var payload = subscription.toJSON();
    var subscribeResponse = await fetch(config.subscribe_endpoint, {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({
        source_id: config.source_id,
        endpoint: payload.endpoint,
        subid: params.get("subid") || (script && script.dataset.subid) || "",
        channel: params.get("channel") || (script && script.dataset.channel) || "",
        landing_url: window.location.href,
        referrer: document.referrer,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        keys: {
          p256dh: payload.keys.p256dh,
          auth: payload.keys.auth
        }
      })
    });
    if (!subscribeResponse.ok) return;
    var subscribeResult = await subscribeResponse.json();
    if (readyRegistration.active) {
      readyRegistration.active.postMessage({
        type: "push_booster_subscription",
        subscription_id: subscribeResult.subscription_id
      });
    }
  }());
}());
`, configURL), nil
}

func (s *Service) Subscribe(ctx context.Context, input SubscribeInput) (SubscribeResult, error) {
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	input.SubID = strings.TrimSpace(input.SubID)
	input.Channel = strings.TrimSpace(input.Channel)
	input.LandingURL = strings.TrimSpace(input.LandingURL)
	input.Referrer = strings.TrimSpace(input.Referrer)
	input.Targeting = normalizeTargeting(input.Targeting, input.UserAgent)
	if input.SourceID == "" {
		return SubscribeResult{}, errors.Join(ErrInvalidInput, errors.New("source_id is required"))
	}
	if input.Endpoint == "" {
		return SubscribeResult{}, errors.Join(ErrInvalidInput, errors.New("endpoint is required"))
	}
	if strings.TrimSpace(input.Keys.P256DH) == "" {
		return SubscribeResult{}, errors.Join(ErrInvalidInput, errors.New("keys.p256dh is required"))
	}
	if strings.TrimSpace(input.Keys.Auth) == "" {
		return SubscribeResult{}, errors.Join(ErrInvalidInput, errors.New("keys.auth is required"))
	}
	subscriptionID, err := generateSubscriptionID()
	if err != nil {
		return SubscribeResult{}, err
	}
	input.SubscriptionID = subscriptionID
	source, err := s.sources.GetSource(ctx, input.SourceID)
	if err != nil {
		return SubscribeResult{}, err
	}
	if err := validateSourceOrigin(source, input.RequestOrigin, input.RequestReferrer); err != nil {
		return SubscribeResult{}, err
	}
	if err := s.store.Save(ctx, input); err != nil {
		return SubscribeResult{}, err
	}
	return SubscribeResult{SubscriptionID: subscriptionID, Status: "accepted"}, nil
}

func normalizeTargeting(targeting Targeting, userAgent string) Targeting {
	targeting.IP = strings.TrimSpace(targeting.IP)
	targeting.Country = strings.TrimSpace(targeting.Country)
	targeting.Region = strings.TrimSpace(targeting.Region)
	targeting.City = strings.TrimSpace(targeting.City)
	targeting.Timezone = strings.TrimSpace(targeting.Timezone)
	targeting.Language = firstLanguage(targeting.Language)
	targeting.UAPlatform = cleanClientHint(targeting.UAPlatform)
	targeting.UAPlatformVersion = cleanClientHint(targeting.UAPlatformVersion)
	targeting.UAFullVersion = cleanClientHint(targeting.UAFullVersion)
	targeting.UAArch = cleanClientHint(targeting.UAArch)
	targeting.UABitness = cleanClientHint(targeting.UABitness)

	parsed := parseUserAgent(userAgent, targeting.UAPlatform, targeting.UAMobile)
	targeting.BrowserName = parsed.BrowserName
	targeting.BrowserVersion = parsed.BrowserVersion
	targeting.OSName = parsed.OSName
	targeting.OSVersion = parsed.OSVersion
	targeting.DeviceType = parsed.DeviceType
	targeting.DeviceVendor = parsed.DeviceVendor
	targeting.DeviceModel = parsed.DeviceModel
	return targeting
}

type parsedUserAgent struct {
	BrowserName    string
	BrowserVersion string
	OSName         string
	OSVersion      string
	DeviceType     string
	DeviceVendor   string
	DeviceModel    string
}

func parseUserAgent(
	userAgent string,
	platform string,
	mobileHint bool,
) parsedUserAgent {
	lowerUA := strings.ToLower(userAgent)
	parsed := parsedUserAgent{
		BrowserName: "unknown",
		OSName:      osName(lowerUA, platform),
		OSVersion:   osVersion(userAgent, lowerUA),
		DeviceType:  deviceType(lowerUA, mobileHint),
	}
	parsed.BrowserName, parsed.BrowserVersion = browser(userAgent, lowerUA)
	parsed.DeviceVendor, parsed.DeviceModel = deviceVendorModel(lowerUA)
	return parsed
}

func browser(userAgent string, lowerUA string) (string, string) {
	switch {
	case strings.Contains(lowerUA, "edg/"):
		return "edge", tokenVersion(userAgent, "Edg/")
	case strings.Contains(lowerUA, "opr/"):
		return "opera", tokenVersion(userAgent, "OPR/")
	case strings.Contains(lowerUA, "chrome/") && !strings.Contains(lowerUA, "chromium"):
		return "chrome", tokenVersion(userAgent, "Chrome/")
	case strings.Contains(lowerUA, "firefox/"):
		return "firefox", tokenVersion(userAgent, "Firefox/")
	case strings.Contains(lowerUA, "safari/") && strings.Contains(lowerUA, "version/"):
		return "safari", tokenVersion(userAgent, "Version/")
	case strings.Contains(lowerUA, "bot") || strings.Contains(lowerUA, "crawler") || strings.Contains(lowerUA, "spider"):
		return "bot", ""
	default:
		return "unknown", ""
	}
}

func osName(lowerUA string, platform string) string {
	platform = strings.ToLower(platform)
	switch {
	case strings.Contains(lowerUA, "android") || platform == "android":
		return "android"
	case strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "ipad") || platform == "ios":
		return "ios"
	case strings.Contains(lowerUA, "windows") || platform == "windows":
		return "windows"
	case strings.Contains(lowerUA, "mac os x") || platform == "macos":
		return "macos"
	case strings.Contains(lowerUA, "linux") || platform == "linux":
		return "linux"
	default:
		return "unknown"
	}
}

func osVersion(userAgent string, lowerUA string) string {
	switch {
	case strings.Contains(lowerUA, "android "):
		return tokenVersion(userAgent, "Android ")
	case strings.Contains(lowerUA, "cpu iphone os "):
		return strings.ReplaceAll(tokenVersion(userAgent, "CPU iPhone OS "), "_", ".")
	case strings.Contains(lowerUA, "cpu os "):
		return strings.ReplaceAll(tokenVersion(userAgent, "CPU OS "), "_", ".")
	case strings.Contains(lowerUA, "windows nt "):
		return tokenVersion(userAgent, "Windows NT ")
	case strings.Contains(lowerUA, "mac os x "):
		return strings.ReplaceAll(tokenVersion(userAgent, "Mac OS X "), "_", ".")
	default:
		return ""
	}
}

func deviceType(lowerUA string, mobileHint bool) string {
	switch {
	case strings.Contains(lowerUA, "bot") || strings.Contains(lowerUA, "crawler") || strings.Contains(lowerUA, "spider"):
		return "bot"
	case strings.Contains(lowerUA, "ipad") || strings.Contains(lowerUA, "tablet"):
		return "tablet"
	case mobileHint || strings.Contains(lowerUA, "mobi") || strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "android"):
		return "mobile"
	case lowerUA != "":
		return "desktop"
	default:
		return "unknown"
	}
}

func deviceVendorModel(lowerUA string) (string, string) {
	switch {
	case strings.Contains(lowerUA, "iphone"):
		return "apple", "iphone"
	case strings.Contains(lowerUA, "ipad"):
		return "apple", "ipad"
	case strings.Contains(lowerUA, "pixel"):
		return "google", ""
	case strings.Contains(lowerUA, "samsung") || strings.Contains(lowerUA, "sm-"):
		return "samsung", ""
	default:
		return "", ""
	}
}

func tokenVersion(userAgent string, token string) string {
	index := strings.Index(userAgent, token)
	if index < 0 {
		return ""
	}
	start := index + len(token)
	end := start
	for end < len(userAgent) {
		char := userAgent[end]
		if !((char >= '0' && char <= '9') || char == '.' || char == '_') {
			break
		}
		end++
	}
	return userAgent[start:end]
}

func firstLanguage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func cleanClientHint(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return value
}

func (s *Service) TrackServiceWorkerEvent(ctx context.Context, input ServiceWorkerEventInput) error {
	input.SubscriptionID = strings.TrimSpace(input.SubscriptionID)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.DeliveryID = strings.TrimSpace(input.DeliveryID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.CreativeID = strings.TrimSpace(input.CreativeID)
	input.EventID = strings.TrimSpace(input.EventID)
	input.URL = strings.TrimSpace(input.URL)
	input.EventType = strings.TrimSpace(input.EventType)
	if input.SubscriptionID == "" {
		return errors.Join(ErrInvalidInput, errors.New("subscription_id is required"))
	}
	if input.EventType == "" {
		return errors.Join(ErrInvalidInput, errors.New("event_type is required"))
	}
	if !validServiceWorkerEvent(input.EventType) {
		return errors.Join(ErrInvalidInput, errors.New("invalid event_type"))
	}
	sourceID, err := s.store.SourceIDForSubscription(ctx, input.SubscriptionID)
	if err != nil {
		return err
	}
	input.SourceID = sourceID
	source, err := s.sources.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}
	if err := validateSourceOrigin(source, input.RequestOrigin, input.RequestReferrer); err != nil {
		return err
	}
	if s.eventIDs != nil {
		allowed, err := s.eventIDs.AllowEvent(ctx, input)
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}
	}
	return s.store.SaveEvent(ctx, input)
}

func (s *Service) PushPayload(ctx context.Context, input PushPayloadRequest) (PushPayload, error) {
	input.TriggerID = strings.TrimSpace(input.TriggerID)
	if input.TriggerID == "" {
		return PushPayload{}, errors.Join(ErrInvalidInput, errors.New("trigger_id is required"))
	}
	if s.triggers != nil {
		trigger, err := s.triggers.ResolveTrigger(ctx, input.TriggerID)
		if err == nil {
			return s.pushPayloadForContext(
				ctx,
				input.TriggerID,
				trigger.DeliveryID,
				trigger.SubscriptionID,
				trigger.SourceID,
				trigger.CampaignID,
				input.UserAgent,
			)
		}
		if !errors.Is(err, inventory.ErrNotFound) {
			return PushPayload{}, err
		}
	}
	return s.pushPayloadForContext(
		ctx,
		input.TriggerID,
		"",
		input.TriggerID,
		"",
		"",
		input.UserAgent,
	)
}

func (s *Service) pushPayloadForContext(
	ctx context.Context,
	triggerID string,
	deliveryID string,
	subscriptionID string,
	sourceID string,
	campaignID string,
	userAgent string,
) (PushPayload, error) {
	decision := PayloadDecisionInput{
		TriggerID:      triggerID,
		SubscriptionID: subscriptionID,
		SourceID:       sourceID,
		CampaignID:     campaignID,
	}
	if sourceID == "" {
		resolvedSourceID, err := s.store.SourceIDForSubscription(ctx, subscriptionID)
		if err != nil {
			result := payloadDecisionError
			if errors.Is(err, inventory.ErrNotFound) {
				result = payloadDecisionNotFound
			}
			return PushPayload{}, s.recordPayloadDecision(
				ctx,
				decision,
				"",
				result,
				payloadDecisionReasonSourceLookupFailed,
				err,
			)
		}
		sourceID = resolvedSourceID
		decision.SourceID = sourceID
	}
	if s.creatives == nil {
		payload, err := s.store.PayloadForSubscription(ctx, subscriptionID, userAgent)
		if err != nil {
			return PushPayload{}, err
		}
		payload.TriggerID = triggerID
		payload.DeliveryID = deliveryID
		payload.SubscriptionID = subscriptionID
		payload.SourceID = sourceID
		payload.ClickURL = s.clickURL(deliveryID, payload.URL)
		return payload, nil
	}
	var creatives []inventory.Creative
	var err error
	if campaignID != "" {
		creatives, err = s.creatives.ActiveCreativesForCampaign(ctx, campaignID)
	} else {
		creatives, err = s.creatives.ActiveCreativesForSource(ctx, sourceID)
	}
	if err != nil {
		result := payloadDecisionError
		reason := payloadDecisionReasonCreativeLookup
		if errors.Is(err, inventory.ErrNotFound) {
			result = payloadDecisionNotFound
			reason = payloadDecisionReasonNoActiveCreative
		}
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			result,
			reason,
			err,
		)
	}
	if len(creatives) == 0 {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionNotFound,
			payloadDecisionReasonNoActiveCreative,
			inventory.ErrNotFound,
		)
	}
	decision.CampaignID = creatives[0].CampaignID
	creatives, err = s.filterCreativesByTargeting(
		ctx,
		subscriptionID,
		creatives,
	)
	if err != nil {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionError,
			payloadDecisionReasonTargetingLookup,
			err,
		)
	}
	if len(creatives) == 0 {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionSuppressed,
			payloadDecisionReasonCampaignTargeting,
			ErrPayloadSuppressed,
		)
	}
	orderedCreatives, err := s.orderCreativesByExposure(
		ctx,
		subscriptionID,
		creatives,
	)
	if err != nil {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionError,
			payloadDecisionReasonExposureLookup,
			err,
		)
	}
	selected, ok, err := s.firstAllowedCreative(
		ctx,
		subscriptionID,
		orderedCreatives,
	)
	if err != nil {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionError,
			payloadDecisionReasonCapCheck,
			err,
		)
	}
	if !ok {
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			"",
			payloadDecisionSuppressed,
			payloadDecisionReasonCapsExceeded,
			ErrPayloadSuppressed,
		)
	}
	if err := s.recordCreativeExposure(
		ctx,
		subscriptionID,
		sourceID,
		selected,
	); err != nil {
		decision.CampaignID = selected.CampaignID
		return PushPayload{}, s.recordPayloadDecision(
			ctx,
			decision,
			selected.ID,
			payloadDecisionError,
			payloadDecisionReasonExposureRecord,
			err,
		)
	}
	decision.CampaignID = selected.CampaignID
	if err := s.recordPayloadDecision(
		ctx,
		decision,
		selected.ID,
		payloadDecisionSelected,
		payloadDecisionReasonSelected,
		nil,
	); err != nil {
		return PushPayload{}, err
	}
	return PushPayload{
		Title:          selected.Title,
		Body:           selected.Body,
		Icon:           selected.Icon,
		URL:            selected.URL,
		ClickURL:       s.clickURL(deliveryID, selected.URL),
		TriggerID:      triggerID,
		DeliveryID:     deliveryID,
		SubscriptionID: subscriptionID,
		SourceID:       sourceID,
		CampaignID:     selected.CampaignID,
		CreativeID:     selected.ID,
	}, nil
}

func (s *Service) filterCreativesByTargeting(
	ctx context.Context,
	subscriptionID string,
	creatives []inventory.Creative,
) ([]inventory.Creative, error) {
	if !hasTargetingRules(creatives) {
		return creatives, nil
	}
	snapshot, err := s.store.TargetingForSubscription(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	filtered := make([]inventory.Creative, 0, len(creatives))
	for _, creative := range creatives {
		if matchesTargeting(snapshot, creative.CampaignTargetingRules) {
			filtered = append(filtered, creative)
		}
	}
	return filtered, nil
}

func hasTargetingRules(creatives []inventory.Creative) bool {
	for _, creative := range creatives {
		if targetingRulesEnabled(creative.CampaignTargetingRules) {
			return true
		}
	}
	return false
}

func targetingRulesEnabled(rules inventory.TargetingRules) bool {
	return len(rules.Countries) > 0 ||
		len(rules.Languages) > 0 ||
		len(rules.DeviceTypes) > 0 ||
		len(rules.OSNames) > 0 ||
		len(rules.BrowserNames) > 0
}

func matchesTargeting(
	snapshot SubscriberTargetingSnapshot,
	rules inventory.TargetingRules,
) bool {
	return matchesRuleValue(rules.Countries, snapshot.Country) &&
		matchesLanguage(rules.Languages, snapshot.Language) &&
		matchesRuleValue(rules.DeviceTypes, snapshot.DeviceType) &&
		matchesRuleValue(rules.OSNames, snapshot.OSName) &&
		matchesRuleValue(rules.BrowserNames, snapshot.BrowserName)
}

func matchesLanguage(allowed []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if matchesRuleValue(allowed, value) {
		return true
	}
	base, _, found := strings.Cut(value, "-")
	return found && matchesRuleValue(allowed, base)
}

func matchesRuleValue(allowed []string, value string) bool {
	if len(allowed) == 0 {
		return true
	}
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowed {
		if strings.ToLower(strings.TrimSpace(candidate)) == value {
			return true
		}
	}
	return false
}

func (s *Service) recordPayloadDecision(
	ctx context.Context,
	input PayloadDecisionInput,
	creativeID string,
	result string,
	reason string,
	decisionErr error,
) error {
	if s.decisions == nil {
		if decisionErr != nil {
			return decisionErr
		}
		return nil
	}
	input.CreativeID = creativeID
	input.Result = result
	input.Reason = reason
	input.OccurredAt = time.Now().UTC()
	if decisionErr != nil {
		input.Error = decisionErr.Error()
	}
	if err := s.decisions.RecordPayloadDecision(ctx, input); err != nil {
		if decisionErr != nil {
			return errors.Join(decisionErr, err)
		}
		return err
	}
	return decisionErr
}

func (s *Service) firstAllowedCreative(
	ctx context.Context,
	subscriptionID string,
	creatives []inventory.Creative,
) (inventory.Creative, bool, error) {
	for _, creative := range creatives {
		allowed, err := s.creativeAllowed(ctx, subscriptionID, creative)
		if err != nil {
			return inventory.Creative{}, false, err
		}
		if allowed {
			return creative, true, nil
		}
	}
	return inventory.Creative{}, false, nil
}

func (s *Service) orderCreativesByExposure(
	ctx context.Context,
	subscriptionID string,
	creatives []inventory.Creative,
) ([]inventory.Creative, error) {
	if s.exposures == nil || s.exposureWindow <= 0 {
		return creatives, nil
	}
	seen, err := s.exposures.SeenCreativeIDsSince(
		ctx,
		subscriptionID,
		creatives[0].CampaignID,
		time.Now().UTC().Add(-s.exposureWindow),
	)
	if err != nil {
		return nil, err
	}
	unseen := make([]inventory.Creative, 0, len(creatives))
	for _, creative := range creatives {
		if seen[creative.ID] {
			continue
		}
		unseen = append(unseen, creative)
	}
	if len(unseen) == 0 {
		return creatives, nil
	}
	return unseen, nil
}

func (s *Service) recordCreativeExposure(
	ctx context.Context,
	subscriptionID string,
	sourceID string,
	creative inventory.Creative,
) error {
	if s.exposures == nil {
		return nil
	}
	return s.exposures.RecordCreativeExposure(ctx, CreativeExposureInput{
		SubscriptionID: subscriptionID,
		SourceID:       sourceID,
		CampaignID:     creative.CampaignID,
		CreativeID:     creative.ID,
		OccurredAt:     time.Now().UTC(),
	})
}

func (s *Service) creativeAllowed(
	ctx context.Context,
	subscriptionID string,
	creative inventory.Creative,
) (bool, error) {
	if s.caps == nil {
		return true, nil
	}
	allowed, err := s.caps.Allow(ctx, CapCheckInput{
		SubscriptionID:          subscriptionID,
		CampaignID:              creative.CampaignID,
		CreativeID:              creative.ID,
		DailyCapPerSubscription: creative.CampaignDailyCapPerSubscription,
		TotalCapPerSubscription: creative.CampaignTotalCapPerSubscription,
		CreativeDailyCap:        creative.DailyCapPerSubscription,
		CreativeTotalCap:        creative.TotalCapPerSubscription,
	})
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func (s *Service) CreateTrigger(ctx context.Context, input CreateTriggerInput) (DeliveryTrigger, error) {
	if s.triggers == nil {
		return DeliveryTrigger{}, ErrTriggerStoreNotReady
	}
	input.SubscriptionID = strings.TrimSpace(input.SubscriptionID)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.SubscriptionID == "" {
		return DeliveryTrigger{}, errors.Join(ErrInvalidInput, errors.New("subscription_id is required"))
	}
	sourceID, err := s.store.SourceIDForSubscription(ctx, input.SubscriptionID)
	if err != nil {
		return DeliveryTrigger{}, err
	}
	if input.SourceID != "" && input.SourceID != sourceID {
		return DeliveryTrigger{}, errors.Join(ErrInvalidInput, errors.New("source_id does not match subscription"))
	}
	input.SourceID = sourceID
	if input.TTL == 0 {
		input.TTL = s.triggerTTL
	}
	if input.TTL == 0 {
		input.TTL = 5 * time.Minute
	}
	return s.triggers.CreateTrigger(ctx, input)
}

func (s *Service) CountBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.CountBySource(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) CountBySourceToday(ctx context.Context, sourceID string) (int64, error) {
	return s.store.CountBySourceToday(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) CountEventsBySourceToday(ctx context.Context, sourceID string) (int64, error) {
	return s.store.CountEventsBySourceToday(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) CountEventsBySourceTodayByType(ctx context.Context, sourceID string) (map[string]int64, error) {
	return s.store.CountEventsBySourceTodayByType(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) LastEventAtBySource(ctx context.Context, sourceID string) (string, error) {
	return s.store.LastEventAtBySource(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) CampaignReport(ctx context.Context, campaignID string) (CampaignReport, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignReport{}, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	return s.store.CampaignReport(ctx, campaignID)
}

func (s *Service) EstimateAudience(
	ctx context.Context,
	sourceIDs []string,
	rules inventory.TargetingRules,
) (int64, error) {
	sourceIDs = normalizeSourceIDs(sourceIDs)
	return s.store.EstimateAudience(ctx, sourceIDs, rules)
}

func (s *Service) BuildAudience(ctx context.Context, input inventory.BuildAudienceInput) (int64, error) {
	input.LaunchID = strings.TrimSpace(input.LaunchID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.SourceIDs = normalizeSourceIDs(input.SourceIDs)
	if input.LaunchID == "" {
		return 0, errors.Join(ErrInvalidInput, errors.New("launch_id is required"))
	}
	if input.CampaignID == "" {
		return 0, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	return s.store.BuildAudience(ctx, input)
}

func normalizeSourceIDs(sourceIDs []string) []string {
	normalized := make([]string, 0, len(sourceIDs))
	seen := map[string]bool{}
	for _, sourceID := range sourceIDs {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" || seen[sourceID] {
			continue
		}
		seen[sourceID] = true
		normalized = append(normalized, sourceID)
	}
	return normalized
}

func (s *Service) AudienceBatch(
	ctx context.Context,
	input inventory.AudienceBatchInput,
) ([]inventory.AudienceRow, error) {
	input.LaunchID = strings.TrimSpace(input.LaunchID)
	input.AfterSubscriptionID = strings.TrimSpace(input.AfterSubscriptionID)
	if input.LaunchID == "" {
		return nil, errors.Join(ErrInvalidInput, errors.New("launch_id is required"))
	}
	if input.Limit <= 0 {
		return nil, errors.Join(ErrInvalidInput, errors.New("limit must be greater than zero"))
	}
	return s.store.AudienceBatch(ctx, input)
}

func (s *Service) Timezones(ctx context.Context) ([]string, error) {
	return s.store.Timezones(ctx)
}

func (s *Service) endpoint(baseURL string, path string) string {
	if baseURL == "" {
		return path
	}
	return baseURL + path
}

func (s *Service) clickURL(deliveryID string, targetURL string) string {
	deliveryID = strings.TrimSpace(deliveryID)
	targetURL = strings.TrimSpace(targetURL)
	if deliveryID == "" || targetURL == "" {
		return targetURL
	}
	return s.endpoint(s.publicURL, "/api/click/"+deliveryID) +
		"?url=" +
		url.QueryEscape(targetURL)
}

func validServiceWorkerEvent(eventType string) bool {
	switch eventType {
	case "installed",
		"activated",
		"sdk_configured",
		"push_received",
		"payload_resolved",
		"payload_failed",
		"notification_shown",
		"notification_click",
		"notification_close",
		"subscription_changed":
		return true
	default:
		return false
	}
}

func validateSourceOrigin(
	source inventory.Source,
	origin string,
	referrer string,
) error {
	allowedDomain := normalizeDomain(source.Domain)
	if allowedDomain == "" {
		return nil
	}
	requestHost := requestHost(origin, referrer)
	if requestHost == "" || localHost(requestHost) {
		return nil
	}
	if requestHost == allowedDomain || strings.HasSuffix(requestHost, "."+allowedDomain) {
		return nil
	}
	return ErrForbiddenOrigin
}

func requestHost(origin string, referrer string) string {
	host := hostFromURL(origin)
	if host != "" {
		return host
	}
	return hostFromURL(referrer)
}

func hostFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		host = parsed.Host
	}
	return normalizeDomain(host)
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimSuffix(domain, ".")
	host, _, err := net.SplitHostPort(domain)
	if err == nil {
		domain = host
	}
	if strings.HasPrefix(domain, "www.") {
		return strings.TrimPrefix(domain, "www.")
	}
	return domain
}

func localHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func generateSubscriptionID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return encoded[0:8] + "-" +
		encoded[8:12] + "-" +
		encoded[12:16] + "-" +
		encoded[16:20] + "-" +
		encoded[20:32], nil
}
