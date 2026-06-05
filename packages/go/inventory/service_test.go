package inventory

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
)

type fakeStore struct {
	publisherInput        CreatePublisherInput
	sourceInput           CreateSourceInput
	vapidKey              VAPIDKey
	attachInput           AttachVAPIDKeyInput
	campaignInput         CreateCampaignInput
	creativeInput         CreateCreativeInput
	providerConfigInput   CreateCreativeProviderConfigInput
	providerCreativeInput UpsertProviderCreativeInput
}

func (s *fakeStore) CreatePublisher(_ context.Context, input CreatePublisherInput) (Publisher, error) {
	s.publisherInput = input
	return Publisher{ID: "publisher", Name: input.Name, Status: StatusActive}, nil
}

func (s *fakeStore) ListPublishers(_ context.Context) ([]Publisher, error) {
	return nil, nil
}

func (s *fakeStore) CreateSource(_ context.Context, input CreateSourceInput) (Source, error) {
	s.sourceInput = input
	return Source{
		ID:          "source",
		PublisherID: input.PublisherID,
		Name:        input.Name,
		Domain:      input.Domain,
		Status:      StatusActive,
	}, nil
}

func (s *fakeStore) ListSources(_ context.Context, _ string) ([]Source, error) {
	return nil, nil
}

func (s *fakeStore) GetSource(_ context.Context, id string) (Source, error) {
	return Source{ID: id, Status: StatusActive}, nil
}

func (s *fakeStore) CreateVAPIDKey(_ context.Context, key VAPIDKey) (VAPIDKey, error) {
	s.vapidKey = key
	key.ID = "vapid-key"
	return key, nil
}

func (s *fakeStore) ListVAPIDKeys(_ context.Context) ([]VAPIDKey, error) {
	return nil, nil
}

func (s *fakeStore) UpdateVAPIDKeyStatus(_ context.Context, id string, status string) (VAPIDKey, error) {
	return VAPIDKey{ID: id, Status: status}, nil
}

func (s *fakeStore) AttachVAPIDKeyToSource(_ context.Context, input AttachVAPIDKeyInput) (Source, error) {
	s.attachInput = input
	return Source{ID: input.SourceID, VAPIDKeyID: input.VAPIDKeyID, Status: StatusActive}, nil
}

func (s *fakeStore) CreateCampaign(_ context.Context, input CreateCampaignInput) (Campaign, error) {
	s.campaignInput = input
	return Campaign{
		ID:                      "campaign",
		PublisherID:             input.PublisherID,
		SourceID:                input.SourceID,
		SourceIDs:               input.SourceIDs,
		AudienceScope:           input.AudienceScope,
		Name:                    input.Name,
		Status:                  CampaignStatusDraft,
		TargetingRules:          input.TargetingRules,
		DailyCapPerSubscription: input.DailyCapPerSubscription,
		TotalCapPerSubscription: input.TotalCapPerSubscription,
	}, nil
}

func (s *fakeStore) ListCampaigns(_ context.Context, _ string) ([]Campaign, error) {
	return nil, nil
}

func (s *fakeStore) GetCampaign(_ context.Context, id string) (Campaign, error) {
	return Campaign{
		ID:            id,
		SourceID:      "source",
		SourceIDs:     []string{"source"},
		AudienceScope: CampaignAudienceScopeSelectedSources,
		Status:        CampaignStatusActive,
	}, nil
}

func (s *fakeStore) UpdateCampaignStatus(_ context.Context, input UpdateCampaignStatusInput) (Campaign, error) {
	return Campaign{ID: input.ID, Status: input.Status}, nil
}

func (s *fakeStore) CreateCampaignLaunch(_ context.Context, input CreateCampaignLaunchInput) (CampaignLaunch, error) {
	return CampaignLaunch{
		ID:            "launch",
		CampaignID:    input.CampaignID,
		Status:        CampaignLaunchStatusBuilding,
		EnqueueStatus: EnqueueStatusPending,
	}, nil
}

func (s *fakeStore) GetCampaignLaunch(_ context.Context, id string) (CampaignLaunch, error) {
	return CampaignLaunch{
		ID:             id,
		CampaignID:     "campaign",
		Status:         CampaignLaunchStatusCompleted,
		AudienceTotal:  2,
		ProcessedTotal: 2,
		EnqueueStatus:  EnqueueStatusPending,
	}, nil
}

