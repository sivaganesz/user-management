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

// UpdatePassword updates the user's password hash
func (r *UserRepository) UpdatePassword(ctx context.Context, id primitive.ObjectID, passwordHash string) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"password_hash": passwordHash,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating password %w",err)
	}
	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}
	return nil
}

// SetOTP sets the OTP hash and expiry time for password reset
func (r *UserRepository) SetOTP(ctx context.Context, userID primitive.ObjectID, otpHash string, expiresAt time.Time) error {
	filter := bson.M{"_id":userID}
	update := bson.M{
		"$set": bson.M{
			"otp_hash":       otpHash,
			"otp_expires_at":   expiresAt,
			"updated_at":     time.Now(),
		},
	}
	
	result, err := r.collection.UpdateOne(ctx,filter, update)
	if err != nil {
		return fmt.Errorf("error setting OTP: %w", err)
	}
	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

func (r *UserRepository) ClearOTP(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$unset": bson.M{
			"otp_hash":       "",
			"otp_expires_at":   "",
		},
		"$set": bson.M{
			"updated_at":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error clearing OTP: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// ListByRegion retrieves users in a specific region with pagination
func (r *UserRepository) ListByRegion(ctx context.Context, region string, limit, offset int) ([]*models.User, error) {
	filter := bson.M{"region": region}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing users by region: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// ListByRole retrieves users with a specific role with pagination
func (r *UserRepository) ListByRole(ctx context.Context, role models.UserRole, limit, offset int) ([]*models.User, error) {
	filter := bson.M{"role": role}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}}) // Sort by name ascending

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing users by role: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// ListByTeam retrieves users in a specific team with pagination
func (r *UserRepository) ListByTeam(ctx context.Context, team string, limit, offset int) ([]*models.User, error) {
	filter := bson.M{"team": team}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing users by team: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// Delete removes a user by ID (soft delete recommended in production)
func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	if result.DeletedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// Count returns the total number of users (optionally filtered by region)
func (r *UserRepository) Count(ctx context.Context, region string) (int64, error) {
	filter := bson.M{}
	if region != "" {
		filter["region"] = region
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}

	return count, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID primitive.ObjectID, loginTime time.Time) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"lastLoginAt": loginTime,
			"updated_at":  time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating last login: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// ActivateUser activates a user account
func (r *UserRepository) ActivateUser(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"isActive":   true,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error activating user: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// DeactivateUser deactivates a user account
func (r *UserRepository) DeactivateUser(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"isActive":   false,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error deactivating user: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// GetAllUsers retrieves all users with pagination (from UserManagementRepository)
func (r *UserRepository) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing all users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// GetUsersByRole retrieves users by role (from UserManagementRepository)
func (r *UserRepository) GetUsersByRole(ctx context.Context, role models.UserRole, limit, offset int) ([]*models.User, error) {
	return r.ListByRole(ctx, role, limit, offset)
}

// GetUsersByTeam retrieves users by team (from UserManagementRepository)
func (r *UserRepository) GetUsersByTeam(ctx context.Context, team string, limit, offset int) ([]*models.User, error) {
	return r.ListByTeam(ctx, team, limit, offset)
}

// UpdateUserProfile updates user profile information (name, email) from UserSettingsRepository
func (r *UserRepository) UpdateUserProfile(ctx context.Context, userID primitive.ObjectID, name, email string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"name":       name,
			"email":      email,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating user profile: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

// EnsureIndexes creates the required indexes for the users collection
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "region", Value: 1},
				{Key: "role", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "team", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "isActive", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("error creating indexes: %w", err)
	}
	
	return nil
}
