package postbacks

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

type fakeConfigStore struct {
	configs []Config
	config  Config
	created CreateConfigInput
}

func (s *fakeConfigStore) CreateConfig(_ context.Context, input CreateConfigInput) (Config, error) {
	s.created = input
	return Config{
		ID:                  "config",
		Name:                input.Name,
		Token:               input.Token,
		Status:              StatusActive,
		ClickIDParam:        input.ClickIDParam,
		DeliveryIDParam:     input.DeliveryIDParam,
		SubscriptionIDParam: input.SubscriptionIDParam,
		ExternalIDParam:     input.ExternalIDParam,
		PayoutParam:         input.PayoutParam,
		CurrencyParam:       input.CurrencyParam,
		StatusParam:         input.StatusParam,
		DefaultCurrency:     input.DefaultCurrency,
	}, nil
}

func (s *fakeConfigStore) ListConfigs(_ context.Context) ([]Config, error) {
	return s.configs, nil
}

func (s *fakeConfigStore) GetConfig(_ context.Context, _ string) (Config, error) {
	if s.config.ID != "" {
		return s.config, nil
	}
	return Config{
		ID:                  "config",
		Name:                "Default",
		Token:               "secret",
		Status:              StatusActive,
		ClickIDParam:        "click_id",
		DeliveryIDParam:     "delivery_id",
		SubscriptionIDParam: "subscription_id",
		ExternalIDParam:     "external_id",
		PayoutParam:         "payout",
		CurrencyParam:       "currency",
		StatusParam:         "status",
		DefaultCurrency:     "USD",
	}, nil
}

func (s *fakeConfigStore) UpdateConfigStatus(
	_ context.Context,
	input UpdateConfigStatusInput,
) (Config, error) {
	return Config{ID: input.ID, Status: input.Status}, nil
}

type fakeEventStore struct {
	exists      bool
	recorded    Event
	recentInput RecentEventsInput
	attribution Attribution
}

func (s *fakeEventStore) RecordEvent(_ context.Context, event Event) error {
	s.recorded = event
	return nil
}

func (s *fakeEventStore) EventExists(
	_ context.Context,
	_ string,
	_ string,
) (bool, error) {
	return s.exists, nil
}

func (s *fakeEventStore) RecentEvents(_ context.Context, input RecentEventsInput) ([]Event, error) {
	s.recentInput = input
	return []Event{s.recorded}, nil
}

func (s *fakeEventStore) ResolveAttribution(
	_ context.Context,
	input Attribution,
) (Attribution, error) {
	if s.attribution.SourceID != "" {
		return s.attribution, nil
	}
	return input, nil
}

func TestServiceCreateConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	configs := &fakeConfigStore{}
	service := NewService(configs, &fakeEventStore{})
	cfg, err := service.CreateConfig(context.Background(), CreateConfigInput{Name: " Main "})
	if err != nil {
		t.Fatalf("create config failed: %v", err)
	}
	if cfg.Token == "" {
		t.Fatal("expected generated token")
	}
	if cfg.ExternalIDParam != "external_id" || cfg.DefaultCurrency != "USD" {
		t.Fatalf(
			"expected default params, got %+v",
			cfg,
		)
	}
}

func TestServiceIngestRecordsAttributedPostback(t *testing.T) {
	t.Parallel()

	events := &fakeEventStore{attribution: Attribution{
		DeliveryID:     "delivery",
		SubscriptionID: "subscription",
		SourceID:       "source",
		CampaignID:     "campaign",
		CreativeID:     "creative",
	}}
	service := NewService(&fakeConfigStore{}, events)
	event, err := service.Ingest(context.Background(), IngestInput{
		ConfigID: "config",
		Query: url.Values{
			"token":       []string{"secret"},
			"click_id":    []string{"delivery"},
			"external_id": []string{"tx-1"},
			"payout":      []string{"1.25"},
		},
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if event.Attribution != AttributionResolved || event.CampaignID != "campaign" {
		t.Fatalf(
			"expected attributed event, got %+v",
			event,
		)
	}
	if events.recorded.DedupeKey != "tx-1" || events.recorded.Payout != 1.25 {
		t.Fatalf("unexpected recorded event: %+v", events.recorded)
	}
}

func TestServiceIngestRejectsBadToken(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeConfigStore{}, &fakeEventStore{})
	_, err := service.Ingest(context.Background(), IngestInput{
		ConfigID: "config",
		Query:    url.Values{"token": []string{"bad"}},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestServiceIngestDetectsDuplicate(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeConfigStore{}, &fakeEventStore{exists: true})
	_, err := service.Ingest(context.Background(), IngestInput{
		ConfigID: "config",
		Query: url.Values{
			"token":       []string{"secret"},
			"external_id": []string{"tx-1"},
		},
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestServiceRecentEventsPassesConfigFilter(t *testing.T) {
	t.Parallel()

	events := &fakeEventStore{}
	service := NewService(&fakeConfigStore{}, events)
	_, err := service.RecentEvents(context.Background(), RecentEventsInput{
		PostbackConfigID: " config ",
		Limit:            25,
	})
	if err != nil {
		t.Fatalf("recent events failed: %v", err)
	}
	if events.recentInput.PostbackConfigID != "config" || events.recentInput.Limit != 25 {
		t.Fatalf("unexpected recent input: %+v", events.recentInput)
	}
}