func (s *fakeStore) ListCampaignLaunches(_ context.Context, campaignID string) ([]CampaignLaunch, error) {
	return []CampaignLaunch{{ID: "launch", CampaignID: campaignID}}, nil
}

func (s *fakeStore) UpdateCampaignLaunch(_ context.Context, input UpdateCampaignLaunchInput) (CampaignLaunch, error) {
	return CampaignLaunch{
		ID:             input.ID,
		Status:         input.Status,
		AudienceTotal:  input.AudienceTotal,
		ProcessedTotal: input.ProcessedTotal,
		ErrorMessage:   input.ErrorMessage,
		EnqueueStatus:  EnqueueStatusPending,
	}, nil
}

func (s *fakeStore) UpdateCampaignLaunchEnqueue(
	_ context.Context,
	input UpdateCampaignLaunchEnqueueInput,
) (CampaignLaunch, error) {
	return CampaignLaunch{
		ID:            input.ID,
		CampaignID:    "campaign",
		Status:        CampaignLaunchStatusCompleted,
		EnqueueStatus: input.EnqueueStatus,
		EnqueuedTotal: input.EnqueuedTotal,
		EnqueueError:  input.EnqueueError,
	}, nil
}

func (s *fakeStore) ListCampaignSchedules(_ context.Context, campaignID string) ([]CampaignSchedule, error) {
	return []CampaignSchedule{{ID: "schedule", CampaignID: campaignID, Status: ScheduleStatusActive}}, nil
}

func (s *fakeStore) ListActiveCampaignSchedules(_ context.Context) ([]CampaignSchedule, error) {
	return nil, nil
}

func (s *fakeStore) CreateCampaignSchedule(
	_ context.Context,
	input CreateCampaignScheduleInput,
) (CampaignSchedule, error) {
	return CampaignSchedule{
		ID:               "schedule",
		CampaignID:       input.CampaignID,
		Status:           input.Status,
		FallbackTimezone: input.FallbackTimezone,
		GraceMinutes:     input.GraceMinutes,
	}, nil
}

func (s *fakeStore) UpdateCampaignScheduleStatus(
	_ context.Context,
	input UpdateCampaignScheduleStatusInput,
) (CampaignSchedule, error) {
	return CampaignSchedule{ID: input.ID, Status: input.Status}, nil
}

func (s *fakeStore) CreateScheduleRun(
	_ context.Context,
	input CreateScheduleRunInput,
) (CampaignScheduleRun, bool, error) {
	return CampaignScheduleRun{
		ID:         "schedule-run",
		ScheduleID: input.ScheduleID,
		SlotID:     input.SlotID,
		CampaignID: input.CampaignID,
		Timezone:   input.Timezone,
	}, true, nil
}

func (s *fakeStore) CompleteScheduleRun(
	_ context.Context,
	input CompleteScheduleRunInput,
) (CampaignScheduleRun, error) {
	return CampaignScheduleRun{
		ID:            input.ID,
		Status:        input.Status,
		AudienceTotal: input.AudienceTotal,
		EnqueuedTotal: input.EnqueuedTotal,
	}, nil
}

func (s *fakeStore) ListCampaignScheduleRuns(_ context.Context, campaignID string) ([]CampaignScheduleRun, error) {
	return []CampaignScheduleRun{{ID: "schedule-run", CampaignID: campaignID}}, nil
}

func (s *fakeStore) CreateCreative(_ context.Context, input CreateCreativeInput) (Creative, error) {
	s.creativeInput = input
	return Creative{
		ID:                      "creative",
		CampaignID:              input.CampaignID,
		Title:                   input.Title,
		Body:                    input.Body,
		URL:                     input.URL,
		Icon:                    input.Icon,
		Status:                  CreativeStatusActive,
		DailyCapPerSubscription: input.DailyCapPerSubscription,
		TotalCapPerSubscription: input.TotalCapPerSubscription,
	}, nil
}

func (s *fakeStore) ListCreatives(_ context.Context, _ string) ([]Creative, error) {
	return nil, nil
}

func (s *fakeStore) UpdateCreativeStatus(_ context.Context, input UpdateCreativeStatusInput) (Creative, error) {
	return Creative{ID: input.ID, Status: input.Status}, nil
}

