package subscribers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
)

type fakeSubscriberStore struct {
	saved     SubscribeInput
	event     ServiceWorkerEventInput
	sourceID  string
	targeting SubscriberTargetingSnapshot
}

type fakeTriggerStore struct {
	trigger DeliveryTrigger
	err     error
	created CreateTriggerInput
}

type fakeCreativeSelector struct {
	creatives []inventory.Creative
}

type fakeCapStore struct {
	allowed           bool
	blockedCreativeID string
	inputs            []CapCheckInput
}

type fakeExposureStore struct {
	seen     map[string]bool
	recorded []CreativeExposureInput
}

type fakeDecisionStore struct {
	recorded []PayloadDecisionInput
}

type fakeEventIDStore struct {
	allowed bool
	calls   int
}

func (s *fakeEventIDStore) AllowEvent(
	_ context.Context,
	_ ServiceWorkerEventInput,
) (bool, error) {
	s.calls++
	return s.allowed, nil
}

func (s *fakeSubscriberStore) Save(_ context.Context, input SubscribeInput) error {
	s.saved = input
	return nil
}

func (s *fakeSubscriberStore) SaveEvent(_ context.Context, input ServiceWorkerEventInput) error {
	if input.SubscriptionID == "" {
		return errors.New("missing subscription id")
	}
	s.event = input
	return nil
}

func (s *fakeSubscriberStore) SourceIDForSubscription(_ context.Context, _ string) (string, error) {
	if s.sourceID != "" {
		return s.sourceID, nil
	}
	return "source", nil
}

func (s *fakeSubscriberStore) TargetingForSubscription(
	_ context.Context,
	_ string,
) (SubscriberTargetingSnapshot, error) {
	if s.targeting != (SubscriberTargetingSnapshot{}) {
		return s.targeting, nil
	}
	return SubscriberTargetingSnapshot{
		Country:     "us",
		Language:    "en-US",
		DeviceType:  "mobile",
		OSName:      "ios",
		BrowserName: "safari",
	}, nil
}

func (s *fakeSubscriberStore) PayloadForSubscription(
	_ context.Context,
	subscriptionID string,
	_ string,
) (PushPayload, error) {
	return PushPayload{Title: "Title", URL: "/", SourceID: subscriptionID}, nil
}

func (s *fakeSubscriberStore) CountBySource(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *fakeSubscriberStore) CampaignReport(_ context.Context, campaignID string) (CampaignReport, error) {
	return CampaignReport{CampaignID: campaignID}, nil
}

func (s *fakeSubscriberStore) EstimateAudience(
	_ context.Context,
	_ []string,
	_ inventory.TargetingRules,
) (int64, error) {
	return 0, nil
}

func (s *fakeSubscriberStore) BuildAudience(_ context.Context, _ inventory.BuildAudienceInput) (int64, error) {
	return 0, nil
}

func (s *fakeSubscriberStore) Timezones(_ context.Context) ([]string, error) {
	return []string{"UTC"}, nil
}

func (s *fakeSubscriberStore) AudienceBatch(
	_ context.Context,
	_ inventory.AudienceBatchInput,
) ([]inventory.AudienceRow, error) {
	return nil, nil
}

func TestServicePushPayloadValidatesTriggerID(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServicePushPayloadResolvesTrigger(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{TriggerID: "trigger", SubscriptionID: "subscription"},
	}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{TriggerStore: triggers})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.SourceID != "source" {
		t.Fatalf("expected resolved payload source, got %q", payload.SourceID)
	}
	if payload.DeliveryID != "" {
		t.Fatalf("expected empty fallback delivery id, got %q", payload.DeliveryID)
	}
}

func TestServicePushPayloadFallsBackToSubscriptionID(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{err: inventory.ErrNotFound}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{TriggerStore: triggers})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "subscription"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.SourceID != "source" {
		t.Fatalf("expected fallback resolved source, got %q", payload.SourceID)
	}
}

func TestServicePushPayloadSelectsCreativeForTriggerCampaign(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{{
		ID:         "creative",
		CampaignID: "campaign",
		Title:      "Creative title",
		Body:       "Creative body",
		URL:        "https://example.com/click",
		Icon:       "https://example.com/icon.png",
	}}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
	})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.Title != "Creative title" || payload.CreativeID != "creative" || payload.CampaignID != "campaign" {
		t.Fatalf("unexpected creative payload: %+v", payload)
	}
	if payload.TriggerID != "trigger" || payload.SubscriptionID != "subscription" {
		t.Fatalf("unexpected payload context ids: %+v", payload)
	}
}

