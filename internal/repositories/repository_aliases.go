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
type ActivityRepository = MongoActivityRepository



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

// TemplateFilters contains filters for querying templates
type TemplateFilters struct {
	Type         string
	Channel      string
	Category     string
	Tags         []string
	Search       string
	IsActive     *bool
	TenantID     string
	CreatedBy    string
	Status       string
	SortBy       string
	SortOrder    string
	Limit        int
	Offset       int
	Page         int
	ForStage     []string // Filter by funnel stages (prospect, mql, sql, etc.)
	Industries   []string // Filter by industries
	ApprovalFlag string             // Filter by approval flag (green, yellow, red)
	Performance  string             // Filter by performance level (high, medium, low)
	ServiceID    string // Filter by service ID (ObjectID reference)
}