func (s *fakeStore) CreateCreativeProviderConfig(
	_ context.Context,
	input CreateCreativeProviderConfigInput,
) (CreativeProviderConfig, error) {
	s.providerConfigInput = input
	return CreativeProviderConfig{
		ID:             "provider-config",
		CampaignID:     input.CampaignID,
		Name:           input.Name,
		ProviderName:   input.ProviderName,
		FetchURL:       input.FetchURL,
		RequestHeaders: input.RequestHeaders,
		Status:         ProviderConfigStatusActive,
	}, nil
}

func (s *fakeStore) ListCreativeProviderConfigs(
	_ context.Context,
	_ string,
) ([]CreativeProviderConfig, error) {
	return nil, nil
}

func (s *fakeStore) GetCreativeProviderConfig(_ context.Context, id string) (CreativeProviderConfig, error) {
	return CreativeProviderConfig{
		ID:           id,
		CampaignID:   "campaign",
		ProviderName: "generic",
		FetchURL:     "https://provider.example/creatives",
		Status:       ProviderConfigStatusActive,
	}, nil
}

func (s *fakeStore) UpdateCreativeProviderConfigStatus(
	_ context.Context,
	input UpdateCreativeProviderConfigStatusInput,
) (CreativeProviderConfig, error) {
	return CreativeProviderConfig{
		ID:     input.ID,
		Status: input.Status,
	}, nil
}

func (s *fakeStore) CreateCreativeSyncLog(
	_ context.Context,
	providerConfigID string,
	campaignID string,
) (CreativeSyncLog, error) {
	return CreativeSyncLog{
		ID:               "sync-log",
		ProviderConfigID: providerConfigID,
		CampaignID:       campaignID,
		Status:           CreativeSyncLogStatusRunning,
	}, nil
}

func (s *fakeStore) CompleteCreativeSyncLog(
	_ context.Context,
	input CompleteCreativeSyncLogInput,
) (CreativeSyncLog, error) {
	return CreativeSyncLog{
		ID:            input.ID,
		Status:        input.Status,
		FetchedTotal:  input.FetchedTotal,
		UpsertedTotal: input.UpsertedTotal,
		ErrorMessage:  input.ErrorMessage,
	}, nil
}

func (s *fakeStore) ListCreativeSyncLogs(
	_ context.Context,
	_ string,
	_ string,
) ([]CreativeSyncLog, error) {
	return nil, nil
}

func (s *fakeStore) UpsertProviderCreative(
	_ context.Context,
	input UpsertProviderCreativeInput,
) (Creative, error) {
	s.providerCreativeInput = input
	return Creative{
		ID:                 "provider-creative",
		CampaignID:         input.CampaignID,
		Title:              input.Title,
		Body:               input.Body,
		URL:                input.URL,
		Status:             input.Status,
		SourceType:         CreativeSourceProviderAPI,
		ProviderConfigID:   input.ProviderConfigID,
		ProviderName:       input.ProviderName,
		ProviderExternalID: input.ProviderExternalID,
		SyncStatus:         CreativeSyncStatusSynced,
	}, nil
}

func (s *fakeStore) MarkMissingProviderCreativesStale(
	_ context.Context,
	_ string,
	_ []string,
) (int64, error) {
	return 0, nil
}

func (s *fakeStore) TryAcquireCreativeProviderSyncLock(
	_ context.Context,
	_ string,
) (bool, error) {
	return true, nil
}

func (s *fakeStore) ReleaseCreativeProviderSyncLock(
	_ context.Context,
	_ string,
) error {
	return nil
}

func (s *fakeStore) ActiveCreativeForCampaign(_ context.Context, campaignID string) (Creative, error) {
	return Creative{ID: "creative", CampaignID: campaignID, Status: CreativeStatusActive}, nil
}

func (s *fakeStore) ActiveCreativeForSource(_ context.Context, _ string) (Creative, error) {
	return Creative{ID: "creative", CampaignID: "campaign", Status: CreativeStatusActive}, nil
}

func (s *fakeStore) ActiveCreativesForCampaign(_ context.Context, campaignID string) ([]Creative, error) {
	return []Creative{{ID: "creative", CampaignID: campaignID, Status: CreativeStatusActive}}, nil
}

func (s *fakeStore) ActiveCreativesForSource(_ context.Context, _ string) ([]Creative, error) {
	return []Creative{{ID: "creative", CampaignID: "campaign", Status: CreativeStatusActive}}, nil
}

