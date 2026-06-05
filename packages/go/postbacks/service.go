package postbacks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidInput = errors.New("invalid postback input")
	ErrNotFound     = errors.New("postback record not found")
	ErrForbidden    = errors.New("postback token validation failed")
	ErrDuplicate    = errors.New("duplicate postback")
)

const (
	StatusActive   = "active"
	StatusPaused   = "paused"
	StatusArchived = "archived"

	AttributionResolved   = "resolved"
	AttributionUnresolved = "unresolved"
)

type Config struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	SourceID            string    `json:"source_id,omitempty"`
	Token               string    `json:"token,omitempty"`
	Status              string    `json:"status"`
	ClickIDParam        string    `json:"click_id_param"`
	DeliveryIDParam     string    `json:"delivery_id_param"`
	SubscriptionIDParam string    `json:"subscription_id_param"`
	ExternalIDParam     string    `json:"external_id_param"`
	PayoutParam         string    `json:"payout_param"`
	CurrencyParam       string    `json:"currency_param"`
	StatusParam         string    `json:"status_param"`
	DefaultCurrency     string    `json:"default_currency"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateConfigInput struct {
	Name                string `json:"name"`
	SourceID            string `json:"source_id"`
	Token               string `json:"token"`
	ClickIDParam        string `json:"click_id_param"`
	DeliveryIDParam     string `json:"delivery_id_param"`
	SubscriptionIDParam string `json:"subscription_id_param"`
	ExternalIDParam     string `json:"external_id_param"`
	PayoutParam         string `json:"payout_param"`
	CurrencyParam       string `json:"currency_param"`
	StatusParam         string `json:"status_param"`
	DefaultCurrency     string `json:"default_currency"`
}

type UpdateConfigStatusInput struct {
	ID     string
	Status string
}

type RecentEventsInput struct {
	PostbackConfigID string
	Limit            int
}

type IngestInput struct {
	ConfigID string
	Method   string
	Query    url.Values
	Form     url.Values
	RawBody  string
}

type Event struct {
	PostbackConfigID string    `json:"postback_config_id"`
	DedupeKey        string    `json:"dedupe_key"`
	ExternalID       string    `json:"external_id"`
	ClickID          string    `json:"click_id"`
	DeliveryID       string    `json:"delivery_id"`
	SubscriptionID   string    `json:"subscription_id"`
	SourceID         string    `json:"source_id"`
	CampaignID       string    `json:"campaign_id"`
	CreativeID       string    `json:"creative_id"`
	Payout           float64   `json:"payout"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	Attribution      string    `json:"attribution_status"`
	RawPayload       string    `json:"raw_payload"`
	ReceivedAt       time.Time `json:"received_at"`
}

type Attribution struct {
	DeliveryID     string
	SubscriptionID string
	SourceID       string
	CampaignID     string
	CreativeID     string
}

type ConfigStore interface {
	CreateConfig(ctx context.Context, input CreateConfigInput) (Config, error)
	ListConfigs(ctx context.Context) ([]Config, error)
	GetConfig(ctx context.Context, id string) (Config, error)
	UpdateConfigStatus(ctx context.Context, input UpdateConfigStatusInput) (Config, error)
}

type EventStore interface {
	RecordEvent(ctx context.Context, event Event) error
	EventExists(
		ctx context.Context,
		configID string,
		dedupeKey string,
	) (bool, error)
	RecentEvents(ctx context.Context, input RecentEventsInput) ([]Event, error)
	ResolveAttribution(ctx context.Context, input Attribution) (Attribution, error)
}

type Service struct {
	configs ConfigStore
	events  EventStore
}

func NewService(configs ConfigStore, events EventStore) *Service {
	return &Service{configs: configs, events: events}
}

func (s *Service) CreateConfig(ctx context.Context, input CreateConfigInput) (Config, error) {
	input = defaults(input)
	input.Name = strings.TrimSpace(input.Name)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.Token = strings.TrimSpace(input.Token)
	if input.Name == "" {
		return Config{}, errors.Join(ErrInvalidInput, errors.New("name is required"))
	}
	if input.Token == "" {
		input.Token = generateToken(input.Name)
	}
	return s.configs.CreateConfig(ctx, input)
}

func (s *Service) ListConfigs(ctx context.Context) ([]Config, error) {
	return s.configs.ListConfigs(ctx)
}

func (s *Service) UpdateConfigStatus(
	ctx context.Context,
	input UpdateConfigStatusInput,
) (Config, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Status = strings.TrimSpace(input.Status)
	if input.ID == "" {
		return Config{}, errors.Join(ErrInvalidInput, errors.New("id is required"))
	}
	if !validStatus(input.Status) {
		return Config{}, errors.Join(ErrInvalidInput, errors.New("invalid status"))
	}
	return s.configs.UpdateConfigStatus(ctx, input)
}

func (s *Service) RecentEvents(ctx context.Context, input RecentEventsInput) ([]Event, error) {
	input.PostbackConfigID = strings.TrimSpace(input.PostbackConfigID)
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 50
	}
	return s.events.RecentEvents(ctx, input)
}

