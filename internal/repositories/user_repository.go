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

type MongoUserRepository struct {
	client     *mongodb.Client
	collection *mongo.Collection
}

func NewMongoUserRepository(client *mongodb.Client) *MongoUserRepository {
	return &MongoUserRepository{
		client:     client,
		collection: client.Collection("users"),
	}
}

func (r *MongoUserRepository) GetByID(ctx context.Context, id string) (*models.MongoUser, error) {
	var user models.MongoUser

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user by their email address
func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*models.MongoUser, error) {
	var user models.MongoUser

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
func (r *MongoUserRepository) Create(ctx context.Context, user *models.MongoUser) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.ID = uuid.MustNewUUID()
	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("user with email %s already exists", user.Email)
		}
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
}

// Update modifies an existing user document
func (r *MongoUserRepository) Update(ctx context.Context, user *models.MongoUser) error {
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
func (r *MongoUserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"password_hash": passwordHash,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating password %w", err)
	}
	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}
	return nil
}

// SetOTP sets the OTP hash and expiry time for password reset
func (r *MongoUserRepository) SetOTP(ctx context.Context, userID string, otpHash string, expiresAt time.Time) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"otp_hash":       otpHash,
			"otp_expires_at": expiresAt,
			"updated_at":     time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error setting OTP: %w", err)
	}
	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}

