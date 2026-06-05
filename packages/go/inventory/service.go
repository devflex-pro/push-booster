package inventory

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const StatusActive = "active"
const (
	CampaignStatusDraft    = "draft"
	CampaignStatusActive   = "active"
	CampaignStatusPaused   = "paused"
	CampaignStatusArchived = "archived"
)
const (
	CampaignAudienceScopeAll             = "all"
	CampaignAudienceScopeSelectedSources = "selected_sources"
)
const (
	CreativeStatusActive   = "active"
	CreativeStatusPaused   = "paused"
	CreativeStatusArchived = "archived"
)
const (
	CreativeSourceManual      = "manual"
	CreativeSourceProviderAPI = "provider_api"
)
const (
	CreativeSyncStatusSynced  = "synced"
	CreativeSyncStatusInvalid = "invalid"
	CreativeSyncStatusStale   = "stale"
)
const (
	ProviderConfigStatusActive   = "active"
	ProviderConfigStatusPaused   = "paused"
	ProviderConfigStatusArchived = "archived"
)
const (
	CreativeSyncLogStatusRunning   = "running"
	CreativeSyncLogStatusCompleted = "completed"
	CreativeSyncLogStatusFailed    = "failed"
)
const (
	CampaignLaunchStatusBuilding  = "building"
	CampaignLaunchStatusCompleted = "completed"
	CampaignLaunchStatusFailed    = "failed"
)
const (
	EnqueueStatusPending   = "pending"
	EnqueueStatusEnqueuing = "enqueuing"
	EnqueueStatusCompleted = "completed"
	EnqueueStatusFailed    = "failed"
)
const (
	ScheduleStatusActive   = "active"
	ScheduleStatusPaused   = "paused"
	ScheduleStatusArchived = "archived"
)
const (
	ScheduleRunStatusPending   = "pending"
	ScheduleRunStatusRunning   = "running"
	ScheduleRunStatusCompleted = "completed"
	ScheduleRunStatusFailed    = "failed"
)
const (
	VAPIDStatusActive     = "active"
	VAPIDStatusDeprecated = "deprecated"
	VAPIDStatusRevoked    = "revoked"
)

var (
	ErrInvalidInput = errors.New("invalid inventory input")
	ErrNotFound     = errors.New("inventory record not found")
)

