package sender

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/devflex-pro/push-booster/packages/go/subscribers"
)

type fakeKeyStore struct{}

func (s fakeKeyStore) ActiveVAPIDKeyForSource(
	_ context.Context,
	sourceID string,
) (inventory.VAPIDKey, error) {
	return inventory.VAPIDKey{
		ID:         "key",
		PublicKey:  "public",
		PrivateKey: "private",
		Status:     inventory.VAPIDStatusActive,
	}, nil
}

type fakeTriggerStore struct{}

func (s fakeTriggerStore) CreateTrigger(
	_ context.Context,
	input subscribers.CreateTriggerInput,
) (subscribers.DeliveryTrigger, error) {
	return subscribers.DeliveryTrigger{
		TriggerID:      "11111111-1111-4111-8111-111111111111",
		DeliveryID:     "22222222-2222-4222-8222-222222222222",
		SubscriptionID: input.SubscriptionID,
		SourceID:       input.SourceID,
		CampaignID:     input.CampaignID,
	}, nil
}

type fakeEventStore struct {
	events []PushEventInput
}

func (s *fakeEventStore) RecordPushEvent(_ context.Context, input PushEventInput) error {
	s.events = append(s.events, input)
	return nil
}

type fakeRetryProducer struct {
	topic string
	task  inventory.DeliveryTask
}

func (p *fakeRetryProducer) ProduceRetryTask(
	_ context.Context,
	topic string,
	task inventory.DeliveryTask,
) error {
	p.topic = topic
	p.task = task
	return nil
}

type fakeWebPushSender struct {
	statusCode int
	err        error
}

func (s fakeWebPushSender) SendTrigger(
	_ context.Context,
	_ inventory.DeliveryTask,
	_ inventory.VAPIDKey,
	_ subscribers.DeliveryTrigger,
) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return StaticResponse(s.statusCode), nil
}

func TestServiceProcessTaskSendsTrigger(t *testing.T) {
	t.Parallel()

	events := &fakeEventStore{}
	service := NewService(
		fakeKeyStore{},
		fakeTriggerStore{},
		events,
		&fakeRetryProducer{},
		fakeWebPushSender{statusCode: http.StatusCreated},
		nil,
		Config{},
	)
	if err := service.ProcessTask(context.Background(), validTask()); err != nil {
		t.Fatalf("process task failed: %v", err)
	}
	if events.events[len(events.events)-1].EventType != EventSent {
		t.Fatalf(
			"expected sent event, got %+v",
			events.events,
		)
	}
}

func TestServiceProcessTaskRetriesFailure(t *testing.T) {
	t.Parallel()

	events := &fakeEventStore{}
	retries := &fakeRetryProducer{}
	service := NewService(
		fakeKeyStore{},
		fakeTriggerStore{},
		events,
		retries,
		fakeWebPushSender{err: errors.New("send failed")},
		nil,
		Config{},
	)
	if err := service.ProcessTask(context.Background(), validTask()); err != nil {
		t.Fatalf("process task failed: %v", err)
	}
	if retries.topic != Retry1mTopic || retries.task.Attempt != 1 {
		t.Fatalf(
			"expected retry task, got topic=%q task=%+v",
			retries.topic,
			retries.task,
		)
	}
	if events.events[len(events.events)-1].EventType != EventRetryEnqueued {
		t.Fatalf(
			"expected retry event, got %+v",
			events.events,
		)
	}
}

func TestServiceProcessTaskInvalidEndpointDoesNotRetry(t *testing.T) {
	t.Parallel()

	events := &fakeEventStore{}
	retries := &fakeRetryProducer{}
	service := NewService(
		fakeKeyStore{},
		fakeTriggerStore{},
		events,
		retries,
		fakeWebPushSender{statusCode: http.StatusGone},
		nil,
		Config{},
	)
	if err := service.ProcessTask(context.Background(), validTask()); err != nil {
		t.Fatalf("process task failed: %v", err)
	}
	if retries.topic != "" {
		t.Fatalf(
			"expected no retry, got topic=%q",
			retries.topic,
		)
	}
	if events.events[len(events.events)-1].EventType != EventInvalidEndpoint {
		t.Fatalf(
			"expected invalid endpoint event, got %+v",
			events.events,
		)
	}
}

func TestServiceDefaultsConcurrency(t *testing.T) {
	t.Parallel()

	service := NewService(
		fakeKeyStore{},
		fakeTriggerStore{},
		&fakeEventStore{},
		&fakeRetryProducer{},
		fakeWebPushSender{statusCode: http.StatusCreated},
		nil,
		Config{},
	)

	if service.Concurrency() != 8 {
		t.Fatalf("expected default concurrency 8, got %d", service.Concurrency())
	}
}

func TestServiceKeepsConfiguredConcurrency(t *testing.T) {
	t.Parallel()

	service := NewService(
		fakeKeyStore{},
		fakeTriggerStore{},
		&fakeEventStore{},
		&fakeRetryProducer{},
		fakeWebPushSender{statusCode: http.StatusCreated},
		nil,
		Config{Concurrency: 3},
	)

	if service.Concurrency() != 3 {
		t.Fatalf("expected configured concurrency 3, got %d", service.Concurrency())
	}
}

func TestEncryptWebPushPayloadBuildsAES128GCMBody(t *testing.T) {
	t.Parallel()

	curve := ecdh.P256()
	receiverKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate receiver key: %v", err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatalf("generate auth secret: %v", err)
	}
	body, serverPublicKey, err := encryptWebPushPayload(
		[]byte(`{"trigger_id":"trigger"}`),
		base64.RawURLEncoding.EncodeToString(receiverKey.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(authSecret),
	)
	if err != nil {
		t.Fatalf("encrypt payload failed: %v", err)
	}
	if len(serverPublicKey) != p256PublicKeySize {
		t.Fatalf(
			"expected server public key size %d, got %d",
			p256PublicKeySize,
			len(serverPublicKey),
		)
	}
	if len(body) <= 21+p256PublicKeySize {
		t.Fatalf(
			"expected encrypted body, got %d bytes",
			len(body),
		)
	}
	if int(body[20]) != p256PublicKeySize {
		t.Fatalf(
			"expected key size marker %d, got %d",
			p256PublicKeySize,
			body[20],
		)
	}
}

func TestVAPIDJWTBuildsSignedToken(t *testing.T) {
	t.Parallel()

	privateKey := make([]byte, p256PrivateKeySize)
	if _, err := rand.Read(privateKey); err != nil {
		t.Fatalf("generate vapid key: %v", err)
	}
	token, err := vapidJWT(
		"https://push.example/endpoint",
		base64.RawURLEncoding.EncodeToString(privateKey),
		"mailto:admin@example.com",
	)
	if err != nil {
		t.Fatalf("create vapid jwt failed: %v", err)
	}
	if parts := strings.Split(token, "."); len(parts) != 3 {
		t.Fatalf(
			"expected jwt with 3 parts, got %q",
			token,
		)
	}
}

func validTask() inventory.DeliveryTask {
	return inventory.DeliveryTask{
		DeliveryID:     "33333333-3333-4333-8333-333333333333",
		LaunchID:       "44444444-4444-4444-8444-444444444444",
		CampaignID:     "55555555-5555-4555-8555-555555555555",
		SourceID:       "66666666-6666-4666-8666-666666666666",
		SubscriptionID: "77777777-7777-4777-8777-777777777777",
		Endpoint:       "https://push.example/endpoint",
		P256DH:         "p256dh",
		Auth:           "auth",
	}
}
