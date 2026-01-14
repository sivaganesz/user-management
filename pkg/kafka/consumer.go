// package kafka

// import (
// 	"fmt"

// 	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
// 	"github.com/white/user-management/config"
// )

// // Consumer wraps a Kafka consumer
// type Consumer struct {
// 	consumer *kafka.Consumer
// 	config   config.KafkaConfig
// }

// // MessageHandler is a function that processes Kafka messages
// type MessageHandler func(message *kafka.Message) error

// // NewConsumer creates a new Kafka consumer
// func NewConsumer(cfg config.KafkaConfig) (*Consumer, error) {
// 	configMap := &kafka.ConfigMap{
// 		"bootstrap.servers": joinStringSlice(cfg.Brokers, ","),
// 		"group.id":          cfg.ConsumerGroup,
// 		"auto.offset.reset": "earliest",
// 	}

// 	consumer, err := kafka.NewConsumer(configMap)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
// 	}

// 	return &Consumer{
// 		consumer: consumer,
// 		config:   cfg,
// 	}, nil
// }

// // Subscribe subscribes to Kafka topics
// func (c *Consumer) Subscribe(topics []string) error {
// 	return c.consumer.SubscribeTopics(topics, nil)
// }

// // Consume starts consuming messages and calls the handler for each message
// func (c *Consumer) Consume(handler MessageHandler) error {
// 	for {
// 		msg, err := c.consumer.ReadMessage(-1)
// 		if err != nil {
// 			return fmt.Errorf("error reading message: %w", err)
// 		}

// 		if err := handler(msg); err != nil {
// 			fmt.Printf("error processing message: %v\n", err)
// 			continue
// 		}

// 		_, err = c.consumer.CommitMessage(msg)
// 		if err != nil {
// 			fmt.Printf("error committing message: %v\n", err)
// 		}
// 	}
// }

// // Close closes the Kafka consumer
// func (c *Consumer) Close() {
// 	if c.consumer != nil {
// 		_ = c.consumer.Close()
// 	}
// }

package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/white/user-management/config"
)

// Consumer wraps a kafka-go reader
type Consumer struct {
	reader *kafka.Reader
	config config.KafkaConfig
}

// MessageHandler processes Kafka messages
type MessageHandler func(message kafka.Message) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig, topic string) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("no Kafka brokers provided")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.ConsumerGroup,
		Topic:       topic,
		StartOffset: kafka.FirstOffset,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})

	return &Consumer{
		reader: reader,
		config: cfg,
	}, nil
}

// Consume reads messages and calls the handler
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("error reading message: %w", err)
		}

		if err := handler(msg); err != nil {
			fmt.Printf("error processing message: %v\n", err)
		}
	}
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