type Publisher struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Source struct {
	ID          string    `json:"id"`
	PublisherID string    `json:"publisher_id"`
	Name        string    `json:"name"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	VAPIDKeyID  string    `json:"vapid_key_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type VAPIDKey struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"private_key,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Campaign struct {
	ID                      string         `json:"id"`
	PublisherID             string         `json:"publisher_id,omitempty"`
	SourceID                string         `json:"source_id,omitempty"`
	SourceIDs               []string       `json:"source_ids"`
	AudienceScope           string         `json:"audience_scope"`
	Name                    string         `json:"name"`
	Status                  string         `json:"status"`
	TargetingRules          TargetingRules `json:"targeting_rules"`
	DailyCapPerSubscription int            `json:"daily_cap_per_subscription"`
	TotalCapPerSubscription int            `json:"total_cap_per_subscription"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
}

type TargetingRules struct {
	Countries    []string `json:"countries,omitempty"`
	Languages    []string `json:"languages,omitempty"`
	DeviceTypes  []string `json:"device_types,omitempty"`
	OSNames      []string `json:"os_names,omitempty"`
	BrowserNames []string `json:"browser_names,omitempty"`
}

type Creative struct {
	ID                              string         `json:"id"`
	CampaignID                      string         `json:"campaign_id"`
	Title                           string         `json:"title"`
	Body                            string         `json:"body"`
	URL                             string         `json:"url"`
	Icon                            string         `json:"icon,omitempty"`
	Status                          string         `json:"status"`
	SourceType                      string         `json:"source_type"`
	ProviderConfigID                string         `json:"provider_config_id,omitempty"`
	ProviderName                    string         `json:"provider_name,omitempty"`
	ProviderExternalID              string         `json:"provider_external_id,omitempty"`
	LastSyncedAt                    *time.Time     `json:"last_synced_at,omitempty"`
	SyncStatus                      string         `json:"sync_status,omitempty"`
	DailyCapPerSubscription         int            `json:"daily_cap_per_subscription"`
	TotalCapPerSubscription         int            `json:"total_cap_per_subscription"`
	CampaignDailyCapPerSubscription int            `json:"campaign_daily_cap_per_subscription,omitempty"`
	CampaignTotalCapPerSubscription int            `json:"campaign_total_cap_per_subscription,omitempty"`
	CampaignTargetingRules          TargetingRules `json:"campaign_targeting_rules,omitempty"`
	CreatedAt                       time.Time      `json:"created_at"`
	UpdatedAt                       time.Time      `json:"updated_at"`
}

type CreativeProviderConfig struct {
	ID             string            `json:"id"`
	CampaignID     string            `json:"campaign_id"`
	Name           string            `json:"name"`
	ProviderName   string            `json:"provider_name"`
	FetchURL       string            `json:"fetch_url"`
	RequestHeaders map[string]string `json:"request_headers"`
	Status         string            `json:"status"`
	LastSyncAt     *time.Time        `json:"last_sync_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type CreativeSyncLog struct {
	ID               string     `json:"id"`
	ProviderConfigID string     `json:"provider_config_id"`
	CampaignID       string     `json:"campaign_id"`
	Status           string     `json:"status"`
	FetchedTotal     int        `json:"fetched_total"`
	UpsertedTotal    int        `json:"upserted_total"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type CampaignLaunch struct {
	ID                 string     `json:"id"`
	CampaignID         string     `json:"campaign_id"`
	Status             string     `json:"status"`
	AudienceTotal      int64      `json:"audience_total"`
	ProcessedTotal     int64      `json:"processed_total"`
	ErrorMessage       string     `json:"error_message,omitempty"`
	EnqueueStatus      string     `json:"enqueue_status"`
	EnqueuedTotal      int64      `json:"enqueued_total"`
	EnqueueError       string     `json:"enqueue_error,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	EnqueueStartedAt   *time.Time `json:"enqueue_started_at,omitempty"`
	EnqueueCompletedAt *time.Time `json:"enqueue_completed_at,omitempty"`
}

type CampaignSchedule struct {
	ID               string                 `json:"id"`
	CampaignID       string                 `json:"campaign_id"`
	Status           string                 `json:"status"`
	TimezoneMode     string                 `json:"timezone_mode"`
	FallbackTimezone string                 `json:"fallback_timezone"`
	GraceMinutes     int                    `json:"grace_minutes"`
	Slots            []CampaignScheduleSlot `json:"slots"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type CampaignScheduleSlot struct {
	ID         string    `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	LocalTime  string    `json:"local_time"`
	DaysOfWeek []int     `json:"days_of_week"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

type CampaignScheduleRun struct {
	ID             string     `json:"id"`
	ScheduleID     string     `json:"schedule_id"`
	SlotID         string     `json:"slot_id"`
	CampaignID     string     `json:"campaign_id"`
	LaunchID       string     `json:"launch_id,omitempty"`
	LocalDate      string     `json:"local_date"`
	LocalTime      string     `json:"local_time"`
	Timezone       string     `json:"timezone"`
	ScheduledUTCAt time.Time  `json:"scheduled_utc_at"`
	Status         string     `json:"status"`
	AudienceTotal  int64      `json:"audience_total"`
	EnqueuedTotal  int64      `json:"enqueued_total"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type ScheduleTickResult struct {
	CreatedRuns int `json:"created_runs"`
	Completed   int `json:"completed"`
	Failed      int `json:"failed"`
}

type AudienceEstimate struct {
	CampaignID    string   `json:"campaign_id"`
	SourceID      string   `json:"source_id,omitempty"`
	SourceIDs     []string `json:"source_ids"`
	AudienceScope string   `json:"audience_scope"`
	Total         int64    `json:"total"`
}

type SourceStats struct {
	SourceID         string           `json:"source_id"`
	Subscribers      int64            `json:"subscribers"`
	SubscribersToday int64            `json:"subscribers_today"`
	EventsToday      int64            `json:"events_today"`
	EventBreakdown   map[string]int64 `json:"event_breakdown"`
	LastEventAt      string           `json:"last_event_at,omitempty"`
	Health           SourceHealth     `json:"health"`
}

type SourceHealth struct {
	Status string   `json:"status"`
	Issues []string `json:"issues"`
}

type CreatePublisherInput struct {
	Name string
}

type CreateSourceInput struct {
	PublisherID string
	Name        string
	Domain      string
}

type CreateVAPIDKeyInput struct {
	Name string
}

type AttachVAPIDKeyInput struct {
	SourceID   string
	VAPIDKeyID string
}

type CreateCampaignInput struct {
	PublisherID             string
	SourceID                string
	SourceIDs               []string
	AudienceScope           string
	Name                    string
	TargetingRules          TargetingRules
	DailyCapPerSubscription int
	TotalCapPerSubscription int
}

type UpdateCampaignStatusInput struct {
	ID     string
	Status string
}

type CreateCreativeInput struct {
	CampaignID              string
	Title                   string
	Body                    string
	URL                     string
	Icon                    string
	DailyCapPerSubscription int
	TotalCapPerSubscription int
}

type UpdateCreativeStatusInput struct {
	ID     string
	Status string
}

type UpdateCreativeProviderConfigStatusInput struct {
	ID     string
	Status string
}

type CreateCreativeProviderConfigInput struct {
	CampaignID     string
	Name           string
	ProviderName   string
	FetchURL       string
	RequestHeaders map[string]string
}

type UpsertProviderCreativeInput struct {
	CampaignID              string
	ProviderConfigID        string
	ProviderName            string
	ProviderExternalID      string
	Title                   string
	Body                    string
	URL                     string
	Icon                    string
	Status                  string
	DailyCapPerSubscription int
	TotalCapPerSubscription int
	RawProviderPayload      string
}

type CompleteCreativeSyncLogInput struct {
	ID            string
	Status        string
	FetchedTotal  int
	UpsertedTotal int
	ErrorMessage  string
}

type CreateCampaignLaunchInput struct {
	CampaignID string
}

type CreateCampaignScheduleInput struct {
	CampaignID       string
	Status           string
	FallbackTimezone string
	GraceMinutes     int
	Slots            []CreateCampaignScheduleSlotInput
}

type CreateCampaignScheduleSlotInput struct {
	LocalTime  string
	DaysOfWeek []int
	Position   int
}

type UpdateCampaignScheduleStatusInput struct {
	ID     string
	Status string
}

type CreateScheduleRunInput struct {
	ScheduleID     string
	SlotID         string
	CampaignID     string
	LocalDate      string
	LocalTime      string
	Timezone       string
	ScheduledUTCAt time.Time
}

type CompleteScheduleRunInput struct {
	ID            string
	LaunchID      string
	Status        string
	AudienceTotal int64
	EnqueuedTotal int64
	ErrorMessage  string
}

type UpdateCampaignLaunchInput struct {
	ID             string
	Status         string
	AudienceTotal  int64
	ProcessedTotal int64
	ErrorMessage   string
}

type BuildAudienceInput struct {
	LaunchID       string
	CampaignID     string
	SourceIDs      []string
	TargetingRules TargetingRules
	Timezone       string
}

type AudienceBatchInput struct {
	LaunchID            string
	AfterShard          uint16
	AfterSubscriptionID string
	Limit               int
}

type AudienceRow struct {
	LaunchID       string `json:"launch_id"`
	CampaignID     string `json:"campaign_id"`
	SourceID       string `json:"source_id"`
	SubscriptionID string `json:"subscription_id"`
	Endpoint       string `json:"endpoint"`
	P256DH         string `json:"p256dh"`
	Auth           string `json:"auth"`
	Shard          uint16 `json:"shard"`
}

type DeliveryTask struct {
	DeliveryID     string    `json:"delivery_id"`
	LaunchID       string    `json:"launch_id"`
	CampaignID     string    `json:"campaign_id"`
	SourceID       string    `json:"source_id"`
	SubscriptionID string    `json:"subscription_id"`
	Endpoint       string    `json:"endpoint"`
	P256DH         string    `json:"p256dh"`
	Auth           string    `json:"auth"`
	Shard          uint16    `json:"shard"`
	Attempt        int       `json:"attempt"`
	CreatedAt      time.Time `json:"created_at"`
}

type UpdateCampaignLaunchEnqueueInput struct {
	ID            string
	EnqueueStatus string
	EnqueuedTotal int64
	EnqueueError  string
}

type Store interface {
	CreatePublisher(ctx context.Context, input CreatePublisherInput) (Publisher, error)
	ListPublishers(ctx context.Context) ([]Publisher, error)
	CreateSource(ctx context.Context, input CreateSourceInput) (Source, error)
	ListSources(ctx context.Context, publisherID string) ([]Source, error)
	GetSource(ctx context.Context, id string) (Source, error)
	CreateVAPIDKey(ctx context.Context, key VAPIDKey) (VAPIDKey, error)
	ListVAPIDKeys(ctx context.Context) ([]VAPIDKey, error)
	UpdateVAPIDKeyStatus(ctx context.Context, id string, status string) (VAPIDKey, error)
	AttachVAPIDKeyToSource(ctx context.Context, input AttachVAPIDKeyInput) (Source, error)
	CreateCampaign(ctx context.Context, input CreateCampaignInput) (Campaign, error)
	GetCampaign(ctx context.Context, id string) (Campaign, error)
	ListCampaigns(ctx context.Context, sourceID string) ([]Campaign, error)
	UpdateCampaignStatus(ctx context.Context, input UpdateCampaignStatusInput) (Campaign, error)
	CreateCampaignLaunch(ctx context.Context, input CreateCampaignLaunchInput) (CampaignLaunch, error)
	GetCampaignLaunch(ctx context.Context, id string) (CampaignLaunch, error)
	ListCampaignLaunches(ctx context.Context, campaignID string) ([]CampaignLaunch, error)
	UpdateCampaignLaunch(ctx context.Context, input UpdateCampaignLaunchInput) (CampaignLaunch, error)
	UpdateCampaignLaunchEnqueue(ctx context.Context, input UpdateCampaignLaunchEnqueueInput) (CampaignLaunch, error)
	ListCampaignSchedules(ctx context.Context, campaignID string) ([]CampaignSchedule, error)
	ListActiveCampaignSchedules(ctx context.Context) ([]CampaignSchedule, error)
	CreateCampaignSchedule(ctx context.Context, input CreateCampaignScheduleInput) (CampaignSchedule, error)
	UpdateCampaignScheduleStatus(ctx context.Context, input UpdateCampaignScheduleStatusInput) (CampaignSchedule, error)
	CreateScheduleRun(ctx context.Context, input CreateScheduleRunInput) (CampaignScheduleRun, bool, error)
	CompleteScheduleRun(ctx context.Context, input CompleteScheduleRunInput) (CampaignScheduleRun, error)
	ListCampaignScheduleRuns(ctx context.Context, campaignID string) ([]CampaignScheduleRun, error)
	CreateCreative(ctx context.Context, input CreateCreativeInput) (Creative, error)
	ListCreatives(ctx context.Context, campaignID string) ([]Creative, error)
	UpdateCreativeStatus(ctx context.Context, input UpdateCreativeStatusInput) (Creative, error)
	CreateCreativeProviderConfig(ctx context.Context, input CreateCreativeProviderConfigInput) (CreativeProviderConfig, error)
	ListCreativeProviderConfigs(ctx context.Context, campaignID string) ([]CreativeProviderConfig, error)
	GetCreativeProviderConfig(ctx context.Context, id string) (CreativeProviderConfig, error)
	CreateCreativeSyncLog(ctx context.Context, providerConfigID string, campaignID string) (CreativeSyncLog, error)
	CompleteCreativeSyncLog(ctx context.Context, input CompleteCreativeSyncLogInput) (CreativeSyncLog, error)
	UpsertProviderCreative(ctx context.Context, input UpsertProviderCreativeInput) (Creative, error)
	MarkMissingProviderCreativesStale(ctx context.Context, providerConfigID string, externalIDs []string) (int64, error)
	UpdateCreativeProviderConfigStatus(ctx context.Context, input UpdateCreativeProviderConfigStatusInput) (CreativeProviderConfig, error)
	ListCreativeSyncLogs(ctx context.Context, providerConfigID string, campaignID string) ([]CreativeSyncLog, error)
	TryAcquireCreativeProviderSyncLock(ctx context.Context, providerConfigID string) (bool, error)
	ReleaseCreativeProviderSyncLock(ctx context.Context, providerConfigID string) error
	ActiveCreativeForCampaign(ctx context.Context, campaignID string) (Creative, error)
	ActiveCreativeForSource(ctx context.Context, sourceID string) (Creative, error)
	ActiveCreativesForCampaign(ctx context.Context, campaignID string) ([]Creative, error)
	ActiveCreativesForSource(ctx context.Context, sourceID string) ([]Creative, error)
}

type SubscriberCounter interface {
	CountBySource(ctx context.Context, sourceID string) (int64, error)
	CountBySourceToday(ctx context.Context, sourceID string) (int64, error)
	CountEventsBySourceToday(ctx context.Context, sourceID string) (int64, error)
	CountEventsBySourceTodayByType(ctx context.Context, sourceID string) (map[string]int64, error)
	LastEventAtBySource(ctx context.Context, sourceID string) (string, error)
}

type AudienceBuilder interface {
	EstimateAudience(
		ctx context.Context,
		sourceIDs []string,
		rules TargetingRules,
	) (int64, error)
	BuildAudience(ctx context.Context, input BuildAudienceInput) (int64, error)
	AudienceBatch(ctx context.Context, input AudienceBatchInput) ([]AudienceRow, error)
	Timezones(ctx context.Context) ([]string, error)
}

type DeliveryProducer interface {
	ProduceDeliveryTasks(ctx context.Context, tasks []DeliveryTask) error
}

type CreativeProviderFetcher interface {
	FetchCreatives(ctx context.Context, config CreativeProviderConfig) ([]ProviderCreative, error)
}

type ProviderCreative struct {
	ExternalID              string
	Title                   string
	Body                    string
	URL                     string
	Icon                    string
	Status                  string
	DailyCapPerSubscription int
	TotalCapPerSubscription int
	RawProviderPayload      string
}

type ProviderFetchConfig struct {
	MaxBodyBytes          int64
	MaxAttempts           int
	RetryBackoff          time.Duration
	AllowPrivateFetchURLs bool
}

type Service struct {
	store           Store
	subscribers     SubscriberCounter
	audience        AudienceBuilder
	delivery        DeliveryProducer
	providerFetcher CreativeProviderFetcher
	publicURL       string
	batchSize       int
}

func NewService(
	store Store,
	subscribers SubscriberCounter,
	publicURL string,
) *Service {
	return &Service{
		store:       store,
		subscribers: subscribers,
		audience:    audienceBuilder(subscribers),
		providerFetcher: HTTPProviderFetcher{
			Client: &http.Client{Timeout: 15 * time.Second},
			Config: ProviderFetchConfig{
				MaxBodyBytes: 2 * 1024 * 1024,
				MaxAttempts:  2,
				RetryBackoff: 500 * time.Millisecond,
			},
		},
		publicURL: strings.TrimRight(publicURL, "/"),
		batchSize: 1000,
	}
}

func (s *Service) SetDeliveryProducer(producer DeliveryProducer) {
	s.delivery = producer
}

func (s *Service) SetCreativeProviderFetcher(fetcher CreativeProviderFetcher) {
	s.providerFetcher = fetcher
}

func audienceBuilder(subscribers SubscriberCounter) AudienceBuilder {
	builder, ok := subscribers.(AudienceBuilder)
	if !ok {
		return nil
	}
	return builder
}

func (s *Service) CreatePublisher(ctx context.Context, input CreatePublisherInput) (Publisher, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Publisher{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	return s.store.CreatePublisher(ctx, input)
}

func (s *Service) ListPublishers(ctx context.Context) ([]Publisher, error) {
	return s.store.ListPublishers(ctx)
}

func (s *Service) CreateSource(ctx context.Context, input CreateSourceInput) (Source, error) {
	input.PublisherID = strings.TrimSpace(input.PublisherID)
	input.Name = strings.TrimSpace(input.Name)
	input.Domain = strings.ToLower(strings.TrimSpace(input.Domain))
	if input.PublisherID == "" {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("publisher_id is required"))
	}
	if input.Name == "" {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	if input.Domain == "" {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("domain is required"))
	}
	return s.store.CreateSource(ctx, input)
}

func (s *Service) ListSources(ctx context.Context, publisherID string) ([]Source, error) {
	return s.store.ListSources(ctx, strings.TrimSpace(publisherID))
}

func (s *Service) GetSource(ctx context.Context, id string) (Source, error) {
	return s.store.GetSource(ctx, strings.TrimSpace(id))
}

func (s *Service) CreateVAPIDKey(ctx context.Context, input CreateVAPIDKeyInput) (VAPIDKey, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return VAPIDKey{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	publicKey, privateKey, err := generateVAPIDKeyPair()
	if err != nil {
		return VAPIDKey{}, err
	}
	return s.store.CreateVAPIDKey(ctx, VAPIDKey{
		Name:       input.Name,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Status:     VAPIDStatusActive,
	})
}

func (s *Service) ListVAPIDKeys(ctx context.Context) ([]VAPIDKey, error) {
	return s.store.ListVAPIDKeys(ctx)
}

func (s *Service) UpdateVAPIDKeyStatus(ctx context.Context, id string, status string) (VAPIDKey, error) {
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(status)
	if id == "" {
		return VAPIDKey{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	switch status {
	case VAPIDStatusActive, VAPIDStatusDeprecated, VAPIDStatusRevoked:
		return s.store.UpdateVAPIDKeyStatus(ctx, id, status)
	default:
		return VAPIDKey{}, errors.Join(ErrInvalidInput, errors.New("invalid vapid key status"))
	}
}

func (s *Service) AttachVAPIDKeyToSource(ctx context.Context, input AttachVAPIDKeyInput) (Source, error) {
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.VAPIDKeyID = strings.TrimSpace(input.VAPIDKeyID)
	if input.SourceID == "" {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("source_id is required"))
	}
	if input.VAPIDKeyID == "" {
		return Source{}, errors.Join(ErrInvalidInput, errors.New("vapid_key_id is required"))
	}
	return s.store.AttachVAPIDKeyToSource(ctx, input)
}

func (s *Service) CreateCampaign(ctx context.Context, input CreateCampaignInput) (Campaign, error) {
	input.PublisherID = strings.TrimSpace(input.PublisherID)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.SourceIDs = normalizeRuleValues(append(input.SourceIDs, input.SourceID))
	input.AudienceScope = strings.TrimSpace(input.AudienceScope)
	if input.AudienceScope == "" {
		input.AudienceScope = CampaignAudienceScopeSelectedSources
	}
	input.Name = strings.TrimSpace(input.Name)
	input.TargetingRules = normalizeTargetingRules(input.TargetingRules)
	switch input.AudienceScope {
	case CampaignAudienceScopeAll:
		input.PublisherID = ""
		input.SourceID = ""
		input.SourceIDs = nil
	case CampaignAudienceScopeSelectedSources:
		if len(input.SourceIDs) == 0 {
			return Campaign{}, errors.Join(ErrInvalidInput, errors.New("source_ids is required"))
		}
		input.SourceID = input.SourceIDs[0]
	default:
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("invalid audience_scope"))
	}
	if input.Name == "" {
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	if input.DailyCapPerSubscription < 0 {
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("daily_cap_per_subscription must be greater than or equal to zero"))
	}
	if input.TotalCapPerSubscription < 0 {
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("total_cap_per_subscription must be greater than or equal to zero"))
	}
	return s.store.CreateCampaign(ctx, input)
}

func normalizeTargetingRules(rules TargetingRules) TargetingRules {
	return TargetingRules{
		Countries:    normalizeRuleValues(rules.Countries),
		Languages:    normalizeRuleValues(rules.Languages),
		DeviceTypes:  normalizeRuleValues(rules.DeviceTypes),
		OSNames:      normalizeRuleValues(rules.OSNames),
		BrowserNames: normalizeRuleValues(rules.BrowserNames),
	}
}

func normalizeRuleValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := map[string]bool{}
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

func (s *Service) ListCampaigns(ctx context.Context, sourceID string) ([]Campaign, error) {
	return s.store.ListCampaigns(ctx, strings.TrimSpace(sourceID))
}

func (s *Service) GetCampaign(ctx context.Context, id string) (Campaign, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	return s.store.GetCampaign(ctx, id)
}

func (s *Service) UpdateCampaignStatus(ctx context.Context, input UpdateCampaignStatusInput) (Campaign, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Status = strings.TrimSpace(input.Status)
	if input.ID == "" {
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	switch input.Status {
	case CampaignStatusDraft, CampaignStatusActive, CampaignStatusPaused, CampaignStatusArchived:
		return s.store.UpdateCampaignStatus(ctx, input)
	default:
		return Campaign{}, errors.Join(ErrInvalidInput, errors.New("invalid campaign status"))
	}
}

func (s *Service) EstimateCampaignAudience(ctx context.Context, campaignID string) (AudienceEstimate, error) {
	campaign, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return AudienceEstimate{}, err
	}
	if s.audience == nil {
		return AudienceEstimate{}, errors.Join(ErrInvalidInput, errors.New("audience builder is not configured"))
	}
	total, err := s.audience.EstimateAudience(
		ctx,
		campaignAudienceSourceIDs(campaign),
		campaign.TargetingRules,
	)
	if err != nil {
		return AudienceEstimate{}, err
	}
	return AudienceEstimate{
		CampaignID:    campaign.ID,
		SourceID:      campaign.SourceID,
		SourceIDs:     campaign.SourceIDs,
		AudienceScope: campaign.AudienceScope,
		Total:         total,
	}, nil
}

func campaignAudienceSourceIDs(campaign Campaign) []string {
	if campaign.AudienceScope == CampaignAudienceScopeAll {
		return nil
	}
	return campaign.SourceIDs
}

func normalizeCampaignScheduleInput(input CreateCampaignScheduleInput) (CreateCampaignScheduleInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = ScheduleStatusActive
	}
	input.FallbackTimezone = strings.TrimSpace(input.FallbackTimezone)
	if input.FallbackTimezone == "" {
		input.FallbackTimezone = "UTC"
	}
	if _, err := time.LoadLocation(input.FallbackTimezone); err != nil {
		return CreateCampaignScheduleInput{}, errors.Join(ErrInvalidInput, errors.New("invalid fallback_timezone"))
	}
	if input.GraceMinutes == 0 {
		input.GraceMinutes = 10
	}
	if input.CampaignID == "" {
		return CreateCampaignScheduleInput{}, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	switch input.Status {
	case ScheduleStatusActive, ScheduleStatusPaused, ScheduleStatusArchived:
	default:
		return CreateCampaignScheduleInput{}, errors.Join(ErrInvalidInput, errors.New("invalid schedule status"))
	}
	if input.GraceMinutes < 1 || input.GraceMinutes > 120 {
		return CreateCampaignScheduleInput{}, errors.Join(ErrInvalidInput, errors.New("grace_minutes must be between 1 and 120"))
	}
	if len(input.Slots) == 0 {
		return CreateCampaignScheduleInput{}, errors.Join(ErrInvalidInput, errors.New("slots are required"))
	}
	for index := range input.Slots {
		slot, err := normalizeCampaignScheduleSlot(input.Slots[index], index)
		if err != nil {
			return CreateCampaignScheduleInput{}, err
		}
		input.Slots[index] = slot
	}
	return input, nil
}

func normalizeCampaignScheduleSlot(
	input CreateCampaignScheduleSlotInput,
	index int,
) (CreateCampaignScheduleSlotInput, error) {
	input.LocalTime = strings.TrimSpace(input.LocalTime)
	if _, err := time.Parse("15:04", input.LocalTime); err != nil {
		return CreateCampaignScheduleSlotInput{}, errors.Join(ErrInvalidInput, errors.New("invalid slot local_time"))
	}
	if len(input.DaysOfWeek) == 0 {
		input.DaysOfWeek = []int{1, 2, 3, 4, 5, 6, 7}
	}
	seen := map[int]bool{}
	days := make([]int, 0, len(input.DaysOfWeek))
	for _, day := range input.DaysOfWeek {
		if day < 1 || day > 7 {
			return CreateCampaignScheduleSlotInput{}, errors.Join(ErrInvalidInput, errors.New("invalid slot day_of_week"))
		}
		if seen[day] {
			continue
		}
		seen[day] = true
		days = append(days, day)
	}
	input.DaysOfWeek = days
	if input.Position == 0 {
		input.Position = index + 1
	}
	return input, nil
}

func (s *Service) CreateCampaignLaunch(ctx context.Context, campaignID string) (CampaignLaunch, error) {
	return s.createCampaignLaunch(ctx, campaignID, "")
}

func (s *Service) CreateScheduledCampaignLaunch(
	ctx context.Context,
	campaignID string,
	timezone string,
) (CampaignLaunch, error) {
	return s.createCampaignLaunch(ctx, campaignID, timezone)
}

func (s *Service) createCampaignLaunch(
	ctx context.Context,
	campaignID string,
	timezone string,
) (CampaignLaunch, error) {
	campaign, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return CampaignLaunch{}, err
	}
	if campaign.Status != CampaignStatusActive {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("campaign must be active"))
	}
	if s.audience == nil {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("audience builder is not configured"))
	}
	launch, err := s.store.CreateCampaignLaunch(ctx, CreateCampaignLaunchInput{CampaignID: campaign.ID})
	if err != nil {
		return CampaignLaunch{}, err
	}
	total, err := s.audience.BuildAudience(ctx, BuildAudienceInput{
		LaunchID:       launch.ID,
		CampaignID:     campaign.ID,
		SourceIDs:      campaignAudienceSourceIDs(campaign),
		TargetingRules: campaign.TargetingRules,
		Timezone:       strings.TrimSpace(timezone),
	})
	if err != nil {
		failed, updateErr := s.store.UpdateCampaignLaunch(ctx, UpdateCampaignLaunchInput{
			ID:             launch.ID,
			Status:         CampaignLaunchStatusFailed,
			AudienceTotal:  0,
			ProcessedTotal: 0,
			ErrorMessage:   err.Error(),
		})
		if updateErr != nil {
			return CampaignLaunch{}, errors.Join(err, updateErr)
		}
		return failed, err
	}
	return s.store.UpdateCampaignLaunch(ctx, UpdateCampaignLaunchInput{
		ID:             launch.ID,
		Status:         CampaignLaunchStatusCompleted,
		AudienceTotal:  total,
		ProcessedTotal: total,
	})
}

