package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TemplateCache provides Redis caching for templates
type TemplateCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewTemplateCache creates a new template cache with 15-minute TTL
func NewTemplateCache(client *redis.Client) *TemplateCache {
	return &TemplateCache{
		client: client,
		ttl:    15 * time.Minute, // 900 seconds as per spec
	}
}

// Get retrieves a template from cache
// Returns error if cache miss or deserialization fails
func (c *TemplateCache) Get(tenantID, templateID primitive.ObjectID) (*models.MongoTemplate, error) {
	ctx := context.Background()
	key := c.buildKey(tenantID, templateID)

	// Get value from Redis
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss: template not found in cache")
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	// Deserialize JSON
	var template models.MongoTemplate
	if err := json.Unmarshal([]byte(val), &template); err != nil {
		return nil, fmt.Errorf("failed to deserialize template: %w", err)
	}

	return &template, nil
}

// Set stores a template in cache with TTL
// Serializes template to JSON and stores in Redis
func (c *TemplateCache) Set(template *models.MongoTemplate) error {
	ctx := context.Background()
	key := c.buildKey(template.TenantID, template.ID)

	// Serialize template to JSON
	data, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to serialize template: %w", err)
	}

	// Store in Redis with TTL
	err = c.client.Set(ctx, key, data, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a template from cache
// Used for cache invalidation on update/delete/publish/unpublish
func (c *TemplateCache) Delete(tenantID, templateID primitive.ObjectID) error {
	ctx := context.Background()
	key := c.buildKey(tenantID, templateID)

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// buildKey creates the Redis key for a template
// Format: template:{tenant_id}:{template_id}
func (c *TemplateCache) buildKey(tenantID, templateID primitive.ObjectID) string {
	return fmt.Sprintf("template:%s:%s", tenantID.String(), templateID.String())
}
