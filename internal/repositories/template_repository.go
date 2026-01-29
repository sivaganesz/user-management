package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/uuid"
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
func (r *MongoTemplateRepository) GetByID(ctx context.Context, id string) (*models.MongoTemplate, error) {
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
	_ , err := r.collection.InsertOne(ctx, template)
	if err != nil {
		return fmt.Errorf("error creating template: %w", err)
	}

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
func (r *MongoTemplateRepository) Delete(ctx context.Context, id string) error {
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


// =============================================================================
// Service layer compatibility methods
// =============================================================================

// GetByIDCompat retrieves a sequence template by ID with its steps
func (r *MongoTemplateRepository) GetByIDCompat(templateID primitive.ObjectID) (*models.SequenceTemplateWithSteps, error) {
	ctx := context.Background()

	var template models.SequenceTemplateWithSteps
	filter := bson.M{"_id": templateID}

	err := r.sequenceCollection.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
		}
		return nil, fmt.Errorf("error finding sequence template by ID: %w", err)
	}

	return &template, nil
}

// CreateSequenceTemplate creates a sequence template with steps
func (r *MongoTemplateRepository) CreateSequenceTemplate(template *models.SequenceTemplateWithSteps) error {
	ctx := context.Background()

	// Generate new ID if not set
	if template.Template.TemplateID == "" {
		template.Template.TemplateID = uuid.MustNewUUID()
	}

	// Set timestamps
	now := time.Now()
	template.Template.CreatedAt = now
	template.Template.UpdatedAt = now

	// Set step timestamps and template IDs
	for i := range template.Steps {
		template.Steps[i].TemplateID = template.Template.TemplateID
		template.Steps[i].CreatedAt = now
		// Ensure step_order is set correctly
		if template.Steps[i].StepOrder == 0 {
			template.Steps[i].StepOrder = i + 1
		}
	}

	// Insert into sequence_templates collection
	_, err := r.sequenceCollection.InsertOne(ctx, template)
	if err != nil {
		return fmt.Errorf("error creating sequence template: %w", err)
	}

	return nil
}

// List lists sequence templates with filters
func (r *MongoTemplateRepository) List(filters SequenceTemplateFilters) ([]*models.SequenceTemplateWithSteps, error) {
	ctx := context.Background()

	filter := bson.M{}

	if filters.Channel != "" {
		filter["steps.channel"] = filters.Channel
	}
	if filters.IsActive != nil {
		filter["is_active"] = *filters.IsActive
	}
	if filters.Search != "" {
		filter["name"] = bson.M{"$regex": filters.Search, "$options": "i"}
	}

	limit := filters.Limit
	if limit == 0 {
		limit = 20
	}

	cursor, err := r.sequenceCollection.Find(ctx, filter, options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(filters.Offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("error listing sequence templates: %w", err)
	}
	defer cursor.Close(ctx)

	var templates []*models.SequenceTemplateWithSteps
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, fmt.Errorf("error decoding sequence templates: %w", err)
	}

	return templates, nil
}

// GetStepCount returns the number of steps for a template
func (r *MongoTemplateRepository) GetStepCount(templateID primitive.ObjectID) (int, error) {
	template, err := r.GetByIDCompat(templateID)
	if err != nil {
		return 0, err
	}
	return len(template.Steps), nil
}

// UpdateSequenceTemplate updates a sequence template with its steps
func (r *MongoTemplateRepository) UpdateSequenceTemplate(template *models.SequenceTemplateWithSteps) error {
	ctx := context.Background()

	// Update timestamp
	template.Template.UpdatedAt = time.Now()

	// Update step timestamps and template IDs
	for i := range template.Steps {
		template.Steps[i].TemplateID = template.Template.TemplateID
		if template.Steps[i].StepOrder == 0 {
			template.Steps[i].StepOrder = i + 1
		}
	}

	filter := bson.M{"_id": template.Template.TemplateID}
	update := bson.M{
		"$set": bson.M{
			"name":        template.Template.Name,
			"description": template.Template.Description,
			"schedule_id": template.Template.ScheduleID,
			"version":     template.Template.Version,
			"is_active":   template.Template.IsActive,
			"updated_at":  template.Template.UpdatedAt,
			"steps":       template.Steps,
		},
	}

	result, err := r.sequenceCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating sequence template: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
	}

	return nil
}