func (s *Service) ListCampaignSchedules(ctx context.Context, campaignID string) ([]CampaignSchedule, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	return s.store.ListCampaignSchedules(ctx, campaignID)
}

func (s *Service) CreateCampaignSchedule(
	ctx context.Context,
	input CreateCampaignScheduleInput,
) (CampaignSchedule, error) {
	normalized, err := normalizeCampaignScheduleInput(input)
	if err != nil {
		return CampaignSchedule{}, err
	}
	if _, err := s.GetCampaign(ctx, normalized.CampaignID); err != nil {
		return CampaignSchedule{}, err
	}
	return s.store.CreateCampaignSchedule(ctx, normalized)
}

func (s *Service) UpdateCampaignScheduleStatus(
	ctx context.Context,
	input UpdateCampaignScheduleStatusInput,
) (CampaignSchedule, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Status = strings.TrimSpace(input.Status)
	if input.ID == "" {
		return CampaignSchedule{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	switch input.Status {
	case ScheduleStatusActive, ScheduleStatusPaused, ScheduleStatusArchived:
		return s.store.UpdateCampaignScheduleStatus(ctx, input)
	default:
		return CampaignSchedule{}, errors.Join(ErrInvalidInput, errors.New("invalid schedule status"))
	}
}

func (s *Service) ListCampaignScheduleRuns(ctx context.Context, campaignID string) ([]CampaignScheduleRun, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	return s.store.ListCampaignScheduleRuns(ctx, campaignID)
}

func (s *Service) ProcessDueSchedules(ctx context.Context, now time.Time) (ScheduleTickResult, error) {
	if s.audience == nil {
		return ScheduleTickResult{}, errors.Join(ErrInvalidInput, errors.New("audience builder is not configured"))
	}
	if s.delivery == nil {
		return ScheduleTickResult{}, errors.Join(ErrInvalidInput, errors.New("delivery producer is not configured"))
	}
	schedules, err := s.store.ListActiveCampaignSchedules(ctx)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	timezones, err := s.audience.Timezones(ctx)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	result := ScheduleTickResult{}
	for _, schedule := range schedules {
		scheduleTimezones := timezones
		if len(scheduleTimezones) == 0 {
			scheduleTimezones = []string{schedule.FallbackTimezone}
		}
		for _, slot := range schedule.Slots {
			for _, timezone := range scheduleTimezones {
				due, localDate, scheduledAt := scheduleSlotDue(now, timezone, slot, schedule.GraceMinutes)
				if !due {
					continue
				}
				run, created, err := s.store.CreateScheduleRun(ctx, CreateScheduleRunInput{
					ScheduleID:     schedule.ID,
					SlotID:         slot.ID,
					CampaignID:     schedule.CampaignID,
					LocalDate:      localDate,
					LocalTime:      slot.LocalTime,
					Timezone:       timezone,
					ScheduledUTCAt: scheduledAt,
				})
				if err != nil {
					return result, err
				}
				if !created {
					continue
				}
				result.CreatedRuns++
				if err := s.processScheduleRun(ctx, run); err != nil {
					result.Failed++
					continue
				}
				result.Completed++
			}
		}
	}
	return result, nil
}

func (s *Service) processScheduleRun(ctx context.Context, run CampaignScheduleRun) error {
	launch, err := s.CreateScheduledCampaignLaunch(ctx, run.CampaignID, run.Timezone)
	if err != nil {
		_, updateErr := s.store.CompleteScheduleRun(ctx, CompleteScheduleRunInput{
			ID:           run.ID,
			Status:       ScheduleRunStatusFailed,
			ErrorMessage: err.Error(),
		})
		if updateErr != nil {
			return errors.Join(err, updateErr)
		}
		return err
	}
	launch, err = s.EnqueueCampaignLaunch(ctx, run.CampaignID, launch.ID)
	if err != nil {
		_, updateErr := s.store.CompleteScheduleRun(ctx, CompleteScheduleRunInput{
			ID:            run.ID,
			LaunchID:      launch.ID,
			Status:        ScheduleRunStatusFailed,
			AudienceTotal: launch.AudienceTotal,
			EnqueuedTotal: launch.EnqueuedTotal,
			ErrorMessage:  err.Error(),
		})
		if updateErr != nil {
			return errors.Join(err, updateErr)
		}
		return err
	}
	_, err = s.store.CompleteScheduleRun(ctx, CompleteScheduleRunInput{
		ID:            run.ID,
		LaunchID:      launch.ID,
		Status:        ScheduleRunStatusCompleted,
		AudienceTotal: launch.AudienceTotal,
		EnqueuedTotal: launch.EnqueuedTotal,
	})
	return err
}

func scheduleSlotDue(
	now time.Time,
	timezone string,
	slot CampaignScheduleSlot,
	graceMinutes int,
) (bool, string, time.Time) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return false, "", time.Time{}
	}
	localNow := now.In(location)
	if !slotRunsOnWeekday(slot, localNow.Weekday()) {
		return false, "", time.Time{}
	}
	localClock, err := time.Parse("15:04", slot.LocalTime)
	if err != nil {
		return false, "", time.Time{}
	}
	scheduledLocal := time.Date(
		localNow.Year(),
		localNow.Month(),
		localNow.Day(),
		localClock.Hour(),
		localClock.Minute(),
		0,
		0,
		location,
	)
	if graceMinutes <= 0 {
		graceMinutes = 10
	}
	if localNow.Before(scheduledLocal) || !localNow.Before(scheduledLocal.Add(time.Duration(graceMinutes)*time.Minute)) {
		return false, "", time.Time{}
	}
	return true, scheduledLocal.Format("2006-01-02"), scheduledLocal.UTC()
}