func TestServicePushPayloadRecordsSelectedDecision(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	decisions := &fakeDecisionStore{}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{{
		ID:         "creative",
		CampaignID: "campaign",
		Title:      "Creative title",
		Body:       "Creative body",
		URL:        "https://example.com/click",
	}}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
		Decisions:    decisions,
	})
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if len(decisions.recorded) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions.recorded))
	}
	decision := decisions.recorded[0]
	if decision.Result != payloadDecisionSelected || decision.Reason != payloadDecisionReasonSelected {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if decision.TriggerID != "trigger" || decision.SubscriptionID != "subscription" || decision.CreativeID != "creative" {
		t.Fatalf("unexpected decision ids: %+v", decision)
	}
}

func TestServicePushPayloadSuppressesWhenCapExceeded(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	caps := &fakeCapStore{allowed: false}
	decisions := &fakeDecisionStore{}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{{
		ID:                      "creative",
		CampaignID:              "campaign",
		Title:                   "Creative title",
		Body:                    "Creative body",
		URL:                     "https://example.com/click",
		DailyCapPerSubscription: 1,
	}}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
		Caps:         caps,
		Decisions:    decisions,
	})
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if !errors.Is(err, ErrPayloadSuppressed) {
		t.Fatalf("expected ErrPayloadSuppressed, got %v", err)
	}
	if len(caps.inputs) != 1 {
		t.Fatalf("expected one cap check, got %d", len(caps.inputs))
	}
	if caps.inputs[0].SubscriptionID != "subscription" || caps.inputs[0].CampaignID != "campaign" || caps.inputs[0].CreativeID != "creative" || caps.inputs[0].CreativeDailyCap != 1 {
		t.Fatalf("unexpected cap input: %+v", caps.inputs[0])
	}
	if len(decisions.recorded) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions.recorded))
	}
	decision := decisions.recorded[0]
	if decision.Result != payloadDecisionSuppressed || decision.Reason != payloadDecisionReasonCapsExceeded {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestServicePushPayloadRecordsNotFoundDecision(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	decisions := &fakeDecisionStore{}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    fakeCreativeSelector{},
		Decisions:    decisions,
	})
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if !errors.Is(err, inventory.ErrNotFound) {
		t.Fatalf("expected inventory.ErrNotFound, got %v", err)
	}
	if len(decisions.recorded) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions.recorded))
	}
	decision := decisions.recorded[0]
	if decision.Result != payloadDecisionNotFound || decision.Reason != payloadDecisionReasonNoActiveCreative {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestServicePushPayloadSuppressesCampaignTargetingMismatch(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{
		targeting: SubscriberTargetingSnapshot{
			Country:     "de",
			Language:    "de-DE",
			DeviceType:  "desktop",
			OSName:      "windows",
			BrowserName: "chrome",
		},
	}
	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	decisions := &fakeDecisionStore{}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{{
		ID:         "creative",
		CampaignID: "campaign",
		Title:      "Creative title",
		Body:       "Creative body",
		URL:        "https://example.com/click",
		CampaignTargetingRules: inventory.TargetingRules{
			Countries:   []string{"us"},
			DeviceTypes: []string{"mobile"},
		},
	}}}
	service := NewServiceWithConfig(store, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
		Decisions:    decisions,
	})
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if !errors.Is(err, ErrPayloadSuppressed) {
		t.Fatalf("expected ErrPayloadSuppressed, got %v", err)
	}
	if len(decisions.recorded) != 1 {
		t.Fatalf("expected one decision, got %d", len(decisions.recorded))
	}
	decision := decisions.recorded[0]
	if decision.Result != payloadDecisionSuppressed || decision.Reason != payloadDecisionReasonCampaignTargeting {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestServicePushPayloadAllowsCampaignTargetingMatch(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{
		targeting: SubscriberTargetingSnapshot{
			Country:     "us",
			Language:    "en-US",
			DeviceType:  "mobile",
			OSName:      "ios",
			BrowserName: "safari",
		},
	}
	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{{
		ID:         "creative",
		CampaignID: "campaign",
		Title:      "Creative title",
		Body:       "Creative body",
		URL:        "https://example.com/click",
		CampaignTargetingRules: inventory.TargetingRules{
			Countries:    []string{"us"},
			Languages:    []string{"en"},
			DeviceTypes:  []string{"mobile"},
			OSNames:      []string{"ios"},
			BrowserNames: []string{"safari"},
		},
	}}}
	service := NewServiceWithConfig(store, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
	})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.CreativeID != "creative" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestServicePushPayloadFallsBackToNextCreativeWhenCreativeCapExceeded(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	caps := &fakeCapStore{allowed: true, blockedCreativeID: "creative-capped"}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{
		{
			ID:                      "creative-capped",
			CampaignID:              "campaign",
			Title:                   "Capped creative",
			Body:                    "Creative body",
			URL:                     "https://example.com/capped",
			DailyCapPerSubscription: 1,
		},
		{
			ID:         "creative-open",
			CampaignID: "campaign",
			Title:      "Open creative",
			Body:       "Creative body",
			URL:        "https://example.com/open",
		},
	}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore: triggers,
		Creatives:    creatives,
		Caps:         caps,
	})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.CreativeID != "creative-open" || payload.Title != "Open creative" {
		t.Fatalf("unexpected creative payload: %+v", payload)
	}
	if len(caps.inputs) != 2 {
		t.Fatalf("expected two cap checks, got %d", len(caps.inputs))
	}
	if caps.inputs[0].CreativeID != "creative-capped" || caps.inputs[1].CreativeID != "creative-open" {
		t.Fatalf("unexpected cap check order: %+v", caps.inputs)
	}
}