func (s *Service) Ingest(ctx context.Context, input IngestInput) (Event, error) {
	input.ConfigID = strings.TrimSpace(input.ConfigID)
	if input.ConfigID == "" {
		return Event{}, errors.Join(ErrInvalidInput, errors.New("postback_config_id is required"))
	}
	cfg, err := s.configs.GetConfig(ctx, input.ConfigID)
	if err != nil {
		return Event{}, err
	}
	if cfg.Status != StatusActive {
		return Event{}, errors.Join(ErrInvalidInput, errors.New("postback config is not active"))
	}
	values := mergeValues(input.Query, input.Form)
	if cfg.Token != "" && first(
		values,
		"token",
		"key",
		"api_key",
	) != cfg.Token {
		return Event{}, ErrForbidden
	}
	event := eventFromValues(cfg, values, input.RawBody)
	exists, err := s.events.EventExists(ctx, cfg.ID, event.DedupeKey)
	if err != nil {
		return Event{}, err
	}
	if exists {
		return event, ErrDuplicate
	}
	attribution, err := s.events.ResolveAttribution(ctx, Attribution{
		DeliveryID:     event.DeliveryID,
		SubscriptionID: event.SubscriptionID,
	})
	if err != nil {
		return Event{}, err
	}
	event.DeliveryID = firstNonEmpty(event.DeliveryID, attribution.DeliveryID)
	event.SubscriptionID = firstNonEmpty(event.SubscriptionID, attribution.SubscriptionID)
	event.SourceID = firstNonEmpty(cfg.SourceID, attribution.SourceID)
	event.CampaignID = attribution.CampaignID
	event.CreativeID = attribution.CreativeID
	if event.SourceID != "" {
		event.Attribution = AttributionResolved
	}
	if err := s.events.RecordEvent(ctx, event); err != nil {
		return Event{}, err
	}
	return event, nil
}

func defaults(input CreateConfigInput) CreateConfigInput {
	if strings.TrimSpace(input.ClickIDParam) == "" {
		input.ClickIDParam = "click_id"
	}
	if strings.TrimSpace(input.DeliveryIDParam) == "" {
		input.DeliveryIDParam = "delivery_id"
	}
	if strings.TrimSpace(input.SubscriptionIDParam) == "" {
		input.SubscriptionIDParam = "subscription_id"
	}
	if strings.TrimSpace(input.ExternalIDParam) == "" {
		input.ExternalIDParam = "external_id"
	}
	if strings.TrimSpace(input.PayoutParam) == "" {
		input.PayoutParam = "payout"
	}
	if strings.TrimSpace(input.CurrencyParam) == "" {
		input.CurrencyParam = "currency"
	}
	if strings.TrimSpace(input.StatusParam) == "" {
		input.StatusParam = "status"
	}
	if strings.TrimSpace(input.DefaultCurrency) == "" {
		input.DefaultCurrency = "USD"
	}
	return input
}

func eventFromValues(cfg Config, values url.Values, rawBody string) Event {
	clickID := strings.TrimSpace(values.Get(cfg.ClickIDParam))
	deliveryID := strings.TrimSpace(values.Get(cfg.DeliveryIDParam))
	if deliveryID == "" {
		deliveryID = clickID
	}
	externalID := strings.TrimSpace(values.Get(cfg.ExternalIDParam))
	currency := strings.ToUpper(strings.TrimSpace(values.Get(cfg.CurrencyParam)))
	if currency == "" {
		currency = cfg.DefaultCurrency
	}
	status := strings.TrimSpace(values.Get(cfg.StatusParam))
	if status == "" {
		status = "approved"
	}
	rawPayload := rawPayload(values, rawBody)
	dedupeKey := externalID
	if dedupeKey == "" {
		dedupeKey = hash(rawPayload)
	}
	return Event{
		PostbackConfigID: cfg.ID,
		DedupeKey:        dedupeKey,
		ExternalID:       externalID,
		ClickID:          clickID,
		DeliveryID:       deliveryID,
		SubscriptionID:   strings.TrimSpace(values.Get(cfg.SubscriptionIDParam)),
		SourceID:         cfg.SourceID,
		Payout:           parsePayout(values.Get(cfg.PayoutParam)),
		Currency:         currency,
		Status:           status,
		Attribution:      AttributionUnresolved,
		RawPayload:       rawPayload,
		ReceivedAt:       time.Now().UTC(),
	}
}

func validStatus(status string) bool {
	return status == StatusActive || status == StatusPaused || status == StatusArchived
}

func mergeValues(query url.Values, form url.Values) url.Values {
	values := url.Values{}
	for key, items := range query {
		values[key] = append(values[key], items...)
	}
	for key, items := range form {
		values[key] = append(values[key], items...)
	}
	return values
}

func first(values url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parsePayout(value string) float64 {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func rawPayload(values url.Values, rawBody string) string {
	if strings.TrimSpace(rawBody) != "" {
		return rawBody
	}
	return values.Encode()
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func generateToken(seed string) string {
	return hash(fmt.Sprintf("%s:%d", seed, time.Now().UnixNano()))[:32]
}
