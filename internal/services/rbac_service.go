package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/redis/go-redis/v9"

)

// RBACService handles Role-Based Access Control with Redis caching
// This service manages dynamic permissions for roles (not resource-level sharing)
type RBACService struct {
	repo        *repositories.PermissionRepository
	redisClient *redis.Client
	cacheTTL    time.Duration
}

// CachedRolePermissions is the structure stored in Redis
type CachedRolePermissions struct {
	Permissions []string         `json:"permissions"`
	DataScope   models.DataScope `json:"dataScope"`
	CachedAt    time.Time        `json:"cachedAt"`
}

// NewRBACService creates a new RBACService
// redisClient can be nil - caching will be skipped if unavailable
func NewRBACService(repo *repositories.PermissionRepository, redisClient *redis.Client) *RBACService {
	return &RBACService{
		repo:        repo,
		redisClient: redisClient,
		cacheTTL:    5 * time.Minute, // 5 minute TTL for permission cache
	}
}

// ================================
// Permission Checking (Core Functions)
// ================================

// HasPermission checks if a role has a specific permission
// Uses Redis cache with fallback to MongoDB
// Permission format: "resource:sub_scope:action" (e.g., "funnel:prospects:convert")
func (s *RBACService) HasPermission(ctx context.Context, roleCode, requiredPermission string) bool {
	permissions, _, err := s.GetPermissionsForRole(ctx, roleCode)
	if err != nil {
		log.Printf("Error getting permissions for role %s: %v", roleCode, err)
		return false
	}

	return models.HasPermission(permissions, requiredPermission)
}

// HasAnyPermission checks if a role has any of the specified permissions
func (s *RBACService) HasAnyPermission(ctx context.Context, roleCode string, requiredPermissions ...string) bool {
	permissions, _, err := s.GetPermissionsForRole(ctx, roleCode)
	if err != nil {
		log.Printf("Error getting permissions for role %s: %v", roleCode, err)
		return false
	}

	return models.HasAnyPermission(permissions, requiredPermissions...)
}

// HasAllPermissions checks if a role has all of the specified permissions
func (s *RBACService) HasAllPermissions(ctx context.Context, roleCode string, requiredPermissions ...string) bool {
	permissions, _, err := s.GetPermissionsForRole(ctx, roleCode)
	if err != nil {
		log.Printf("Error getting permissions for role %s: %v", roleCode, err)
		return false
	}

	return models.HasAllPermissions(permissions, requiredPermissions...)
}

// GetPermissionsForRole retrieves permissions for a role (with caching)
func (s *RBACService) GetPermissionsForRole(ctx context.Context, roleCode string) ([]string, *models.DataScope, error) {
	// Try cache first
	if s.redisClient != nil {
		cached, err := s.getFromCache(ctx, roleCode)
		if err == nil && cached != nil {
			return cached.Permissions, &cached.DataScope, nil
		}
		// Cache miss or error - continue to database
	}

	// Get from database
	permissions, dataScope, err := s.repo.GetPermissionsForRole(ctx, roleCode)
	if err != nil {
		return nil, nil, err
	}

	// Store in cache (fire-and-forget)
	if s.redisClient != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := s.setInCache(cacheCtx, roleCode, permissions, dataScope); err != nil {
				log.Printf("Failed to cache permissions for role %s: %v", roleCode, err)
			}
		}()
	}

	return permissions, dataScope, nil
}

// GetDataScopeForRole retrieves the data scope for a role
func (s *RBACService) GetDataScopeForRole(ctx context.Context, roleCode string) (*models.DataScope, error) {
	_, dataScope, err := s.GetPermissionsForRole(ctx, roleCode)
	return dataScope, err
}

// ================================
// Cache Operations
// ================================

// buildCacheKey creates the Redis key for a role's permissions
// Format: rbac:role:{roleCode}
func (s *RBACService) buildCacheKey(roleCode string) string {
	return fmt.Sprintf("rbac:role:%s", roleCode)
}

// getFromCache retrieves permissions from Redis cache
func (s *RBACService) getFromCache(ctx context.Context, roleCode string) (*CachedRolePermissions, error) {
	key := s.buildCacheKey(roleCode)

	val, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var cached CachedRolePermissions
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached permissions: %w", err)
	}

	return &cached, nil
}

// setInCache stores permissions in Redis cache
func (s *RBACService) setInCache(ctx context.Context, roleCode string, permissions []string, dataScope *models.DataScope) error {
	key := s.buildCacheKey(roleCode)

	cached := CachedRolePermissions{
		Permissions: permissions,
		CachedAt:    time.Now(),
	}
	if dataScope != nil {
		cached.DataScope = *dataScope
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("failed to serialize permissions: %w", err)
	}

	return s.redisClient.Set(ctx, key, data, s.cacheTTL).Err()
}