func slotRunsOnWeekday(slot CampaignScheduleSlot, weekday time.Weekday) bool {
	day := int(weekday)
	if day == 0 {
		day = 7
	}
	for _, candidate := range slot.DaysOfWeek {
		if candidate == day {
			return true
		}
	}
	return false
}

func (s *Service) ListCampaignLaunches(ctx context.Context, campaignID string) ([]CampaignLaunch, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	return s.store.ListCampaignLaunches(ctx, campaignID)
}

func (s *Service) EnqueueCampaignLaunch(
	ctx context.Context,
	campaignID string,
	launchID string,
) (CampaignLaunch, error) {
	campaignID = strings.TrimSpace(campaignID)
	launchID = strings.TrimSpace(launchID)
	if campaignID == "" {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	if launchID == "" {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("launch_id is required"))
	}
	if s.audience == nil {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("audience builder is not configured"))
	}
	if s.delivery == nil {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("delivery producer is not configured"))
	}
	launch, err := s.store.GetCampaignLaunch(ctx, launchID)
	if err != nil {
		return CampaignLaunch{}, err
	}
	if launch.CampaignID != campaignID {
		return CampaignLaunch{}, ErrNotFound
	}
	if launch.Status != CampaignLaunchStatusCompleted {
		return CampaignLaunch{}, errors.Join(ErrInvalidInput, errors.New("campaign launch audience is not completed"))
	}
	if launch.EnqueueStatus == EnqueueStatusCompleted {
		return launch, nil
	}
	if _, err := s.store.UpdateCampaignLaunchEnqueue(ctx, UpdateCampaignLaunchEnqueueInput{
		ID:            launch.ID,
		EnqueueStatus: EnqueueStatusEnqueuing,
	}); err != nil {
		return CampaignLaunch{}, err
	}
	enqueued, err := s.enqueueAudience(ctx, launch.ID)
	if err != nil {
		failed, updateErr := s.store.UpdateCampaignLaunchEnqueue(ctx, UpdateCampaignLaunchEnqueueInput{
			ID:            launch.ID,
			EnqueueStatus: EnqueueStatusFailed,
			EnqueuedTotal: enqueued,
			EnqueueError:  err.Error(),
		})
		if updateErr != nil {
			return CampaignLaunch{}, errors.Join(err, updateErr)
		}
		return failed, err
	}
	return s.store.UpdateCampaignLaunchEnqueue(ctx, UpdateCampaignLaunchEnqueueInput{
		ID:            launch.ID,
		EnqueueStatus: EnqueueStatusCompleted,
		EnqueuedTotal: enqueued,
	})
}