func TestServicePushPayloadPrefersUnseenCreative(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	exposures := &fakeExposureStore{seen: map[string]bool{"creative-seen": true}}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{
		{
			ID:         "creative-seen",
			CampaignID: "campaign",
			Title:      "Seen creative",
			Body:       "Creative body",
			URL:        "https://example.com/seen",
		},
		{
			ID:         "creative-unseen",
			CampaignID: "campaign",
			Title:      "Unseen creative",
			Body:       "Creative body",
			URL:        "https://example.com/unseen",
		},
	}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore:   triggers,
		Creatives:      creatives,
		Exposures:      exposures,
		ExposureWindow: time.Hour,
	})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.CreativeID != "creative-unseen" {
		t.Fatalf("expected unseen creative, got %+v", payload)
	}
	if len(exposures.recorded) != 1 || exposures.recorded[0].CreativeID != "creative-unseen" {
		t.Fatalf("expected exposure record for unseen creative, got %+v", exposures.recorded)
	}
}

func TestServicePushPayloadStartsNewCycleWhenAllCreativesSeen(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	exposures := &fakeExposureStore{seen: map[string]bool{
		"creative-one": true,
		"creative-two": true,
	}}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{
		{
			ID:         "creative-one",
			CampaignID: "campaign",
			Title:      "Creative one",
			Body:       "Creative body",
			URL:        "https://example.com/one",
		},
		{
			ID:         "creative-two",
			CampaignID: "campaign",
			Title:      "Creative two",
			Body:       "Creative body",
			URL:        "https://example.com/two",
		},
	}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore:   triggers,
		Creatives:      creatives,
		Exposures:      exposures,
		ExposureWindow: time.Hour,
	})
	payload, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if err != nil {
		t.Fatalf("push payload failed: %v", err)
	}
	if payload.CreativeID != "creative-one" {
		t.Fatalf("expected new cycle to start from first creative, got %+v", payload)
	}
}

func TestServicePushPayloadSuppressesWhenOnlyUnseenCreativeIsCapped(t *testing.T) {
	t.Parallel()

	triggers := &fakeTriggerStore{
		trigger: DeliveryTrigger{
			TriggerID:      "trigger",
			SubscriptionID: "subscription",
			SourceID:       "source",
			CampaignID:     "campaign",
		},
	}
	caps := &fakeCapStore{allowed: true, blockedCreativeID: "creative-unseen"}
	exposures := &fakeExposureStore{seen: map[string]bool{"creative-seen": true}}
	creatives := fakeCreativeSelector{creatives: []inventory.Creative{
		{
			ID:         "creative-seen",
			CampaignID: "campaign",
			Title:      "Seen creative",
			Body:       "Creative body",
			URL:        "https://example.com/seen",
		},
		{
			ID:         "creative-unseen",
			CampaignID: "campaign",
			Title:      "Unseen creative",
			Body:       "Creative body",
			URL:        "https://example.com/unseen",
		},
	}}
	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		TriggerStore:   triggers,
		Creatives:      creatives,
		Caps:           caps,
		Exposures:      exposures,
		ExposureWindow: time.Hour,
	})
	_, err := service.PushPayload(context.Background(), PushPayloadRequest{TriggerID: "trigger"})
	if !errors.Is(err, ErrPayloadSuppressed) {
		t.Fatalf("expected ErrPayloadSuppressed, got %v", err)
	}
	if len(caps.inputs) != 1 || caps.inputs[0].CreativeID != "creative-unseen" {
		t.Fatalf("expected only unseen creative cap check, got %+v", caps.inputs)
	}
	if len(exposures.recorded) != 0 {
		t.Fatalf("expected no exposure record, got %+v", exposures.recorded)
	}
}

