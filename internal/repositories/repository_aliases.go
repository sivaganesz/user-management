package repositories

import "github.com/white/user-management/pkg/mongodb"

// Repository type aliases for backwards compatibility with the service layer.
// These aliases map the legacy repository type names to the MongoDB implementations.

// Core entity repositories
type UserRepository = MongoUserRepository


// User management repositories
type UserManagementRepository = MongoUserRepository
type SessionRepository = MongoUserRepository
type PasswordResetRepository = MongoUserRepository

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