func (s *Service) enqueueAudience(ctx context.Context, launchID string) (int64, error) {
	var enqueued int64
	var afterShard uint16
	afterSubscriptionID := ""
	for {
		rows, err := s.audience.AudienceBatch(ctx, AudienceBatchInput{
			LaunchID:            launchID,
			AfterShard:          afterShard,
			AfterSubscriptionID: afterSubscriptionID,
			Limit:               s.batchSize,
		})
		if err != nil {
			return enqueued, err
		}
		if len(rows) == 0 {
			return enqueued, nil
		}
		tasks := make([]DeliveryTask, 0, len(rows))
		now := time.Now().UTC()
		for _, row := range rows {
			if row.SubscriptionID == "" {
				return enqueued, errors.Join(ErrInvalidInput, errors.New("audience row subscription_id is required"))
			}
			deliveryID, err := generateDeliveryID()
			if err != nil {
				return enqueued, err
			}
			tasks = append(tasks, DeliveryTask{
				DeliveryID:     deliveryID,
				LaunchID:       row.LaunchID,
				CampaignID:     row.CampaignID,
				SourceID:       row.SourceID,
				SubscriptionID: row.SubscriptionID,
				Endpoint:       row.Endpoint,
				P256DH:         row.P256DH,
				Auth:           row.Auth,
				Shard:          row.Shard,
				Attempt:        0,
				CreatedAt:      now,
			})
		}
		if err := s.delivery.ProduceDeliveryTasks(ctx, tasks); err != nil {
			return enqueued, err
		}
		enqueued += int64(len(rows))
		last := rows[len(rows)-1]
		afterShard = last.Shard
		afterSubscriptionID = last.SubscriptionID
	}
}

