package redpanda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/devflex-pro/push-booster/packages/go/inventory"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	DeliveryTasksTopic = "push.delivery.tasks"
	Retry1mTopic       = "push.delivery.retry.1m"
	Retry10mTopic      = "push.delivery.retry.10m"
	Retry1hTopic       = "push.delivery.retry.1h"
	DLQTopic           = "push.delivery.dlq"
)

var DeliveryTopics = []string{
	DeliveryTasksTopic,
	Retry1mTopic,
	Retry10mTopic,
	Retry1hTopic,
	DLQTopic,
}

type Config struct {
	Brokers []string
}

type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(ctx context.Context, cfg Config) (*Producer, error) {
	brokers := normalizedBrokers(cfg.Brokers)
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("create redpanda client: %w", err)
	}
	producer := &Producer{client: client, topic: DeliveryTasksTopic}
	if err := producer.EnsureTopics(ctx); err != nil {
		client.Close()
		return nil, err
	}
	return producer, nil
}

func (p *Producer) Close() {
	p.client.Close()
}

func (p *Producer) EnsureTopics(ctx context.Context) error {
	admin := kadm.NewClient(p.client)
	results, err := admin.CreateTopics(
		ctx,
		12,
		3,
		nil,
		DeliveryTopics...,
	)
	if err != nil {
		return fmt.Errorf("create delivery topics: %w", err)
	}
	for topic, result := range results {
		if result.Err == nil || errors.Is(result.Err, kerr.TopicAlreadyExists) {
			continue
		}
		return fmt.Errorf(
			"create delivery topic %s: %w",
			topic,
			result.Err,
		)
	}
	return nil
}

func (p *Producer) ProduceDeliveryTasks(ctx context.Context, tasks []inventory.DeliveryTask) error {
	for _, task := range tasks {
		if err := p.ProduceRetryTask(
			ctx,
			p.topic,
			task,
		); err != nil {
			return err
		}
	}
	return nil
}

func (p *Producer) ProduceRetryTask(
	ctx context.Context,
	topic string,
	task inventory.DeliveryTask,
) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal delivery task: %w", err)
	}
	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(task.SubscriptionID),
		Value: payload,
	}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce delivery task: %w", err)
	}
	return nil
}

type Consumer struct {
	client *kgo.Client
}

func NewConsumer(
	cfg Config,
	group string,
	topics []string,
) (*Consumer, error) {
	brokers := normalizedBrokers(cfg.Brokers)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create redpanda consumer: %w", err)
	}
	return &Consumer{client: client}, nil
}

func (c *Consumer) Close() {
	c.client.Close()
}

func (c *Consumer) Run(
	ctx context.Context,
	handler func(context.Context, *kgo.Record) error,
) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fetchErr := range errs {
				if errors.Is(fetchErr.Err, context.Canceled) ||
					errors.Is(fetchErr.Err, context.DeadlineExceeded) {
					return ctx.Err()
				}
				return fmt.Errorf("poll redpanda fetches: %w", fetchErr.Err)
			}
		}
		records := fetches.Records()
		for _, record := range records {
			if err := handler(ctx, record); err != nil {
				return err
			}
			if err := c.client.CommitRecords(ctx, record); err != nil {
				return fmt.Errorf("commit delivery task: %w", err)
			}
		}
	}
}

func normalizedBrokers(brokers []string) []string {
	normalized := []string{}
	seen := map[string]bool{}
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" || seen[broker] {
			continue
		}
		seen[broker] = true
		normalized = append(normalized, broker)
	}
	if len(normalized) == 0 {
		return []string{"localhost:19092"}
	}
	return normalized
}
