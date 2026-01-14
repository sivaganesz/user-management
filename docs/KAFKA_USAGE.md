# Kafka Producer - Usage Guide

## Overview

The refactored Kafka producer is now **topic-agnostic** and **simple to use**. You create a **single producer instance** and can publish to **any topic** without needing to create separate producers.

## Setup

### In `main.go`

```go
import "github.com/white/user-management/pkg/kafka"

// Initialize producer once at startup
kafkaProducer, err := kafka.NewProducer(kafkaConfig)
if err != nil {
    log.Printf("Warning: Kafka producer not available: %v", err)
    kafkaProducer = nil
}
defer kafkaProducer.Close()

// Pass to handlers
authHandler := handlers.NewAuthHandler(db, config, kafkaProducer)
```

## Publishing Events

### Simple Usage - Publish JSON to Any Topic

```go
// In your handler
event := map[string]interface{}{
    "event_id": "123",
    "user_id": "user123",
    "action": "logged_in",
    "timestamp": time.Now().Unix(),
}

ctx := context.Background()
err := h.producer.PublishJSON(ctx, "users.logged_in", event)
if err != nil {
    log.Printf("Failed to publish event: %v", err)
}
```

### Raw Byte Message

```go
// Publish raw bytes if needed
key := []byte("user123")
value := []byte(`{"action":"login"}`)

err := h.producer.Produce(ctx, "users.logged_in", key, value)
```

## Multiple Topics Example

```go
// All use the same producer instance!

// Topic 1: User events
h.producer.PublishJSON(ctx, "users.logged_in", loginEvent)
h.producer.PublishJSON(ctx, "users.logged_out", logoutEvent)
h.producer.PublishJSON(ctx, "users.created", creationEvent)

// Topic 2: Password events
h.producer.PublishJSON(ctx, "passwords.reset", resetEvent)
h.producer.PublishJSON(ctx, "passwords.changed", changeEvent)

// Topic 3: Team events
h.producer.PublishJSON(ctx, "teams.created", teamEvent)
h.producer.PublishJSON(ctx, "teams.updated", updateEvent)

// Topic 4: Audit events
h.producer.PublishJSON(ctx, "audit.action", auditEvent)
```

## API Reference

### `NewProducer(config KafkaConfig) (*Producer, error)`
- Creates a single producer for all topics
- `config`: Kafka configuration with brokers and credentials

### `PublishJSON(ctx context.Context, topic string, data interface{}) error`
- Marshals data to JSON and publishes to the specified topic
- **Most common method** for publishing events
- Returns error if marshal or publish fails

### `Produce(ctx context.Context, topic string, key, value []byte) error`
- Low-level method for publishing raw bytes
- Useful if you need custom serialization

### `Close() error`
- Cleanup (mostly no-op with kafka-go, but kept for consistency)

## Environment Variables

```bash
KAFKA_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
KAFKA_USERNAME=your-username      # Optional
KAFKA_PASSWORD=your-password      # Optional
KAFKA_SASL_MECHANISM=plain         # Optional
KAFKA_SSL=true                     # Optional
```

## Error Handling

Always check for errors, but don't fail requests because of Kafka:

```go
go func() {
    if err := h.producer.PublishJSON(ctx, "users.logged_in", event); err != nil {
        // Log but don't fail the login
        log.Printf("Warning: Failed to publish event: %v", err)
    }
}()
```

## Benefits

✅ **Single producer instance** - No per-topic producer overhead  
✅ **Any topic** - Add new topics without code changes  
✅ **Simple API** - Just `PublishJSON(ctx, topic, data)`  
✅ **Scalable** - Publish to hundreds of topics easily  
✅ **Fire-and-forget** - Use goroutines for async publishing  

## Best Practices

1. **Use async publishing** for non-critical events:
   ```go
   go h.producer.PublishJSON(ctx, "topic", event)
   ```

2. **Include timestamps** in your events:
   ```go
   "timestamp": time.Now().Unix()
   ```

3. **Use consistent event structure**:
   ```go
   event := map[string]interface{}{
       "event_id": uuid.New(),
       "event_type": "user.login",
       "timestamp": time.Now().Unix(),
       "data": {...},
   }
   ```

4. **Topic naming convention**:
   - `entity.action` format: `users.logged_in`, `passwords.reset`, `teams.created`
