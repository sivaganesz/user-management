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
