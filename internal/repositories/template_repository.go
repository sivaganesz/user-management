package repositories
import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoTemplateRepository handles template data access with MongoDB
type MongoTemplateRepository struct {
	client             *mongodb.Client
	collection         *mongo.Collection
	sequenceCollection *mongo.Collection
}

// NewMongoTemplateRepository creates a new MongoTemplateRepository
func NewMongoTemplateRepository(client *mongodb.Client) *MongoTemplateRepository {
	return &MongoTemplateRepository{
		client:             client,
		collection:         client.Collection("templates"),
		sequenceCollection: client.Collection("sequence_templates"),
	}
}

// GetByID retrieves a template by its ObjectID
func (r *MongoTemplateRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.MongoTemplate, error) {
	var template models.MongoTemplate

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
		}
		return nil, fmt.Errorf("error finding template by ID: %w", err)
	}

	return &template, nil
}

// Create inserts a new template document
func (r *MongoTemplateRepository) Create(ctx context.Context, template *models.MongoTemplate) error {
	// Set timestamp
	template.CreatedAt = time.Now()

	// Ensure channel matches type for consistency
	if template.Channel == "" {
		template.Channel = template.Type
	}

	// Insert the document
	result, err := r.collection.InsertOne(ctx, template)
	if err != nil {
		return fmt.Errorf("error creating template: %w", err)
	}

	// Set the generated ID
	template.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// Update modifies an existing template document
func (r *MongoTemplateRepository) Update(ctx context.Context, template *models.MongoTemplate) error {
	filter := bson.M{"_id": template.ID}
	update := bson.M{
		"$set": bson.M{
			"name":          template.Name,
			"type":          template.Type,
			"channel":       template.Channel,
			"subject":       template.Subject,
			"body":          template.Body,
			"variables":     template.Variables,
			"category":      template.Category,
			"tags":          template.Tags,
			"status":        template.Status,
			"for_stage":     template.ForStage,
			"industries":    template.Industries,
			"approval_flag": template.ApprovalFlag,
			"ai_enhanced":   template.AiEnhanced,
			"service_id":    template.ServiceID,
			"updated_at":    time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating template: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
	}

	return nil
}

// Delete removes a template by ID
func (r *MongoTemplateRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting template: %w", err)
	}

	if result.DeletedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
	}

	return nil
}

// ListByType retrieves templates of a specific type with pagination
func (r *MongoTemplateRepository) ListByType(ctx context.Context, templateType string, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{"type": templateType}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}}) // Sort alphabetically by name

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing templates by type: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// ListByChannel retrieves templates for a specific channel with pagination
func (r *MongoTemplateRepository) ListByChannel(ctx context.Context, channel string, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{"channel": channel}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing templates by channel: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// ListByCategory retrieves templates in a specific category with pagination
func (r *MongoTemplateRepository) ListByCategory(ctx context.Context, category string, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{"category": category}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing templates by category: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// SearchByTags retrieves templates that have any of the specified tags
func (r *MongoTemplateRepository) SearchByTags(ctx context.Context, tags []string, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{}
	if len(tags) > 0 {
		filter["tags"] = bson.M{"$in": tags}
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error searching templates by tags: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// SearchByName searches for templates by name using regex
func (r *MongoTemplateRepository) SearchByName(ctx context.Context, searchTerm string, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{
		"name": bson.M{
			"$regex":   searchTerm,
			"$options": "i", // Case-insensitive
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error searching templates by name: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// ListAll retrieves all templates with pagination
func (r *MongoTemplateRepository) ListAll(ctx context.Context, limit, offset int) ([]*models.MongoTemplate, error) {
	filter := bson.M{}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by newest first

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing all templates: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.MongoTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding templates: %w", err)
	}

	return templates, nil
}

// Count returns the total number of templates (optionally filtered by type)
func (r *MongoTemplateRepository) Count(ctx context.Context, templateType string) (int64, error) {
	filter := bson.M{}
	if templateType != "" {
		filter["type"] = templateType
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error counting templates: %w", err)
	}

	return count, nil
}

// EnsureIndexes creates the required indexes for the templates collection
func (r *MongoTemplateRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "channel", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "tags", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("error creating indexes: %w", err)
	}

	return nil
}
