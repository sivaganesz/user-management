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
)

type UserRepository struct {
	client 		*mongodb.Client
	collection  *mongo.Collection
}

func NewUserRepository(client *mongodb.Client) *UserRepository {
	return &UserRepository{
		client:     client,
		collection: client.Collection("users"),
	}
}

func (r *UserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user by their email address
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	filter := bson.M{"email": email}
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
		}
		return nil, fmt.Errorf("error finding user by email: %w", err)
	}

	return &user, nil
}

// Create inserts a new user document
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := r.collection.InsertOne(ctx, user);
	if err != nil {
		 if mongo.IsDuplicateKeyError(err){
			return fmt.Errorf("user with email %s already exists", user.Email);
		 }
		return fmt.Errorf("error creating user: %w", err)
	}
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// Update modifies an existing user document
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	// Update timestamp
	user.UpdatedAt = time.Now()

	filter := bson.M{"_id": user.ID}
	update := bson.M{
		"$set": bson.M{
			"email":          user.Email,
			"name":           user.Name,
			"role":           user.Role,
			"region":         user.Region,
			"team":           user.Team,
			"permissions":    user.Permissions,
			"preferences":    user.Preferences,
			"emailSignature": user.EmailSignature,
			"updated_at":     user.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

 