func (s *fakeSubscriberStore) CountBySourceToday(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *fakeSubscriberStore) CountEventsBySourceToday(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *fakeSubscriberStore) CountEventsBySourceTodayByType(_ context.Context, _ string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (s *fakeSubscriberStore) LastEventAtBySource(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s *fakeTriggerStore) CreateTrigger(_ context.Context, input CreateTriggerInput) (DeliveryTrigger, error) {
	s.created = input
	if s.err != nil {
		return DeliveryTrigger{}, s.err
	}
	trigger := s.trigger
	if trigger.TriggerID == "" {
		trigger.TriggerID = "trigger"
	}
	if trigger.SubscriptionID == "" {
		trigger.SubscriptionID = input.SubscriptionID
	}
	if trigger.SourceID == "" {
		trigger.SourceID = input.SourceID
	}
	return trigger, nil
}

func (s *fakeTriggerStore) ResolveTrigger(_ context.Context, _ string) (DeliveryTrigger, error) {
	if s.err != nil {
		return DeliveryTrigger{}, s.err
	}
	return s.trigger, nil
}

func (s fakeCreativeSelector) ActiveCreativesForCampaign(_ context.Context, _ string) ([]inventory.Creative, error) {
	return s.creatives, nil
}

func (s fakeCreativeSelector) ActiveCreativesForSource(_ context.Context, sourceID string) ([]inventory.Creative, error) {
	creatives := make([]inventory.Creative, len(s.creatives))
	copy(creatives, s.creatives)
	for i := range creatives {
		if creatives[i].CampaignID == "" {
			creatives[i].CampaignID = sourceID
		}
	}
	return creatives, nil
}

func (s *fakeCapStore) Allow(_ context.Context, input CapCheckInput) (bool, error) {
	s.inputs = append(s.inputs, input)
	if input.CreativeID == s.blockedCreativeID {
		return false, nil
	}
	return s.allowed, nil
}

func (s *fakeExposureStore) SeenCreativeIDsSince(
	_ context.Context,
	_ string,
	_ string,
	_ time.Time,
) (map[string]bool, error) {
	return s.seen, nil
}

func (s *fakeExposureStore) RecordCreativeExposure(_ context.Context, input CreativeExposureInput) error {
	s.recorded = append(s.recorded, input)
	return nil
}

func (s *fakeDecisionStore) RecordPayloadDecision(_ context.Context, input PayloadDecisionInput) error {
	s.recorded = append(s.recorded, input)
	return nil
}

type fakeSourceStore struct{}

func (s fakeSourceStore) GetSource(_ context.Context, id string) (inventory.Source, error) {
	if id == "missing" {
		return inventory.Source{}, inventory.ErrNotFound
	}
	return inventory.Source{ID: id, Domain: "example.com"}, nil
}

func (s fakeSourceStore) ActiveVAPIDKeyForSource(
	_ context.Context,
	sourceID string,
) (inventory.VAPIDKey, error) {
	if sourceID == "unbound" {
		return inventory.VAPIDKey{}, inventory.ErrNotFound
	}
	return inventory.VAPIDKey{ID: "key", PublicKey: "source-public-key", Status: inventory.VAPIDStatusActive}, nil
}

func TestServiceSubscribeValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	tests := []SubscribeInput{
		{},
		{SourceID: "source", Endpoint: "endpoint"},
		{SourceID: "source", Endpoint: "endpoint", Keys: Keys{P256DH: "key"}},
	}
	for _, tt := range tests {
		_, err := service.Subscribe(context.Background(), tt)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	}
}

