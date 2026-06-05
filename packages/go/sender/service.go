package sender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

const (
	Retry1mTopic  = "push.delivery.retry.1m"
	Retry10mTopic = "push.delivery.retry.10m"
	Retry1hTopic  = "push.delivery.retry.1h"
	DLQTopic      = "push.delivery.dlq"
)

const (
	EventConsumed        = "consumed"
	EventTriggerCreated  = "trigger_created"
	EventSent            = "sent"
	EventFailed          = "failed"
	EventInvalidEndpoint = "invalid_endpoint"
	EventRetryEnqueued   = "retry_enqueued"
	EventDLQEnqueued     = "dlq_enqueued"
)

var ErrInvalidInput = errors.New("invalid sender input")

type VAPIDKeyStore interface {
	ActiveVAPIDKeyForSource(
		ctx context.Context,
		sourceID string,
	) (inventory.VAPIDKey, error)
}

type TriggerStore interface {
	CreateTrigger(
		ctx context.Context,
		input subscribers.CreateTriggerInput,
	) (subscribers.DeliveryTrigger, error)
}

type EventStore interface {
	RecordPushEvent(ctx context.Context, input PushEventInput) error
}

type RetryProducer interface {
	ProduceRetryTask(
		ctx context.Context,
		topic string,
		task inventory.DeliveryTask,
	) error
}

type Throttler interface {
	Wait(ctx context.Context, endpoint string) error
}

type WebPushSender interface {
	SendTrigger(
		ctx context.Context,
		task inventory.DeliveryTask,
		key inventory.VAPIDKey,
		trigger subscribers.DeliveryTrigger,
	) (*http.Response, error)
}

type Config struct {
	TriggerTTL  time.Duration
	MaxAttempts int
	Concurrency int
}

type Service struct {
	keys     VAPIDKeyStore
	triggers TriggerStore
	events   EventStore
	retries  RetryProducer
	push     WebPushSender
	throttle Throttler
	cfg      Config
}

func NewService(
	keys VAPIDKeyStore,
	triggers TriggerStore,
	events EventStore,
	retries RetryProducer,
	push WebPushSender,
	throttle Throttler,
	cfg Config,
) *Service {
	if cfg.TriggerTTL == 0 {
		cfg.TriggerTTL = 5 * time.Minute
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 8
	}
	return &Service{
		keys:     keys,
		triggers: triggers,
		events:   events,
		retries:  retries,
		push:     push,
		throttle: throttle,
		cfg:      cfg,
	}
}

func (s *Service) Concurrency() int {
	return s.cfg.Concurrency
}

func (s *Service) ProcessTask(ctx context.Context, task inventory.DeliveryTask) error {
	if err := validateTask(task); err != nil {
		return err
	}
	if err := s.record(
		ctx,
		task,
		"",
		EventConsumed,
		"",
	); err != nil {
		return err
	}
	key, err := s.keys.ActiveVAPIDKeyForSource(ctx, task.SourceID)
	if err != nil {
		return s.fail(
			ctx,
			task,
			"",
			fmt.Errorf("active vapid key: %w", err),
		)
	}
	trigger, err := s.triggers.CreateTrigger(ctx, subscribers.CreateTriggerInput{
		SubscriptionID: task.SubscriptionID,
		SourceID:       task.SourceID,
		CampaignID:     task.CampaignID,
		TTL:            s.cfg.TriggerTTL,
	})
	if err != nil {
		return s.fail(
			ctx,
			task,
			"",
			fmt.Errorf("create trigger: %w", err),
		)
	}
	if err := s.record(
		ctx,
		task,
		trigger.TriggerID,
		EventTriggerCreated,
		"",
	); err != nil {
		return err
	}
	if s.throttle != nil {
		if err := s.throttle.Wait(ctx, task.Endpoint); err != nil {
			return s.fail(
				ctx,
				task,
				trigger.TriggerID,
				fmt.Errorf("wait provider throttle: %w", err),
			)
		}
	}
	resp, err := s.push.SendTrigger(
		ctx,
		task,
		key,
		trigger,
	)
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			if err != nil {
				return errors.Join(err, closeErr)
			}
			return closeErr
		}
	}
	if err != nil {
		return s.fail(
			ctx,
			task,
			trigger.TriggerID,
			err,
		)
	}
	if resp == nil {
		return s.fail(
			ctx,
			task,
			trigger.TriggerID,
			errors.New("web push response is nil"),
		)
	}
	if invalidEndpoint(resp.StatusCode) {
		return s.record(
			ctx,
			task,
			trigger.TriggerID,
			EventInvalidEndpoint,
			resp.Status,
		)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return s.fail(
			ctx,
			task,
			trigger.TriggerID,
			errors.New(resp.Status),
		)
	}
	return s.record(
		ctx,
		task,
		trigger.TriggerID,
		EventSent,
		"",
	)
}

