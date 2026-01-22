package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoActivityRepository handles activity/audit log data access with MongoDB
type MongoActivityRepository struct {
	client             *mongodb.Client
	collection         *mongo.Collection
	tasksCollection    *mongo.Collection
	eventsCollection   *mongo.Collection
	timelineCollection *mongo.Collection
}

// NewMongoActivityRepository creates a new MongoActivityRepository
func NewMongoActivityRepository(client *mongodb.Client) *MongoActivityRepository {
	return &MongoActivityRepository{
		client:             client,
		collection:         client.Collection("activities"),
		tasksCollection:    client.Collection("tasks"),
		eventsCollection:   client.Collection("calendar_events"),
		timelineCollection: client.Collection("activity_timeline"),
	}
}

// ===================================================================
// Extended Methods for Full Activity Management (Tasks, Events, etc.)
// ===================================================================

// CreateActivity creates a new activity (generic activity entity)
func (r *MongoActivityRepository) CreateActivity(ctx context.Context, activity *models.Activity) error {
	activity.CreatedAt = time.Now()
	activity.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, activity)
	if err != nil {
		return fmt.Errorf("error creating activity: %w", err)
	}

	if id, ok := result.InsertedID.(string); ok {
		activity.ID = id
	}

	return nil
}


// CreateActivityCompat creates a new activity using context.Background() for backward compatibility
// This method is provided for code that doesn't have access to a context
func (r *MongoActivityRepository) CreateActivityCompat(activity *models.Activity) error {
	return r.CreateActivity(context.Background(), activity)
}


// GetActivityByID retrieves an activity by its ObjectID (excludes soft-deleted)
func (r *MongoActivityRepository) GetActivityByID(ctx context.Context, id string) (*models.Activity, error) {
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	var activity models.Activity
	err := r.collection.FindOne(ctx, filter).Decode(&activity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("activity not found")
		}
		return nil, fmt.Errorf("error querying activity: %w", err)
	}

	return &activity, nil
}

// UpdateActivity updates an activity
func (r *MongoActivityRepository) UpdateActivity(ctx context.Context, activity *models.Activity) error {
	activity.UpdatedAt = time.Now()

	filter := bson.M{"_id": activity.ID}
	update := bson.M{"$set": activity}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating activity: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("activity not found")
	}

	return nil
}

// DeleteActivity deletes an activity by ID
func (r *MongoActivityRepository) DeleteActivity(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting activity: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("activity not found")
	}

	return nil
}

// GetActivitiesByOwner retrieves activities for a specific owner (excludes soft-deleted)
func (r *MongoActivityRepository) GetActivitiesByOwner(ctx context.Context, ownerID string, limit int) ([]*models.Activity, error) {
	filter := bson.M{
		"owner":      ownerID,
		"deleted_at": nil,
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing activities by owner: %w", err)
	}
	defer cursor.Close(ctx)

	var activities []*models.Activity
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, fmt.Errorf("error decoding activities: %w", err)
	}

	return activities, nil
}