func TestServiceSubscribeSavesValidInput(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{}
	service := NewService(store, fakeSourceStore{}, "public-key")
	input := SubscribeInput{
		SourceID:      "source",
		Endpoint:      "endpoint",
		SubID:         " sub-1 ",
		Channel:       " news ",
		LandingURL:    " https://example.com/page ",
		Referrer:      " https://ref.example/path ",
		UserAgent:     "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
		RequestOrigin: "https://sub.example.com",
		Targeting: Targeting{
			IP:         "192.0.2.10",
			Language:   "en-US,en;q=0.9",
			UAPlatform: `"iOS"`,
			UAMobile:   true,
		},
		Keys: Keys{
			P256DH: "p256dh",
			Auth:   "auth",
		},
	}
	result, err := service.Subscribe(context.Background(), input)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if result.SubscriptionID == "" || result.Status != "accepted" {
		t.Fatalf("unexpected subscribe result: %+v", result)
	}
	if store.saved.Endpoint != input.Endpoint {
		t.Fatalf("expected saved endpoint %q, got %q", input.Endpoint, store.saved.Endpoint)
	}
	if store.saved.SubscriptionID == "" {
		t.Fatal("expected generated subscription id")
	}
	if store.saved.SubID != "sub-1" || store.saved.Channel != "news" {
		t.Fatalf("expected trimmed attribution, got %+v", store.saved)
	}
	if store.saved.Targeting.DeviceType != "mobile" || store.saved.Targeting.OSName != "ios" {
		t.Fatalf("expected normalized targeting, got %+v", store.saved.Targeting)
	}
	if store.saved.Targeting.Language != "en-US" {
		t.Fatalf("expected first language, got %q", store.saved.Targeting.Language)
	}
}

func TestServiceSubscribeRejectsForeignOrigin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	_, err := service.Subscribe(context.Background(), SubscribeInput{
		SourceID:      "source",
		Endpoint:      "endpoint",
		RequestOrigin: "https://evil.test",
		Keys: Keys{
			P256DH: "p256dh",
			Auth:   "auth",
		},
	})
	if !errors.Is(err, ErrForbiddenOrigin) {
		t.Fatalf("expected ErrForbiddenOrigin, got %v", err)
	}
}

func TestServiceVAPIDPublicKey(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "")
	if _, err := service.VAPIDPublicKey(); !errors.Is(err, ErrVAPIDNotReady) {
		t.Fatalf("expected ErrVAPIDNotReady, got %v", err)
	}

	service = NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	key, err := service.VAPIDPublicKey()
	if err != nil {
		t.Fatalf("vapid key failed: %v", err)
	}
	if key != "public-key" {
		t.Fatalf("expected public-key, got %q", key)
	}
}

func TestServiceSDKConfigUsesSourceVAPIDKey(t *testing.T) {
	t.Parallel()

	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		PublicURL:  "https://public.example",
		PayloadURL: "https://payload.example",
	})
	cfg, err := service.SDKConfig(context.Background(), SDKConfigInput{
		SourceID:      " source ",
		RequestOrigin: "https://example.com",
	})
	if err != nil {
		t.Fatalf("sdk config failed: %v", err)
	}
	if cfg.SourceID != "source" {
		t.Fatalf("expected source id, got %q", cfg.SourceID)
	}
	if cfg.VAPIDPublicKey != "source-public-key" {
		t.Fatalf("expected source vapid key, got %q", cfg.VAPIDPublicKey)
	}
	if cfg.SubscribeEndpoint != "https://public.example/api/subscribe" {
		t.Fatalf("unexpected subscribe endpoint %q", cfg.SubscribeEndpoint)
	}
	if cfg.PushPayloadEndpoint != "https://payload.example/api/push/payload" {
		t.Fatalf("unexpected payload endpoint %q", cfg.PushPayloadEndpoint)
	}
	if cfg.EventsEndpoint != "https://public.example/api/sw/events" {
		t.Fatalf("unexpected events endpoint %q", cfg.EventsEndpoint)
	}
	if cfg.ServiceWorkerURL != "/push-booster-sw.js" {
		t.Fatalf("unexpected service worker URL %q", cfg.ServiceWorkerURL)
	}
}