func (s *Service) fail(
	ctx context.Context,
	task inventory.DeliveryTask,
	triggerID string,
	err error,
) error {
	if recordErr := s.record(
		ctx,
		task,
		triggerID,
		EventFailed,
		err.Error(),
	); recordErr != nil {
		return errors.Join(err, recordErr)
	}
	if task.Attempt+1 >= s.cfg.MaxAttempts {
		task.Attempt++
		if produceErr := s.retries.ProduceRetryTask(
			ctx,
			DLQTopic,
			task,
		); produceErr != nil {
			return errors.Join(err, produceErr)
		}
		if recordErr := s.record(
			ctx,
			task,
			triggerID,
			EventDLQEnqueued,
			err.Error(),
		); recordErr != nil {
			return errors.Join(err, recordErr)
		}
		return nil
	}
	task.Attempt++
	topic := retryTopic(task.Attempt)
	if produceErr := s.retries.ProduceRetryTask(
		ctx,
		topic,
		task,
	); produceErr != nil {
		return errors.Join(err, produceErr)
	}
	if recordErr := s.record(
		ctx,
		task,
		triggerID,
		EventRetryEnqueued,
		err.Error(),
	); recordErr != nil {
		return errors.Join(err, recordErr)
	}
	return nil
}

func (s *Service) record(
	ctx context.Context,
	task inventory.DeliveryTask,
	triggerID string,
	eventType string,
	errText string,
) error {
	return s.events.RecordPushEvent(ctx, PushEventInput{
		DeliveryID:     task.DeliveryID,
		TriggerID:      triggerID,
		LaunchID:       task.LaunchID,
		CampaignID:     task.CampaignID,
		SourceID:       task.SourceID,
		SubscriptionID: task.SubscriptionID,
		EventType:      eventType,
		Attempt:        task.Attempt,
		Error:          errText,
		OccurredAt:     time.Now().UTC(),
	})
}

func validateTask(task inventory.DeliveryTask) error {
	if strings.TrimSpace(task.LaunchID) == "" {
		return errors.Join(ErrInvalidInput, errors.New("launch_id is required"))
	}
	if strings.TrimSpace(task.DeliveryID) == "" {
		return errors.Join(ErrInvalidInput, errors.New("delivery_id is required"))
	}
	if strings.TrimSpace(task.CampaignID) == "" {
		return errors.Join(ErrInvalidInput, errors.New("campaign_id is required"))
	}
	if strings.TrimSpace(task.SourceID) == "" {
		return errors.Join(ErrInvalidInput, errors.New("source_id is required"))
	}
	if strings.TrimSpace(task.SubscriptionID) == "" {
		return errors.Join(ErrInvalidInput, errors.New("subscription_id is required"))
	}
	if strings.TrimSpace(task.Endpoint) == "" {
		return errors.Join(ErrInvalidInput, errors.New("endpoint is required"))
	}
	if strings.TrimSpace(task.P256DH) == "" {
		return errors.Join(ErrInvalidInput, errors.New("p256dh is required"))
	}
	if strings.TrimSpace(task.Auth) == "" {
		return errors.Join(ErrInvalidInput, errors.New("auth is required"))
	}
	return nil
}

func DecodeTask(value []byte) (inventory.DeliveryTask, error) {
	var task inventory.DeliveryTask
	if err := json.Unmarshal(value, &task); err != nil {
		return inventory.DeliveryTask{}, fmt.Errorf("decode delivery task: %w", err)
	}
	return task, nil
}

func StaticResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status: fmt.Sprintf(
			"%d %s",
			statusCode,
			http.StatusText(statusCode),
		),
		Body: io.NopCloser(strings.NewReader("")),
	}
}

func invalidEndpoint(statusCode int) bool {
	return statusCode == http.StatusGone || statusCode == http.StatusNotFound
}

func retryTopic(attempt int) string {
	switch attempt {
	case 1:
		return Retry1mTopic
	case 2:
		return Retry10mTopic
	default:
		return Retry1hTopic
	}
}
