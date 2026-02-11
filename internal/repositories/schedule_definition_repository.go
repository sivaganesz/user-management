package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ScheduleDefinitionRepository handles schedule definition persistence with MongoDB
type ScheduleDefinitionRepository struct {
	client     *mongodb.Client
	collection *mongo.Collection
}

// NewScheduleDefinitionRepository creates a new ScheduleDefinitionRepository
func NewScheduleDefinitionRepository(client *mongodb.Client) *ScheduleDefinitionRepository {
	return &ScheduleDefinitionRepository{
		client:     client,
		collection: client.Collection("schedule_definitions"),
	}
}

// CreateScheduleDefinition creates a new schedule definition
func (r *ScheduleDefinitionRepository) CreateScheduleDefinition(schedule *models.ScheduleDefinition) error {
	if schedule == nil {
		return fmt.Errorf("schedule definition cannot be nil")
	}

	// Generate new ID if not set
	if uuid.IsEmptyUUID(schedule.ID) {
		schedule.ID = uuid.MustNewUUID()
	}

	// Set timestamps
	now := time.Now()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}
	schedule.UpdatedAt = now

	// Set defaults
	if schedule.TimeZone == "" {
		schedule.TimeZone = "UTC"
	}

	_, err := r.collection.InsertOne(context.Background(), schedule)
	if err != nil {
		return fmt.Errorf("error creating schedule definition: %w", err)
	}

	return nil
}

// GetScheduleDefinitionByID retrieves a schedule definition by ID
func (r *ScheduleDefinitionRepository) GetScheduleDefinitionByID(id string) (*models.ScheduleDefinition, error) {
	if uuid.IsEmptyUUID(id) {
		return nil, fmt.Errorf("schedule definition ID is required")
	}

	var schedule models.ScheduleDefinition

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(context.Background(), filter).Decode(&schedule)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("schedule definition not found")
		}
		return nil, fmt.Errorf("error finding schedule definition: %w", err)
	}

	return &schedule, nil
}

// GetAllScheduleDefinitions retrieves all schedule definitions
func (r *ScheduleDefinitionRepository) GetAllScheduleDefinitions() ([]*models.ScheduleDefinition, error) {
	filter := bson.M{}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding schedule definitions: %w", err)
	}
	defer cursor.Close(context.Background())
	var schedules []*models.ScheduleDefinition
	if err = cursor.All(context.Background(), &schedules); err != nil {
		return nil, fmt.Errorf("error decoding schedule definitions: %w", err)
	}

	return schedules, nil
}

// GetActiveScheduleDefinitions retrieves active schedule definitions
func (r *ScheduleDefinitionRepository) GetActiveScheduleDefinitions() ([]models.ScheduleDefinition, error) {
	filter := bson.M{"is_active": true}

	// Sort by name
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding active schedule definitions: %w", err)
	}
	defer cursor.Close(context.Background())

	var schedules []models.ScheduleDefinition
	if err := cursor.All(context.Background(), &schedules); err != nil {
		return nil, fmt.Errorf("error decoding schedule definitions: %w", err)
	}

	return schedules, nil
}

// UpdateScheduleDefinition updates an existing schedule definition
func (r *ScheduleDefinitionRepository) UpdateScheduleDefinition(schedule *models.ScheduleDefinition) error {
	if schedule == nil {
		return fmt.Errorf("schedule definition cannot be nil")
	}

	if uuid.IsEmptyUUID(schedule.ID) {
		return fmt.Errorf("schedule definition ID is required")
	}

	filter := bson.M{"_id": schedule.ID}
	update := bson.M{
		"$set": bson.M{
			"name":                 schedule.Name,
			"description":          schedule.Description,
			"frequency":            schedule.Frequency,
			"day_of_week":          schedule.DayOfWeek,
			"day_of_month":         schedule.DayOfMonth,
			"time":                 schedule.Time,
			"timezone":             schedule.TimeZone,
			"use_contact_timezone": schedule.UseContactTimeZone,
			"excluded_holidays":    schedule.ExcludedHolidays,
			"sending_windows":      schedule.SendingWindows,
			"is_active":            schedule.IsActive,
			"updated_at":           schedule.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("error updating schedule definition: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("schedule definition not found")
	}
	return nil
}

// DeleteScheduleDefinition deletes a schedule definition by ID
func (r *ScheduleDefinitionRepository) DeleteScheduleDefinition(id string) error {
	if uuid.IsEmptyUUID(id) {
		return fmt.Errorf("schedule definition ID is required")
	}
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("error deleting schedule definition: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("schedule definition not found")
	}
	return nil
}

// Count return the total number of schedule definitions
func (r *ScheduleDefinitionRepository) Count(activeOnly bool) (int64, error) {

	filter := bson.M{}
	if activeOnly {
		filter["is_active"] = true
	}

	count, err := r.collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return 0, fmt.Errorf("error counting schedule definitions: %w", err)
	}
	return count, nil
}

// EnsureIndex creates the required indexes for the schedule definitions collection
func (r *ScheduleDefinitionRepository) EnsureIndex() error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "is_active", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "created_by", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
		},
	}
	_, err := r.collection.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return fmt.Errorf("error creating indexes: %w", err)
	}

	return nil
}