func generateDeliveryID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate delivery id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}

func (s *Service) CreateCreative(ctx context.Context, input CreateCreativeInput) (Creative, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.URL = strings.TrimSpace(input.URL)
	input.Icon = strings.TrimSpace(input.Icon)
	if input.CampaignID == "" {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	if input.Title == "" {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("title is required"))
	}
	if input.Body == "" {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("body is required"))
	}
	if input.URL == "" {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("url is required"))
	}
	if input.DailyCapPerSubscription < 0 {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("daily_cap_per_subscription must be greater than or equal to zero"))
	}
	if input.TotalCapPerSubscription < 0 {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("total_cap_per_subscription must be greater than or equal to zero"))
	}
	return s.store.CreateCreative(ctx, input)
}

func (s *Service) ListCreatives(ctx context.Context, campaignID string) ([]Creative, error) {
	return s.store.ListCreatives(ctx, strings.TrimSpace(campaignID))
}

func (s *Service) UpdateCreativeStatus(ctx context.Context, input UpdateCreativeStatusInput) (Creative, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Status = strings.TrimSpace(input.Status)
	if input.ID == "" {
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	switch input.Status {
	case CreativeStatusActive, CreativeStatusPaused, CreativeStatusArchived:
		return s.store.UpdateCreativeStatus(ctx, input)
	default:
		return Creative{}, errors.Join(ErrInvalidInput, errors.New("invalid creative status"))
	}
}

func (s *Service) CreateCreativeProviderConfig(
	ctx context.Context,
	input CreateCreativeProviderConfigInput,
) (CreativeProviderConfig, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Name = strings.TrimSpace(input.Name)
	input.ProviderName = strings.ToLower(strings.TrimSpace(input.ProviderName))
	input.FetchURL = strings.TrimSpace(input.FetchURL)
	input.RequestHeaders = normalizeHeaders(input.RequestHeaders)
	if input.CampaignID == "" {
		return CreativeProviderConfig{}, errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	if input.Name == "" {
		return CreativeProviderConfig{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	if input.ProviderName == "" {
		return CreativeProviderConfig{}, errors.Join(ErrInvalidInput, errors.New("provider_name is required"))
	}
	if err := validateFetchURL(input.FetchURL); err != nil {
		return CreativeProviderConfig{}, err
	}
	return s.store.CreateCreativeProviderConfig(ctx, input)
}

func (s *Service) ListCreativeProviderConfigs(
	ctx context.Context,
	campaignID string,
) ([]CreativeProviderConfig, error) {
	return s.store.ListCreativeProviderConfigs(ctx, strings.TrimSpace(campaignID))
}

func (s *Service) UpdateCreativeProviderConfigStatus(
	ctx context.Context,
	input UpdateCreativeProviderConfigStatusInput,
) (CreativeProviderConfig, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Status = strings.TrimSpace(input.Status)
	if input.ID == "" {
		return CreativeProviderConfig{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	switch input.Status {
	case ProviderConfigStatusActive, ProviderConfigStatusPaused, ProviderConfigStatusArchived:
		return s.store.UpdateCreativeProviderConfigStatus(ctx, input)
	default:
		return CreativeProviderConfig{}, errors.Join(ErrInvalidInput, errors.New("invalid provider config status"))
	}
}

func (s *Service) ListCreativeSyncLogs(
	ctx context.Context,
	providerConfigID string,
	campaignID string,
) ([]CreativeSyncLog, error) {
	return s.store.ListCreativeSyncLogs(
		ctx,
		strings.TrimSpace(providerConfigID),
		strings.TrimSpace(campaignID),
	)
}

func (s *Service) SyncCreativeProviderConfig(
	ctx context.Context,
	id string,
) (CreativeSyncLog, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CreativeSyncLog{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	if s.providerFetcher == nil {
		return CreativeSyncLog{}, errors.Join(ErrInvalidInput, errors.New("creative provider fetcher is not configured"))
	}
	config, err := s.store.GetCreativeProviderConfig(ctx, id)
	if err != nil {
		return CreativeSyncLog{}, err
	}
	if config.Status != ProviderConfigStatusActive {
		return CreativeSyncLog{}, errors.Join(ErrInvalidInput, errors.New("creative provider config must be active"))
	}
	locked, err := s.store.TryAcquireCreativeProviderSyncLock(ctx, config.ID)
	if err != nil {
		return CreativeSyncLog{}, err
	}
	if !locked {
		return CreativeSyncLog{}, errors.Join(ErrInvalidInput, errors.New("creative provider sync is already running"))
	}
	defer func() {
		_ = s.store.ReleaseCreativeProviderSyncLock(context.Background(), config.ID)
	}()
	log, err := s.store.CreateCreativeSyncLog(ctx, config.ID, config.CampaignID)
	if err != nil {
		return CreativeSyncLog{}, err
	}
	providerCreatives, err := s.providerFetcher.FetchCreatives(ctx, config)
	if err != nil {
		return s.completeCreativeSyncLog(ctx, log.ID, CreativeSyncLogStatusFailed, 0, 0, err)
	}
	upserted := 0
	externalIDs := make([]string, 0, len(providerCreatives))
	for _, creative := range providerCreatives {
		normalized, err := normalizeProviderCreative(creative)
		if err != nil {
			return s.completeCreativeSyncLog(
				ctx,
				log.ID,
				CreativeSyncLogStatusFailed,
				len(providerCreatives),
				upserted,
				err,
			)
		}
		externalIDs = append(externalIDs, normalized.ExternalID)
		_, err = s.store.UpsertProviderCreative(ctx, UpsertProviderCreativeInput{
			CampaignID:              config.CampaignID,
			ProviderConfigID:        config.ID,
			ProviderName:            config.ProviderName,
			ProviderExternalID:      normalized.ExternalID,
			Title:                   normalized.Title,
			Body:                    normalized.Body,
			URL:                     normalized.URL,
			Icon:                    normalized.Icon,
			Status:                  normalized.Status,
			DailyCapPerSubscription: normalized.DailyCapPerSubscription,
			TotalCapPerSubscription: normalized.TotalCapPerSubscription,
			RawProviderPayload:      normalized.RawProviderPayload,
		})
		if err != nil {
			return s.completeCreativeSyncLog(
				ctx,
				log.ID,
				CreativeSyncLogStatusFailed,
				len(providerCreatives),
				upserted,
				err,
			)
		}
		upserted++
	}
	if _, err := s.store.MarkMissingProviderCreativesStale(ctx, config.ID, externalIDs); err != nil {
		return s.completeCreativeSyncLog(
			ctx,
			log.ID,
			CreativeSyncLogStatusFailed,
			len(providerCreatives),
			upserted,
			err,
		)
	}
	return s.store.CompleteCreativeSyncLog(ctx, CompleteCreativeSyncLogInput{
		ID:            log.ID,
		Status:        CreativeSyncLogStatusCompleted,
		FetchedTotal:  len(providerCreatives),
		UpsertedTotal: upserted,
	})
}

func (s *Service) completeCreativeSyncLog(
	ctx context.Context,
	id string,
	status string,
	fetchedTotal int,
	upsertedTotal int,
	cause error,
) (CreativeSyncLog, error) {
	log, err := s.store.CompleteCreativeSyncLog(ctx, CompleteCreativeSyncLogInput{
		ID:            id,
		Status:        status,
		FetchedTotal:  fetchedTotal,
		UpsertedTotal: upsertedTotal,
		ErrorMessage:  cause.Error(),
	})
	if err != nil {
		return CreativeSyncLog{}, errors.Join(cause, err)
	}
	return log, cause
}

func normalizeProviderCreative(creative ProviderCreative) (ProviderCreative, error) {
	creative.ExternalID = strings.TrimSpace(creative.ExternalID)
	creative.Title = strings.TrimSpace(creative.Title)
	creative.Body = strings.TrimSpace(creative.Body)
	creative.URL = strings.TrimSpace(creative.URL)
	creative.Icon = strings.TrimSpace(creative.Icon)
	creative.Status = strings.TrimSpace(creative.Status)
	if creative.Status == "" {
		creative.Status = CreativeStatusActive
	}
	if creative.ExternalID == "" {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative external_id is required"))
	}
	if creative.Title == "" {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative title is required"))
	}
	if creative.Body == "" {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative body is required"))
	}
	if creative.URL == "" {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative url is required"))
	}
	switch creative.Status {
	case CreativeStatusActive, CreativeStatusPaused, CreativeStatusArchived:
	default:
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("invalid provider creative status"))
	}
	if creative.DailyCapPerSubscription < 0 {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative daily cap must be greater than or equal to zero"))
	}
	if creative.TotalCapPerSubscription < 0 {
		return ProviderCreative{}, errors.Join(ErrInvalidInput, errors.New("provider creative total cap must be greater than or equal to zero"))
	}
	if creative.RawProviderPayload == "" {
		creative.RawProviderPayload = "{}"
	}
	return creative, nil
}

func normalizeHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	normalized := map[string]string{}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		normalized[key] = value
	}
	return normalized
}

func validateFetchURL(raw string) error {
	if raw == "" {
		return errors.Join(ErrInvalidInput, errors.New("fetch_url is required"))
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.Join(ErrInvalidInput, fmt.Errorf("invalid fetch_url: %w", err))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.Join(ErrInvalidInput, errors.New("fetch_url must use http or https"))
	}
	if parsed.Host == "" {
		return errors.Join(ErrInvalidInput, errors.New("fetch_url host is required"))
	}
	return nil
}

type HTTPProviderFetcher struct {
	Client *http.Client
	Config ProviderFetchConfig
}

func (f HTTPProviderFetcher) FetchCreatives(
	ctx context.Context,
	config CreativeProviderConfig,
) ([]ProviderCreative, error) {
	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}
	fetchConfig := normalizeProviderFetchConfig(f.Config)
	body, err := f.fetchBody(ctx, client, config, fetchConfig)
	if err != nil {
		return nil, err
	}
	creatives, err := parseProviderCreatives(body)
	if err != nil {
		return nil, err
	}
	return creatives, nil
}

func (f HTTPProviderFetcher) fetchBody(
	ctx context.Context,
	client *http.Client,
	config CreativeProviderConfig,
	fetchConfig ProviderFetchConfig,
) ([]byte, error) {
	requestClient := *client
	requestClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateFetchTarget(req.Context(), req.URL, fetchConfig.AllowPrivateFetchURLs); err != nil {
			return err
		}
		if client.CheckRedirect != nil {
			return client.CheckRedirect(req, via)
		}
		if len(via) >= 5 {
			return errors.New("provider redirect limit exceeded")
		}
		return nil
	}

	var lastErr error
	for attempt := 1; attempt <= fetchConfig.MaxAttempts; attempt++ {
		body, retryable, err := fetchProviderBodyOnce(
			ctx,
			&requestClient,
			config,
			fetchConfig,
		)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || attempt == fetchConfig.MaxAttempts {
			break
		}
		timer := time.NewTimer(fetchConfig.RetryBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func fetchProviderBodyOnce(
	ctx context.Context,
	client *http.Client,
	config CreativeProviderConfig,
	fetchConfig ProviderFetchConfig,
) ([]byte, bool, error) {
	parsed, err := url.Parse(config.FetchURL)
	if err != nil {
		return nil, false, fmt.Errorf("parse provider fetch url: %w", err)
	}
	if err := validateFetchTarget(ctx, parsed, fetchConfig.AllowPrivateFetchURLs); err != nil {
		return nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.FetchURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build provider request: %w", err)
	}
	for key, value := range config.RequestHeaders {
		req.Header.Set(key, value)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("fetch provider creatives: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		retryable := res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500
		return nil, retryable, fmt.Errorf("provider returned status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, fetchConfig.MaxBodyBytes+1))
	if err != nil {
		return nil, true, fmt.Errorf("read provider response: %w", err)
	}
	if int64(len(body)) > fetchConfig.MaxBodyBytes {
		return nil, false, errors.New("provider response body exceeds configured limit")
	}
	return body, false, nil
}

func normalizeProviderFetchConfig(config ProviderFetchConfig) ProviderFetchConfig {
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 2 * 1024 * 1024
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 500 * time.Millisecond
	}
	return config
}

func validateFetchTarget(ctx context.Context, target *url.URL, allowPrivate bool) error {
	if target.Scheme != "http" && target.Scheme != "https" {
		return errors.New("provider fetch URL must use http or https")
	}
	host := target.Hostname()
	if host == "" {
		return errors.New("provider fetch URL host is required")
	}
	if allowPrivate {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if privateProviderIP(ip) {
			return errors.New("provider fetch URL resolves to a private address")
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve provider fetch host: %w", err)
	}
	if len(ips) == 0 {
		return errors.New("provider fetch host has no addresses")
	}
	for _, addr := range ips {
		if privateProviderIP(addr.IP) {
			return errors.New("provider fetch URL resolves to a private address")
		}
	}
	return nil
}

func privateProviderIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

type providerCreativeJSON struct {
	ID                      string `json:"id"`
	ExternalID              string `json:"external_id"`
	Title                   string `json:"title"`
	Body                    string `json:"body"`
	URL                     string `json:"url"`
	Icon                    string `json:"icon"`
	Status                  string `json:"status"`
	DailyCapPerSubscription int    `json:"daily_cap_per_subscription"`
	TotalCapPerSubscription int    `json:"total_cap_per_subscription"`
}

func parseProviderCreatives(body []byte) ([]ProviderCreative, error) {
	var wrapped struct {
		Creatives []providerCreativeJSON `json:"creatives"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Creatives != nil {
		return providerCreativesFromJSON(wrapped.Creatives)
	}
	var items []providerCreativeJSON
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("parse provider creatives: %w", err)
	}
	return providerCreativesFromJSON(items)
}

func providerCreativesFromJSON(items []providerCreativeJSON) ([]ProviderCreative, error) {
	creatives := make([]ProviderCreative, 0, len(items))
	for _, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal provider creative raw payload: %w", err)
		}
		externalID := strings.TrimSpace(item.ExternalID)
		if externalID == "" {
			externalID = strings.TrimSpace(item.ID)
		}
		creatives = append(creatives, ProviderCreative{
			ExternalID:              externalID,
			Title:                   item.Title,
			Body:                    item.Body,
			URL:                     item.URL,
			Icon:                    item.Icon,
			Status:                  item.Status,
			DailyCapPerSubscription: item.DailyCapPerSubscription,
			TotalCapPerSubscription: item.TotalCapPerSubscription,
			RawProviderPayload:      string(raw),
		})
	}
	return creatives, nil
}

func (s *Service) SourceSnippet(ctx context.Context, sourceID string) (string, error) {
	source, err := s.GetSource(ctx, sourceID)
	if err != nil {
		return "", err
	}
	scriptURL := s.publicURL + "/api/sdk/sources/" + url.PathEscape(source.ID) + ".js"
	return fmt.Sprintf(`<script async src=%q></script>`, scriptURL), nil
}

func (s *Service) SourceStats(ctx context.Context, sourceID string) (SourceStats, error) {
	source, err := s.GetSource(ctx, sourceID)
	if err != nil {
		return SourceStats{}, err
	}
	if s.subscribers == nil {
		return SourceStats{
			SourceID:       source.ID,
			EventBreakdown: map[string]int64{},
			Health:         sourceHealth(source, 0, 0, ""),
		}, nil
	}
	count, err := s.subscribers.CountBySource(ctx, source.ID)
	if err != nil {
		return SourceStats{}, err
	}
	today, err := s.subscribers.CountBySourceToday(ctx, source.ID)
	if err != nil {
		return SourceStats{}, err
	}
	eventsToday, err := s.subscribers.CountEventsBySourceToday(ctx, source.ID)
	if err != nil {
		return SourceStats{}, err
	}
	breakdown, err := s.subscribers.CountEventsBySourceTodayByType(ctx, source.ID)
	if err != nil {
		return SourceStats{}, err
	}
	lastEventAt, err := s.subscribers.LastEventAtBySource(ctx, source.ID)
	if err != nil {
		return SourceStats{}, err
	}
	return SourceStats{
		SourceID:         source.ID,
		Subscribers:      count,
		SubscribersToday: today,
		EventsToday:      eventsToday,
		EventBreakdown:   breakdown,
		LastEventAt:      lastEventAt,
		Health:           sourceHealth(source, today, eventsToday, lastEventAt),
	}, nil
}

func sourceHealth(source Source, subscribersToday int64, eventsToday int64, lastEventAt string) SourceHealth {
	issues := []string{}
	if strings.TrimSpace(source.Domain) == "" {
		issues = append(issues, "domain_missing")
	}
	if strings.TrimSpace(source.VAPIDKeyID) == "" {
		issues = append(issues, "vapid_key_missing")
	}
	if subscribersToday == 0 {
		issues = append(issues, "no_subscribers_today")
	}
	if eventsToday == 0 {
		issues = append(issues, "no_events_today")
	}
	if strings.TrimSpace(lastEventAt) == "" {
		issues = append(issues, "no_recent_activity")
	}
	status := "ok"
	if len(issues) > 0 {
		status = "attention"
	}
	return SourceHealth{Status: status, Issues: issues}
}

func generateVAPIDKeyPair() (string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate vapid key: %w", err)
	}
	publicBytes := encodeP256PublicKey(key.PublicKey.X, key.PublicKey.Y)
	privateBytes := padPrivateKey(key.D)
	return base64.RawURLEncoding.EncodeToString(publicBytes),
		base64.RawURLEncoding.EncodeToString(privateBytes),
		nil
}

func padPrivateKey(value *big.Int) []byte {
	raw := value.Bytes()
	if len(raw) >= 32 {
		return raw
	}
	padded := make([]byte, 32)
	copy(padded[32-len(raw):], raw)
	return padded
}

func encodeP256PublicKey(x *big.Int, y *big.Int) []byte {
	encoded := make([]byte, 65)
	encoded[0] = 4
	x.FillBytes(encoded[1:33])
	y.FillBytes(encoded[33:65])
	return encoded
}
