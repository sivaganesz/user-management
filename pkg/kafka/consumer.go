package kafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/white/user-management/config"
)

// Consumer wraps a Kafka consumer
type Consumer struct {
	consumer *kafka.Consumer
	config   config.KafkaConfig
}

// MessageHandler is a function that processes Kafka messages
type MessageHandler func(message *kafka.Message) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig) (*Consumer, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": joinStringSlice(cfg.Brokers, ","),
		"group.id":          cfg.ConsumerGroup,
		"auto.offset.reset": "earliest",
	}

	consumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		config:   cfg,
	}, nil
}

// Subscribe subscribes to Kafka topics
func (c *Consumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// Consume starts consuming messages and calls the handler for each message
func (c *Consumer) Consume(handler MessageHandler) error {
	for {
		msg, err := c.consumer.ReadMessage(-1)
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}

		if err := handler(msg); err != nil {
			fmt.Printf("error processing message: %v\n", err)
			continue
		}

		_, err = c.consumer.CommitMessage(msg)
		if err != nil {
			fmt.Printf("error committing message: %v\n", err)
		}
	}
}

// Close closes the Kafka consumer
func (c *Consumer) Close() {
	if c.consumer != nil {
		_ = c.consumer.Close()
	}
}
