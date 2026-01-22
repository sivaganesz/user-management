package repositories

import (

	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson/primitive"

)

// Repository type aliases for backwards compatibility with the service layer.
// These aliases map the legacy repository type names to the MongoDB implementations.

// Core entity repositories
type UserRepository = MongoUserRepository


// User management repositories
type UserManagementRepository = MongoUserRepository
type SessionRepository = MongoUserRepository
type PasswordResetRepository = MongoUserRepository
type TemplateRepository = MongoTemplateRepository



// RegionalDashboardRepository is defined in mongo_regional_dashboard_repository.go
type UserSettingsRepository = MongoUserRepository

func NewUserRepository(client *mongodb.Client) *MongoUserRepository {
	return NewMongoUserRepository(client)
}

func NewUserManagementRepository(client *mongodb.Client) *MongoUserRepository {
	return NewMongoUserRepository(client)
}

func NewSessionRepository(client *mongodb.Client) *MongoUserRepository {
	return NewMongoUserRepository(client)
}

func NewPasswordResetRepository(client *mongodb.Client) *MongoUserRepository {
	return NewMongoUserRepository(client)
}

// SequenceTemplateFilters contains filters for querying sequence templates
type SequenceTemplateFilters struct {
	Channel   string
	IsActive  *bool
	Category  string
	Tags      []string
	Search    string
	CreatedBy primitive.ObjectID
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}