func TestServiceSDKScriptUsesHostedSourceConfig(t *testing.T) {
	t.Parallel()

	service := NewServiceWithConfig(&fakeSubscriberStore{}, fakeSourceStore{}, Config{
		PublicURL:  "https://public.example",
		PayloadURL: "https://payload.example",
	})
	script, err := service.SDKScript(context.Background(), SDKScriptInput{
		SourceID:        "source",
		RequestReferrer: "https://example.com/page",
	})
	if err != nil {
		t.Fatalf("sdk script failed: %v", err)
	}
	for _, part := range []string{
		"PushManager",
		"https://public.example/api/sdk/config?source_id=source",
		"timezone",
		"document.currentScript",
	} {
		if !strings.Contains(script, part) {
			t.Fatalf("expected sdk script to contain %q", part)
		}
	}
}

func TestServiceSDKConfigRequiresBoundVAPIDKey(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	_, err := service.SDKConfig(context.Background(), SDKConfigInput{SourceID: "unbound"})
	if !errors.Is(err, inventory.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceSDKConfigRejectsForeignOrigin(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	_, err := service.SDKConfig(context.Background(), SDKConfigInput{
		SourceID:      "source",
		RequestOrigin: "https://attacker.example",
	})
	if !errors.Is(err, ErrForbiddenOrigin) {
		t.Fatalf("expected ErrForbiddenOrigin, got %v", err)
	}
}

func TestServiceTrackServiceWorkerEventValidates(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSubscriberStore{}, fakeSourceStore{}, "public-key")
	tests := []ServiceWorkerEventInput{
		{},
		{SubscriptionID: "subscription"},
		{SubscriptionID: "subscription", EventType: "unknown"},
	}
	for _, tt := range tests {
		err := service.TrackServiceWorkerEvent(context.Background(), tt)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	}
}

func TestServiceTrackServiceWorkerEventSavesValidInput(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{sourceID: "source"}
	service := NewService(store, fakeSourceStore{}, "public-key")
	err := service.TrackServiceWorkerEvent(context.Background(), ServiceWorkerEventInput{
		SubscriptionID: "subscription",
		DeliveryID:     "delivery",
		CampaignID:     "campaign",
		CreativeID:     "creative",
		EventType:      "notification_shown",
		RequestOrigin:  "https://example.com",
	})
	if err != nil {
		t.Fatalf("track service worker event failed: %v", err)
	}
	if store.event.SourceID != "source" || store.event.DeliveryID != "delivery" {
		t.Fatalf("unexpected saved event: %+v", store.event)
	}
}

func TestServiceTrackServiceWorkerEventSkipsDuplicate(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{sourceID: "source"}
	eventIDs := &fakeEventIDStore{allowed: false}
	service := NewServiceWithConfig(store, fakeSourceStore{}, Config{EventIDs: eventIDs})
	err := service.TrackServiceWorkerEvent(context.Background(), ServiceWorkerEventInput{
		SubscriptionID: "subscription",
		DeliveryID:     "delivery",
		EventType:      "notification_click",
		RequestOrigin:  "https://example.com",
	})
	if err != nil {
		t.Fatalf("track service worker event failed: %v", err)
	}
	if eventIDs.calls != 1 {
		t.Fatalf("expected idempotency check, got %d", eventIDs.calls)
	}
	if store.event.SubscriptionID != "" {
		t.Fatalf("expected duplicate event to be skipped, got %+v", store.event)
	}
}

func TestServiceCreateTrigger(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{sourceID: "source"}
	triggers := &fakeTriggerStore{}
	service := NewServiceWithConfig(store, fakeSourceStore{}, Config{TriggerStore: triggers})
	trigger, err := service.CreateTrigger(context.Background(), CreateTriggerInput{
		SubscriptionID: "subscription",
	})
	if err != nil {
		t.Fatalf("create trigger failed: %v", err)
	}
	if trigger.TriggerID == "" {
		t.Fatal("expected trigger id")
	}
	if triggers.created.SourceID != "source" {
		t.Fatalf("expected resolved source, got %q", triggers.created.SourceID)
	}
}

func TestServiceCreateTriggerRejectsSourceMismatch(t *testing.T) {
	t.Parallel()

	store := &fakeSubscriberStore{sourceID: "source"}
	triggers := &fakeTriggerStore{}
	service := NewServiceWithConfig(store, fakeSourceStore{}, Config{TriggerStore: triggers})
	_, err := service.CreateTrigger(context.Background(), CreateTriggerInput{
		SubscriptionID: "subscription",
		SourceID:       "other-source",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