// InvalidateRoleCache removes a role's permissions from cache
// Call this when role permissions are updated
func (s *RBACService) InvalidateRoleCache(ctx context.Context, roleCode string) error {
	if s.redisClient == nil {
		return nil
	}

	key := s.buildCacheKey(roleCode)
	return s.redisClient.Del(ctx, key).Err()
}

// InvalidateAllRolesCache removes all role permission caches
// Call this when resources are updated (affects all roles)
func (s *RBACService) InvalidateAllRolesCache(ctx context.Context) error {
	if s.redisClient == nil {
		return nil
	}

	// Scan for all permission cache keys
	pattern := "rbac:role:*"
	iter := s.redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan permission cache keys: %w", err)
	}

	if len(keys) > 0 {
		if err := s.redisClient.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete permission cache keys: %w", err)
		}
	}

	return nil
}

// ================================
// Resource Operations (Delegated to Repository)
// ================================

// GetAllResources retrieves all permission resources
func (s *RBACService) GetAllResources(ctx context.Context) ([]models.PermissionResource, error) {
	return s.repo.GetAllResources(ctx)
}

// GetResourceByCode retrieves a permission resource by code
func (s *RBACService) GetResourceByCode(ctx context.Context, code string) (*models.PermissionResource, error) {
	return s.repo.GetResourceByCode(ctx, code)
}

// ================================
// Role Operations (Delegated to Repository with Cache Invalidation)
// ================================

// GetAllRoles retrieves all role permissions
func (s *RBACService) GetAllRoles(ctx context.Context) ([]models.RolePermission, error) {
	return s.repo.GetAllRoles(ctx)
}

// GetRoleByCode retrieves a role by code
func (s *RBACService) GetRoleByCode(ctx context.Context, roleCode string) (*models.RolePermission, error) {
	return s.repo.GetRoleByCode(ctx, roleCode)
}

// CreateRole creates a new custom role
func (s *RBACService) CreateRole(ctx context.Context, role *models.RolePermission) error {
	// Validate permissions before creating
	if err := s.repo.ValidatePermissions(ctx, role.Permissions); err != nil {
		return fmt.Errorf("invalid permissions: %w", err)
	}

	return s.repo.CreateRole(ctx, role)
}

// UpdateRole updates a role's permissions and invalidates cache
func (s *RBACService) UpdateRole(ctx context.Context, roleCode string, update *models.UpdateRolePermissionsRequest, updatedBy string) error {
	// Validate permissions before updating
	if err := s.repo.ValidatePermissions(ctx, update.Permissions); err != nil {
		return fmt.Errorf("invalid permissions: %w", err)
	}

	// Update in database
	if err := s.repo.UpdateRole(ctx, roleCode, update, updatedBy); err != nil {
		return err
	}

	// Invalidate cache
	return s.InvalidateRoleCache(ctx, roleCode)
}

// DeleteRole deletes a custom role and invalidates cache
func (s *RBACService) DeleteRole(ctx context.Context, roleCode string) error {
	if err := s.repo.DeleteRole(ctx, roleCode); err != nil {
		return err
	}

	// Invalidate cache
	return s.InvalidateRoleCache(ctx, roleCode)
}

// ================================
// User Permission Response
// ================================

// GetMyPermissions returns the current user's permissions response
func (s *RBACService) GetMyPermissions(ctx context.Context, roleCode string) (*models.MyPermissionsResponse, error) {
	role, err := s.repo.GetRoleByCode(ctx, roleCode)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, fmt.Errorf("role not found: %s", roleCode)
	}

	return &models.MyPermissionsResponse{
		RoleCode:    role.RoleCode,
		RoleName:    role.RoleName,
		Permissions: role.Permissions,
		DataScope:   role.DataScope,
	}, nil
}

// ================================
// Utility Methods
// ================================

// ValidatePermissions validates that all permission codes are valid
func (s *RBACService) ValidatePermissions(ctx context.Context, permissions []string) error {
	return s.repo.ValidatePermissions(ctx, permissions)
}

// RoleExists checks if a role code exists
func (s *RBACService) RoleExists(ctx context.Context, roleCode string) (bool, error) {
	return s.repo.RoleExists(ctx, roleCode)
}

// GetAllPermissionCodes returns all valid permission codes
func (s *RBACService) GetAllPermissionCodes(ctx context.Context) ([]string, error) {
	return s.repo.GetAllPermissionCodes(ctx)
}