// DeleteSequenceTemplate deletes a sequence template
func (r *MongoTemplateRepository) DeleteSequenceTemplate(templateID primitive.ObjectID) error {
	ctx := context.Background()

	result, err := r.sequenceCollection.DeleteOne(ctx, bson.M{"_id": templateID})
	if err != nil {
		return fmt.Errorf("error deleting sequence template: %w", err)
	}

	if result.DeletedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrTemplateNotFound)
	}

	return nil
}

// Clone clones a sequence template
func (r *MongoTemplateRepository) Clone(templateID primitive.ObjectID, newName string, createdBy string) (*models.SequenceTemplateWithSteps, error) {
	// Get the original template
	original, err := r.GetByIDCompat(templateID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newID := uuid.MustNewUUID()

	// Create a clone with new ID
	clone := &models.SequenceTemplateWithSteps{
		Template: models.SequenceTemplate{
			TemplateID:  newID,
			Name:        newName,
			Description: original.Template.Description,
			ScheduleID:  original.Template.ScheduleID,
			Version:     1,
			IsActive:    true,
			CreatedBy:   createdBy,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Steps: make([]models.CampaignSequenceStep, len(original.Steps)),
	}

	// Copy and update step IDs for the clone
	for i, step := range original.Steps {
		clone.Steps[i] = step
		clone.Steps[i].TemplateID = newID
		clone.Steps[i].CreatedAt = now
	}

	// Save the clone
	if err := r.CreateSequenceTemplate(clone); err != nil {
		return nil, err
	}

	return clone, nil
}

// ValidateSequence validates a sequence template and returns validation errors
func (r *MongoTemplateRepository) ValidateSequence(templateID primitive.ObjectID) (bool, []map[string]interface{}, error) {
	// Get template with steps
	template, err := r.GetByIDCompat(templateID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return false, nil, fmt.Errorf("template not found")
	}

	// Validate using model validation
	validationErr := template.Validate()
	if validationErr == nil {
		return true, []map[string]interface{}{}, nil
	}

	// Build error list
	errors := []map[string]interface{}{
		{
			"stepOrder": 0,
			"field":     "general",
			"message":   validationErr.Error(),
		},
	}

	// Additional validations
	for i, step := range template.Steps {
		stepOrder := i + 1

		// Check required fields
		if step.Channel == "" {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "communicationType",
				"message":   "Communication type is required",
			})
		}

		if step.ContentTemplateID == "" {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "templateId",
				"message":   "Template is required",
			})
		}

		if step.Body == "" {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "message",
				"message":   "Message content is required",
			})
		}

		// Email requires subject
		if step.Channel == "email" && step.Subject == "" {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "subject",
				"message":   "Subject is required for email steps",
			})
		}

		// First step should have 0 wait days
		if stepOrder == 1 && step.WaitDays != 0 {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "waitDays",
				"message":   "First step should have 0 wait days",
			})
		}

		// Check wait days is non-negative
		if step.WaitDays < 0 {
			errors = append(errors, map[string]interface{}{
				"stepOrder": stepOrder,
				"field":     "waitDays",
				"message":   "Wait days cannot be negative",
			})
		}
	}

	isValid := len(errors) == 0
	return isValid, errors, nil
}

// =============================================================================
// Template Handler Compatibility Methods
// =============================================================================

// CreateTemplateCompat creates a template (no context)
func (r *MongoTemplateRepository) CreateTemplateCompat(template *models.MongoTemplate) error {
	return r.Create(context.Background(), template)
}

// UpdateTemplateCompat updates a template (no context)
func (r *MongoTemplateRepository) UpdateTemplateCompat(template *models.MongoTemplate) error {
	template.UpdatedAt = time.Now()
	return r.Update(context.Background(), template)
}

// GetTemplateByIDCompat retrieves a template by ID (no context)
func (r *MongoTemplateRepository) GetTemplateByIDCompat(tenantID, templateID string) (*models.MongoTemplate, error) {
	return r.GetByID(context.Background(), templateID)
}

// DeleteTemplateCompat deletes a template (no context) - stub
func (r *MongoTemplateRepository) DeleteTemplateCompat(tenantID, templateID string) error {
	return r.Delete(context.Background(), templateID)
}

// RemoveTagFromTemplateCompat removes a tag from all templates - stub
func (r *MongoTemplateRepository) RemoveTagFromTemplateCompat(tenantID string, tag string) error {
	// Stub - would need to iterate and update templates
	return nil
}