type fakeCounter struct{}

func (c fakeCounter) CountBySource(_ context.Context, _ string) (int64, error) {
	return 3, nil
}

func (c fakeCounter) CountBySourceToday(_ context.Context, _ string) (int64, error) {
	return 2, nil
}

func (c fakeCounter) CountEventsBySourceToday(_ context.Context, _ string) (int64, error) {
	return 4, nil
}

func (c fakeCounter) CountEventsBySourceTodayByType(_ context.Context, _ string) (map[string]int64, error) {
	return map[string]int64{
		"subscribed":         2,
		"notification_shown": 1,
		"notification_click": 1,
	}, nil
}

func (c fakeCounter) LastEventAtBySource(_ context.Context, _ string) (string, error) {
	return "2026-05-19 12:00:00.000", nil
}

func (c fakeCounter) EstimateAudience(
	_ context.Context,
	_ []string,
	_ TargetingRules,
) (int64, error) {
	return 7, nil
}

func (c fakeCounter) BuildAudience(_ context.Context, _ BuildAudienceInput) (int64, error) {
	return 7, nil
}

func (c fakeCounter) Timezones(_ context.Context) ([]string, error) {
	return []string{"UTC"}, nil
}

func (c fakeCounter) AudienceBatch(_ context.Context, input AudienceBatchInput) ([]AudienceRow, error) {
	if input.AfterSubscriptionID != "" {
		return nil, nil
	}
	return []AudienceRow{
		{
			LaunchID:       input.LaunchID,
			CampaignID:     "campaign",
			SourceID:       "source",
			SubscriptionID: "subscription-1",
			Endpoint:       "https://push.example/1",
			P256DH:         "p256dh",
			Auth:           "auth",
		},
		{
			LaunchID:       input.LaunchID,
			CampaignID:     "campaign",
			SourceID:       "source",
			SubscriptionID: "subscription-2",
			Endpoint:       "https://push.example/2",
			P256DH:         "p256dh",
			Auth:           "auth",
		},
	}, nil
}

type fakeDeliveryProducer struct {
	tasks []DeliveryTask
}

func (p *fakeDeliveryProducer) ProduceDeliveryTasks(_ context.Context, tasks []DeliveryTask) error {
	p.tasks = append(p.tasks, tasks...)
	return nil
}

type fakeProviderFetcher struct {
	creatives []ProviderCreative
}

func (f fakeProviderFetcher) FetchCreatives(
	_ context.Context,
	_ CreativeProviderConfig,
) ([]ProviderCreative, error) {
	return f.creatives, nil
}

