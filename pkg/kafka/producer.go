package kafka

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/white/user-management/config"
)

// Producer wraps a Kafka producer
type Producer struct {
	producer *kafka.Producer
	config   config.KafkaConfig
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": joinStringSlice(cfg.Brokers, ","),
		"client.id":         cfg.ClientID,
		"acks":              "all",
	}

	if cfg.Username != "" && cfg.Password != "" {
		saslMechanism := strings.ToUpper(cfg.SASLMechanism)

		configMap.SetKey("sasl.mechanism", saslMechanism)
		configMap.SetKey("sasl.username", cfg.Username)
		configMap.SetKey("sasl.password", cfg.Password)

		if cfg.SSL {
			configMap.SetKey("security.protocol", "SASL_SSL")
		} else {
			configMap.SetKey("security.protocol", "SASL_PLAINTEXT")
		}
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("delivery failed: %v\n", ev.TopicPartition.Error)
				}
			}
		}
	}()

	return &Producer{
		producer: producer,
		config:   cfg,
	}, nil
}

// Produce sends a message to a Kafka topic (async)
func (p *Producer) Produce(topic string, key, value []byte) error {
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}

	return p.producer.Produce(message, nil)
}

// ProduceSync sends a message and waits for delivery confirmation
func (p *Producer) ProduceSync(topic string, key, value []byte) error {
	deliveryChan := make(chan kafka.Event, 1)

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}

	if err := p.producer.Produce(message, deliveryChan); err != nil {
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	return nil
}

// PublishJSON marshals data to JSON and publishes it
func (p *Producer) PublishJSON(topic string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return p.Produce(topic, nil, jsonData)
}

// Flush waits for all messages to be delivered
func (p *Producer) Flush(timeoutMs int) {
	p.producer.Flush(timeoutMs)
}

// Close closes the Kafka producer
func (p *Producer) Close() {
	if p.producer != nil {
		p.producer.Flush(5000)
		p.producer.Close()
	}
}

// joinStringSlice joins a string slice with a separator
func joinStringSlice(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}