func (r *MongoUserRepository) ClearOTP(ctx context.Context, userID string) error {
	filter := bson.M{"_id": userID}
	update := bson.M{
		"$unset": bson.M{
			"otp_hash":       "",
			"otp_expires_at": "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
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
func (r *MongoUserRepository) ListByRegion(ctx context.Context, region string, limit, offset int) ([]*models.MongoUser, error) {
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

	var users []*models.MongoUser
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// ListByRole retrieves users with a specific role with pagination
func (r *MongoUserRepository) ListByRole(ctx context.Context, role models.UserRole, limit, offset int) ([]*models.MongoUser, error) {
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

	var users []*models.MongoUser
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// ListByTeam retrieves users in a specific team with pagination
func (r *MongoUserRepository) ListByTeam(ctx context.Context, team string, limit, offset int) ([]*models.MongoUser, error) {
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

	var users []*models.MongoUser
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// Delete removes a user by ID (soft delete recommended in production)
func (r *MongoUserRepository) Delete(ctx context.Context, id string) error {
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
func (r *MongoUserRepository) Count(ctx context.Context, region string) (int64, error) {
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
func (r *MongoUserRepository) UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error {
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
func (r *MongoUserRepository) ActivateUser(ctx context.Context, userID string) error {
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
func (r *MongoUserRepository) DeactivateUser(ctx context.Context, userID string) error {
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
func (r *MongoUserRepository) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.MongoUser, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing all users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.MongoUser
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// GetUsersByRole retrieves users by role (from UserManagementRepository)
func (r *MongoUserRepository) GetUsersByRole(ctx context.Context, role models.UserRole, limit, offset int) ([]*models.MongoUser, error) {
	return r.ListByRole(ctx, role, limit, offset)
}

// GetUsersByTeam retrieves users by team (from UserManagementRepository)
func (r *MongoUserRepository) GetUsersByTeam(ctx context.Context, team string, limit, offset int) ([]*models.MongoUser, error) {
	return r.ListByTeam(ctx, team, limit, offset)
}

// UpdateUserProfile updates user profile information (name, email) from UserSettingsRepository
func (r *MongoUserRepository) UpdateUserProfile(ctx context.Context, userID string, name, email string) error {
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
func (r *MongoUserRepository) EnsureIndexes(ctx context.Context) error {
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

// ============================================================================
// SERVICE LAYER COMPATIBILITY METHODS
// ============================================================================

// GetByIDCompat retrieves a user by ID (service layer compatibility - returns models.User)
func (r *MongoUserRepository) GetByIDCompat(id string) (*models.User, error) {
	ctx := context.Background()
	mongoUser, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mongoUser.ToUser(), nil
}

// GetByEmailCompat retrieves a user by email (service layer compatibility - returns models.User)
func (r *MongoUserRepository) GetByEmailCompat(email string) (*models.User, error) {
	ctx := context.Background()
	mongoUser, err := r.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return mongoUser.ToUser(), nil
}

// UpdateLastLoginCompat updates the last login timestamp (service layer compatibility)
func (r *MongoUserRepository) UpdateLastLoginCompat(userID string, loginTime time.Time) error {
	ctx := context.Background()
	return r.UpdateLastLogin(ctx, userID, loginTime)
}

// UpdatePasswordCompat updates the password hash (service layer compatibility)
func (r *MongoUserRepository) UpdatePasswordCompat(userID string, passwordHash string) error {
	ctx := context.Background()
	return r.UpdatePassword(ctx, userID, passwordHash)
}

// ============================================================================
// SESSION MANAGEMENT METHODS (SessionRepository compatibility)
// ============================================================================

// CreateSession creates a new session in MongoDB
func (r *MongoUserRepository) CreateSession(ctx context.Context, session *models.Session) error {
	collection := r.client.Collection("sessions")
	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}
	return nil
}

// CreateSessionCompat creates a new session (service layer compatibility - no context)
func (r *MongoUserRepository) CreateSessionCompat(session models.Session) error {
	ctx := context.Background()
	return r.CreateSession(ctx, &session)
}

// GetByRefreshToken retrieves a session by refresh token
func (r *MongoUserRepository) GetByRefreshToken(refreshToken string) (*models.Session, error) {
	ctx := context.Background()
	collection := r.client.Collection("sessions")
	var session models.Session

	filter := bson.M{
		"refresh_token": refreshToken,
	}

	err := collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("session not found"))
		}
		return nil, fmt.Errorf("error finding session: %w", err)
	}
	return &session, nil
}

// Revoke revokes a session by marking it as revoked
func (r *MongoUserRepository) Revoke(refreshToken string) error {
	ctx := context.Background()
	collection := r.client.Collection("sessions")

	filter := bson.M{"refresh_token": refreshToken}
	update := bson.M{
		"$set": bson.M{
			"is_revoked": true,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error revoking session: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("session not found"))
	}

	return nil
}

// ============================================================================
// PASSWORD RESET METHODS (PasswordResetRepository compatibility)
// ============================================================================

// CreatePasswordReset creates a new password reset token in MongoDB
func (r *MongoUserRepository) CreatePasswordReset(ctx context.Context, reset *models.PasswordReset) error {
	collection := r.client.Collection("password_resets")
	_, err := collection.InsertOne(ctx, reset)
	if err != nil {
		return fmt.Errorf("error creating password reset: %w", err)
	}
	return nil
}

// Create creates a new password reset (service layer compatibility - no context)
// Note: This method name conflicts with session Create, but Go allows it for different types
func (r *MongoUserRepository) CreateReset(reset models.PasswordReset) error {
	ctx := context.Background()
	return r.CreatePasswordReset(ctx, &reset)
}

// GetByToken retrieves a password reset by token
func (r *MongoUserRepository) GetByToken(token string) (*models.PasswordReset, error) {
	ctx := context.Background()
	collection := r.client.Collection("password_resets")

	var reset models.PasswordReset
	filter := bson.M{
		"reset_token": token,
	}
	err := collection.FindOne(ctx, filter).Decode(&reset)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("password reset not found"))
		}
		return nil, fmt.Errorf("error finding password reset: %w", err)
	}

	return &reset, nil
}

// MarkAsUsed marks a password reset token as used
func (r *MongoUserRepository) MarkAsUsed(resetToken string) error {
	ctx := context.Background()
	collection := r.client.Collection("password_resets")

	filter := bson.M{"reset_token": resetToken}
	update := bson.M{
		"$set": bson.M{
			"is_used": true,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error marking password reset as used: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("password reset not found"))
	}

	return nil
}

// =============================================================================
// USER MANAGEMENT SERVICE LAYER COMPATIBILITY METHODS
// =============================================================================

// GetAllUsersCompat retrieves all users with a limit (service layer compatibility)
func (r *MongoUserRepository) GetAllUsersCompat(limit int) ([]*models.User, error) {
	ctx := context.Background()
	mongoUsers, err := r.GetAllUsers(ctx, limit, 0)
	if err != nil {
		return nil, err
	}
	// Convert MongoUser to User
	users := make([]*models.User, len(mongoUsers))
	for i, mu := range mongoUsers {
		users[i] = mu.ToUser()
	}
	return users, nil
}

// GetUsersByTeamCompat retrieves users by team (service layer compatibility)
func (r *MongoUserRepository) GetUsersByTeamCompat(team string, limit int) ([]*models.User, error) {
	ctx := context.Background()
	collection := r.client.Collection("users")

	filter := bson.M{"team": team}
	opts := options.Find().SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding users by team: %w", err)
	}
	defer cursor.Close(ctx)

	var mongoUsers []*models.MongoUser
	if err := cursor.All(ctx, &mongoUsers); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	// Convert to User
	users := make([]*models.User, len(mongoUsers))
	for i, mu := range mongoUsers {
		users[i] = mu.ToUser()
	}
	return users, nil
}

// CreateCompat creates a user (service layer compatibility)
func (r *MongoUserRepository) CreateCompat(user *models.User) error {
	ctx := context.Background()
	// Convert User to MongoUser for storage
	mongoUser := &models.MongoUser{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Role:         models.UserRole(user.Role), // Convert string to UserRole
		Region:       user.Region,
		Team:         user.Team,
		Permissions:  user.Permissions,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	return r.Create(ctx, mongoUser)
}

// UpdateCompat updates a user (service layer compatibility)
func (r *MongoUserRepository) UpdateCompat(user *models.User) error {
	ctx := context.Background()
	// Convert User to MongoUser for storage
	mongoUser := &models.MongoUser{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Role:         models.UserRole(user.Role), // Convert string to UserRole
		Region:       user.Region,
		Team:         user.Team,
		Permissions:  user.Permissions,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	return r.Update(ctx, mongoUser)
}

// LogActivity logs user activity (service layer compatibility)
func (r *MongoUserRepository) LogActivity(activity *models.UserActivityLog) error {
	ctx := context.Background()
	collection := r.client.Collection("user_activity_logs")

	now := time.Now()
	activity.CreatedAt = now
	if activity.ActivityID == "" {
		activity.ActivityID = uuid.MustNewUUID()
	}

	_, err := collection.InsertOne(ctx, activity)
	if err != nil {
		return fmt.Errorf("error logging user activity: %w", err)
	}

	return nil
}

// GetUserActivities retrieves user activities (service layer compatibility)
func (r *MongoUserRepository) GetUserActivities(userID string, limit int) ([]*models.UserActivityLog, error) {
	ctx := context.Background()
	collection := r.client.Collection("user_activity_logs")

	filter := bson.M{"user_id": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding user activities: %w", err)
	}
	defer cursor.Close(ctx)

	var activities []*models.UserActivityLog
	if err := cursor.All(ctx, &activities); err != nil {
		return nil, fmt.Errorf("error decoding user activities: %w", err)
	}

	return activities, nil
}

// =============================================================================
// SESSION METHODS (Security Handler Compatibility)
// =============================================================================

// GetUserSessions retrieves all active sessions for a user
func (r *MongoUserRepository) GetUserSessions(userID string) ([]models.Session, error) {
	ctx := context.Background()
	collection := r.client.Collection("sessions")

	filter := bson.M{
		"user_id":    userID,
		"is_revoked": bson.M{"$ne": true},
		"expires_at": bson.M{"$gt": time.Now()},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error finding sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []models.Session
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("error decoding sessions: %w", err)
	}

	return sessions, nil
}

// GetSession retrieves a session by ID
func (r *MongoUserRepository) GetSession(sessionID string) (*models.Session, error) {
	ctx := context.Background()
	collection := r.client.Collection("sessions")

	var session models.Session
	filter := bson.M{"_id": sessionID}
	err := collection.FindOne(ctx, filter).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("session not found"))
		}
		return nil, fmt.Errorf("error finding session: %w", err)
	}

	return &session, nil
}

// TerminateSession terminates a session by marking it as revoked
func (r *MongoUserRepository) TerminateSession(sessionID string) error {
	ctx := context.Background()
	collection := r.client.Collection("sessions")

	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$set": bson.M{
			"is_revoked": true,
			"revoked_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error terminating session: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, fmt.Errorf("session not found"))
	}

	return nil
}

// RefreshSession refreshes a session with new expiry time
func (r *MongoUserRepository) RefreshSession(sessionID string, newExpiry time.Time) error {
	ctx := context.Background()
	collection := r.client.Collection("sessions")

	filter := bson.M{"_id": sessionID}
	update := bson.M{
		"$set": bson.M{
			"refreshed_at": time.Now(),
			"expires_at":   newExpiry,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error refreshing session: %w", err)
	}

	return nil
}

// ListUsers lists all users with pagination - stub for handler compatibility
func (r *MongoUserRepository) ListUsers(limit, offset int) ([]*models.MongoUser, error) {
	ctx := context.Background()
	return r.GetAllUsers(ctx, limit, offset)
}

// GetByIDForHandler gets a user by ID (no context) - returns MongoUser
func (r *MongoUserRepository) GetByIDForHandler(id string) (*models.MongoUser, error) {
	return r.GetByID(context.Background(), id)
}

// GetByEmailForHandler gets a user by email (no context)
func (r *MongoUserRepository) GetByEmailForHandler(email string) (*models.MongoUser, error) {
	return r.GetByEmail(context.Background(), email)
}

// CreateForHandler creates a user (no context)
func (r *MongoUserRepository) CreateForHandler(user *models.MongoUser) error {
	return r.Create(context.Background(), user)
}

// UpdateForHandler updates a user (no context)
func (r *MongoUserRepository) UpdateForHandler(user *models.MongoUser) error {
	return r.Update(context.Background(), user)
}

// ListUsersFiltered lists users with filters (handler compatibility)
func (r *MongoUserRepository) ListUsersFiltered(role, region string, isActive *bool, limit, offset int) ([]*models.MongoUser, error) {
	ctx := context.Background()
	filter := bson.M{}

	if role != "" {
		filter["role"] = role
	}
	if region != "" {
		filter["region"] = region
	}
	if isActive != nil {
		filter["is_active"] = *isActive
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.MongoUser
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("error decoding users: %w", err)
	}

	return users, nil
}

// ActivateUserForHandler activates a user (no context)
func (r *MongoUserRepository) ActivateUserForHandler(userID string) error {
	return r.ActivateUser(context.Background(), userID)
}

// DeactivateUserForHandler deactivates a user (no context)
func (r *MongoUserRepository) DeactivateUserForHandler(userID string) error {
	return r.DeactivateUser(context.Background(), userID)
}

// =============================================================================
// USER SETTINGS HANDLER COMPATIBILITY METHODS
// =============================================================================

// GetUserSettings retrieves user settings including profile and preferences
func (r *MongoUserRepository) GetUserSettings(userID string) (*models.UserSettings, error) {
	user, err := r.GetByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	settings := &models.UserSettings{
		UserID:      userID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        string(user.Role),
		Region:      user.Region,
		Team:        user.Team,
		Permissions: user.Permissions,
		// Default preferences
		Language:   "en",
		Timezone:   "UTC",
		DateFormat: "MM/DD/YYYY",
		TimeFormat: "12h",
		Theme:      "light",
	}

	// Override with user preferences if available
	if user.Preferences != nil {
		if user.Preferences.Language != "" {
			settings.Language = user.Preferences.Language
		}
		if user.Preferences.Timezone != "" {
			settings.Timezone = user.Preferences.Timezone
		}
		if user.Preferences.DateFormat != "" {
			settings.DateFormat = user.Preferences.DateFormat
		}
		if user.Preferences.TimeFormat != "" {
			settings.TimeFormat = user.Preferences.TimeFormat
		}
		if user.Preferences.Theme != "" {
			settings.Theme = user.Preferences.Theme
		}
		if user.Preferences.DashboardLayout != "" {
			settings.DashboardLayout = user.Preferences.DashboardLayout
		}
	}

	return settings, nil
}

// GetUserByID retrieves a user by ID (handler compatibility - no context)
func (r *MongoUserRepository) GetUserByID(userID string) (*models.MongoUser, error) {
	return r.GetByID(context.Background(), userID)
}

// UpdateUserProfileForHandler updates user profile (handler compatibility - no context)
func (r *MongoUserRepository) UpdateUserProfileForHandler(userID string, name, email string) error {
	return r.UpdateUserProfile(context.Background(), userID, name, email)
}

// UpdatePreferencesForHandler updates user preferences (handler compatibility)
func (r *MongoUserRepository) UpdatePreferences(prefs *models.UserPreferences) error {
	ctx := context.Background()
	filter := bson.M{"_id": prefs.UserID}
	update := bson.M{
		"$set": bson.M{
			"preferences.language":         prefs.Language,
			"preferences.timezone":         prefs.Timezone,
			"preferences.date_format":      prefs.DateFormat,
			"preferences.time_format":      prefs.TimeFormat,
			"preferences.theme":            prefs.Theme,
			"preferences.dashboard_layout": prefs.DashboardLayout,
			"updated_at":                   time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("error updating preferences: %w", err)
	}

	if result.MatchedCount == 0 {
		return WrapNotFound(mongo.ErrNoDocuments, ErrUserNotFound)
	}

	return nil
}