func TestServiceCreatePublisherValidatesName(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.CreatePublisher(context.Background(), CreatePublisherInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceSyncCreativeProviderConfigUpsertsFetchedCreatives(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	service := NewService(store, fakeCounter{}, "")
	service.SetCreativeProviderFetcher(fakeProviderFetcher{
		creatives: []ProviderCreative{
			{
				ExternalID: "external-1",
				Title:      "Provider title",
				Body:       "Provider body",
				URL:        "https://example.com/landing",
				Icon:       "https://example.com/icon.png",
			},
		},
	})

	log, err := service.SyncCreativeProviderConfig(context.Background(), "provider-config")
	if err != nil {
		t.Fatalf("sync provider config: %v", err)
	}

	if log.Status != CreativeSyncLogStatusCompleted {
		t.Fatalf("expected completed sync, got %q", log.Status)
	}
	if log.FetchedTotal != 1 || log.UpsertedTotal != 1 {
		t.Fatalf("expected fetched/upserted 1/1, got %d/%d", log.FetchedTotal, log.UpsertedTotal)
	}
	if store.providerCreativeInput.ProviderExternalID != "external-1" {
		t.Fatalf("expected external id stored, got %q", store.providerCreativeInput.ProviderExternalID)
	}
	if store.providerCreativeInput.Status != CreativeStatusActive {
		t.Fatalf("expected default active status, got %q", store.providerCreativeInput.Status)
	}
}

func TestValidateFetchTargetRejectsPrivateAddressByDefault(t *testing.T) {
	t.Parallel()

	target, err := url.Parse("http://127.0.0.1:19080/creatives.json")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if err := validateFetchTarget(context.Background(), target, false); err == nil {
		t.Fatal("expected private fetch target to be rejected")
	}
}

func TestValidateFetchTargetAllowsPrivateAddressWhenConfigured(t *testing.T) {
	t.Parallel()

	target, err := url.Parse("http://127.0.0.1:19080/creatives.json")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if err := validateFetchTarget(context.Background(), target, true); err != nil {
		t.Fatalf("expected private fetch target to be allowed: %v", err)
	}
}

func TestServiceCreateSourceNormalizesInput(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	service := NewService(store, fakeCounter{}, "")
	source, err := service.CreateSource(context.Background(), CreateSourceInput{
		PublisherID: "publisher",
		Name:        " Source ",
		Domain:      "Example.COM ",
	})
	if err != nil {
		t.Fatalf("create source failed: %v", err)
	}
	if source.Domain != "example.com" {
		t.Fatalf("expected normalized domain, got %q", source.Domain)
	}
	if store.sourceInput.Name != "Source" {
		t.Fatalf("expected trimmed name, got %q", store.sourceInput.Name)
	}
}

func TestServiceCreateCampaignValidatesInput(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.CreateCampaign(context.Background(), CreateCampaignInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceCreateCampaignTrimsInput(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	service := NewService(store, fakeCounter{}, "")
	campaign, err := service.CreateCampaign(context.Background(), CreateCampaignInput{
		PublisherID: " publisher ",
		SourceID:    " source ",
		Name:        " Campaign ",
		TargetingRules: TargetingRules{
			Countries:   []string{" US ", "us", ""},
			Languages:   []string{" EN "},
			DeviceTypes: []string{" Mobile "},
		},
		DailyCapPerSubscription: 1,
		TotalCapPerSubscription: 3,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	if campaign.Status != CampaignStatusDraft {
		t.Fatalf("expected draft campaign, got %q", campaign.Status)
	}
	if store.campaignInput.Name != "Campaign" {
		t.Fatalf("expected trimmed campaign name, got %q", store.campaignInput.Name)
	}
	if campaign.DailyCapPerSubscription != 1 || campaign.TotalCapPerSubscription != 3 {
		t.Fatalf("unexpected campaign caps: %+v", campaign)
	}
	if len(campaign.TargetingRules.Countries) != 1 || campaign.TargetingRules.Countries[0] != "us" {
		t.Fatalf("expected normalized countries, got %+v", campaign.TargetingRules)
	}
	if campaign.TargetingRules.Languages[0] != "en" || campaign.TargetingRules.DeviceTypes[0] != "mobile" {
		t.Fatalf("expected normalized targeting rules, got %+v", campaign.TargetingRules)
	}
}

func TestServiceCreateCampaignRejectsNegativeCaps(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.CreateCampaign(context.Background(), CreateCampaignInput{
		SourceIDs:               []string{"source"},
		Name:                    "Campaign",
		DailyCapPerSubscription: -1,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceEstimateCampaignAudience(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	estimate, err := service.EstimateCampaignAudience(context.Background(), " campaign ")
	if err != nil {
		t.Fatalf("estimate campaign audience failed: %v", err)
	}
	if estimate.CampaignID != "campaign" || estimate.SourceID != "source" || estimate.Total != 7 {
		t.Fatalf("unexpected estimate: %+v", estimate)
	}
}

func TestServiceCreateCampaignLaunchBuildsAudience(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	launch, err := service.CreateCampaignLaunch(context.Background(), " campaign ")
	if err != nil {
		t.Fatalf("create campaign launch failed: %v", err)
	}
	if launch.Status != CampaignLaunchStatusCompleted {
		t.Fatalf("expected completed launch, got %+v", launch)
	}
	if launch.AudienceTotal != 7 || launch.ProcessedTotal != 7 {
		t.Fatalf("unexpected launch totals: %+v", launch)
	}
}

func TestServiceEnqueueCampaignLaunchPublishesAudience(t *testing.T) {
	t.Parallel()

	producer := &fakeDeliveryProducer{}
	service := NewService(&fakeStore{}, fakeCounter{}, "")
	service.SetDeliveryProducer(producer)
	launch, err := service.EnqueueCampaignLaunch(context.Background(), "campaign", "launch")
	if err != nil {
		t.Fatalf("enqueue campaign launch failed: %v", err)
	}
	if launch.EnqueueStatus != EnqueueStatusCompleted || launch.EnqueuedTotal != 2 {
		t.Fatalf("unexpected launch enqueue status: %+v", launch)
	}
	if len(producer.tasks) != 2 {
		t.Fatalf("expected two delivery tasks, got %+v", producer.tasks)
	}
	if producer.tasks[0].Attempt != 0 || producer.tasks[0].SubscriptionID != "subscription-1" {
		t.Fatalf("unexpected first delivery task: %+v", producer.tasks[0])
	}
}

func TestServiceCreateCreativeValidatesInput(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.CreateCreative(context.Background(), CreateCreativeInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceCreateCreativeTrimsInput(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	service := NewService(store, fakeCounter{}, "")
	creative, err := service.CreateCreative(context.Background(), CreateCreativeInput{
		CampaignID:              " campaign ",
		Title:                   " Title ",
		Body:                    " Body ",
		URL:                     " https://example.com ",
		Icon:                    " https://example.com/icon.png ",
		DailyCapPerSubscription: 2,
		TotalCapPerSubscription: 5,
	})
	if err != nil {
		t.Fatalf("create creative failed: %v", err)
	}
	if creative.Status != CreativeStatusActive {
		t.Fatalf("expected active creative, got %q", creative.Status)
	}
	if store.creativeInput.Title != "Title" || store.creativeInput.URL != "https://example.com" {
		t.Fatalf("expected trimmed creative input, got %+v", store.creativeInput)
	}
	if creative.DailyCapPerSubscription != 2 || creative.TotalCapPerSubscription != 5 {
		t.Fatalf("unexpected creative caps: %+v", creative)
	}
}

func TestServiceCreateCreativeRejectsNegativeCaps(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.CreateCreative(context.Background(), CreateCreativeInput{
		CampaignID:              "campaign",
		Title:                   "Title",
		Body:                    "Body",
		URL:                     "https://example.com",
		DailyCapPerSubscription: -1,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceSourceStats(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	stats, err := service.SourceStats(context.Background(), "source")
	if err != nil {
		t.Fatalf("source stats failed: %v", err)
	}
	if stats.Subscribers != 3 || stats.SubscribersToday != 2 || stats.EventsToday != 4 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.EventBreakdown["notification_shown"] != 1 {
		t.Fatalf("unexpected event breakdown: %+v", stats.EventBreakdown)
	}
	if stats.Health.Status != "attention" {
		t.Fatalf("expected attention health for source without vapid key/domain, got %+v", stats.Health)
	}
}

func TestServiceCreateVAPIDKeyGeneratesPair(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	service := NewService(store, fakeCounter{}, "")
	key, err := service.CreateVAPIDKey(context.Background(), CreateVAPIDKeyInput{Name: " Main key "})
	if err != nil {
		t.Fatalf("create vapid key failed: %v", err)
	}
	if key.Name != "Main key" {
		t.Fatalf("expected trimmed name, got %q", key.Name)
	}
	if key.PublicKey == "" || key.PrivateKey == "" {
		t.Fatalf("expected generated key pair: %+v", key)
	}
	if key.Status != VAPIDStatusActive {
		t.Fatalf("expected active status, got %q", key.Status)
	}
	if store.vapidKey.PrivateKey == "" {
		t.Fatal("expected private key to be passed to store")
	}
}

func TestServiceUpdateVAPIDKeyStatusValidates(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.UpdateVAPIDKeyStatus(context.Background(), "key", "deleted")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceAttachVAPIDKeyToSourceValidates(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "")
	_, err := service.AttachVAPIDKeyToSource(context.Background(), AttachVAPIDKeyInput{
		SourceID: "source",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceSourceSnippetUsesPushManager(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeStore{}, fakeCounter{}, "https://api.example")
	snippet, err := service.SourceSnippet(context.Background(), "source")
	if err != nil {
		t.Fatalf("source snippet failed: %v", err)
	}
	for _, part := range []string{"<script async", "https://api.example/api/sdk/sources/source.js"} {
		if !strings.Contains(snippet, part) {
			t.Fatalf("expected snippet to contain %q", part)
		}
	}
	if strings.Contains(snippet, "PushManager") {
		t.Fatalf("expected install snippet to be a loader tag only, got %q", snippet)
	}
}
