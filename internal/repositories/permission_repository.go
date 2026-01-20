package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/white/user-management/pkg/uuid"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PermissionRepository handles permission resource and role permission operations
type PermissionRepository struct {
	client              *mongodb.Client
	resourcesCollection *mongo.Collection
	rolesCollection     *mongo.Collection
}

// NewPermissionRepository creates a new PermissionRepository
func NewPermissionRepository(client *mongodb.Client) *PermissionRepository {
	return &PermissionRepository{
		client:              client,
		resourcesCollection: client.Collection("permission_resources"),
		rolesCollection:     client.Collection("role_permissions"),
	}
}

// ================================
// Permission Resource Operations
// ================================

// GetAllResources retrieves all permission resources with their sub-scopes and actions
func (r *PermissionRepository) GetAllResources(ctx context.Context) ([]models.PermissionResource, error) {
	filter := bson.M{"isActive": true}
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})

	cursor, err := r.resourcesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find permission resources: %w", err)
	}
	defer cursor.Close(ctx)

	var resources []models.PermissionResource
	if err := cursor.All(ctx, &resources); err != nil {
		return nil, fmt.Errorf("failed to decode permission resources: %w", err)
	}

	return resources, nil
}

// GetResourceByCode retrieves a single permission resource by its code
func (r *PermissionRepository) GetResourceByCode(ctx context.Context, code string) (*models.PermissionResource, error) {
	filter := bson.M{"code": code}

	var resource models.PermissionResource
	err := r.resourcesCollection.FindOne(ctx, filter).Decode(&resource)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find permission resource: %w", err)
	}

	return &resource, nil
}

// CreateResource creates a new permission resource
func (r *PermissionRepository) CreateResource(ctx context.Context, resource *models.PermissionResource) error {
	resource.ID = uuid.MustNewUUID()
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()

	_, err := r.resourcesCollection.InsertOne(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to create permission resource: %w", err)
	}

	return nil
}

// UpdateResource updates an existing permission resource
func (r *PermissionRepository) UpdateResource(ctx context.Context, code string, resource *models.PermissionResource) error {
	filter := bson.M{"code": code}
	resource.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":        resource.Name,
			"description": resource.Description,
			"category":    resource.Category,
			"icon":        resource.Icon,
			"order":       resource.Order,
			"subScopes":   resource.SubScopes,
			"isActive":    resource.IsActive,
			"updatedAt":   resource.UpdatedAt,
		},
	}

	result, err := r.resourcesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update permission resource: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("permission resource not found: %s", code)
	}

	return nil
}

// ================================
// Role Permission Operations
// ================================

// GetAllRoles retrieves all role permissions
func (r *PermissionRepository) GetAllRoles(ctx context.Context) ([]models.RolePermission, error) {
	filter := bson.M{"isActive": true}
	opts := options.Find().SetSort(bson.D{{Key: "roleCode", Value: 1}})

	cursor, err := r.rolesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find role permissions: %w", err)
	}
	defer cursor.Close(ctx)

	var roles []models.RolePermission
	if err := cursor.All(ctx, &roles); err != nil {
		return nil, fmt.Errorf("failed to decode role permissions: %w", err)
	}

	return roles, nil
}

// GetRoleByCode retrieves a single role permission by its code
func (r *PermissionRepository) GetRoleByCode(ctx context.Context, roleCode string) (*models.RolePermission, error) {
	filter := bson.M{"roleCode": roleCode}

	var role models.RolePermission
	err := r.rolesCollection.FindOne(ctx, filter).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find role permission: %w", err)
	}

	return &role, nil
}

// CreateRole creates a new custom role
func (r *PermissionRepository) CreateRole(ctx context.Context, role *models.RolePermission) error {
	role.ID = uuid.MustNewUUID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	_, err := r.rolesCollection.InsertOne(ctx, role)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("role code already exists: %s", role.RoleCode)
		}
		return fmt.Errorf("failed to create role permission: %w", err)
	}

	return nil
}

// UpdateRole updates an existing role's permissions
func (r *PermissionRepository) UpdateRole(ctx context.Context, roleCode string, update *models.UpdateRolePermissionsRequest, updatedBy string) error {
	filter := bson.M{"roleCode": roleCode}

	updateDoc := bson.M{
		"$set": bson.M{
			"roleName":    update.RoleName,
			"description": update.Description,
			"permissions": update.Permissions,
			"dataScope":   update.DataScope,
			"updatedBy":   updatedBy,
			"updatedAt":   time.Now(),
		},
	}

	result, err := r.rolesCollection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update role permission: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role not found: %s", roleCode)
	}

	return nil
}

// DeleteRole soft-deletes a custom role (system roles cannot be deleted)
func (r *PermissionRepository) DeleteRole(ctx context.Context, roleCode string) error {
	// Check if it's a system role
	if models.IsSystemRole(roleCode) {
		return fmt.Errorf("cannot delete system role: %s", roleCode)
	}

	filter := bson.M{"roleCode": roleCode, "isSystemRole": false}
	update := bson.M{
		"$set": bson.M{
			"isActive":  false,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.rolesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role not found or is a system role: %s", roleCode)
	}

	return nil
}

// GetPermissionsForRole retrieves the permissions array for a specific role
// This is the main method used for permission checking
func (r *PermissionRepository) GetPermissionsForRole(ctx context.Context, roleCode string) ([]string, *models.DataScope, error) {
	filter := bson.M{"roleCode": roleCode, "isActive": true}

	var role models.RolePermission
	err := r.rolesCollection.FindOne(ctx, filter).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil, fmt.Errorf("role not found: %s", roleCode)
		}
		return nil, nil, fmt.Errorf("failed to find role: %w", err)
	}

	return role.Permissions, &role.DataScope, nil
}

// ================================
// Index Management
// ================================

// EnsureIndexes creates necessary indexes for permission collections
func (r *PermissionRepository) EnsureIndexes(ctx context.Context) error {
	// Unique index on permission_resources.code
	resourceIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "code", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("code_unique"),
	}
	if _, err := r.resourcesCollection.Indexes().CreateOne(ctx, resourceIndex); err != nil {
		return fmt.Errorf("failed to create index on permission_resources.code: %w", err)
	}

	// Unique index on role_permissions.roleCode
	roleIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "roleCode", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("roleCode_unique"),
	}
	if _, err := r.rolesCollection.Indexes().CreateOne(ctx, roleIndex); err != nil {
		return fmt.Errorf("failed to create index on role_permissions.roleCode: %w", err)
	}

	return nil
}

// ================================
// Utility Methods
// ================================

// GetAllPermissionCodes returns all valid permission codes from the resources
// This is useful for validation and UI
func (r *PermissionRepository) GetAllPermissionCodes(ctx context.Context) ([]string, error) {
	resources, err := r.GetAllResources(ctx)
	if err != nil {
		return nil, err
	}

	var codes []string
	for _, resource := range resources {
		for _, subScope := range resource.SubScopes {
			for _, action := range subScope.Actions {
				code := models.BuildPermission(resource.Code, subScope.Code, action.Code)
				codes = append(codes, code)
			}
		}
	}

	return codes, nil
}

// ValidatePermissions validates that all permission codes in the array are valid
func (r *PermissionRepository) ValidatePermissions(ctx context.Context, permissions []string) error {
	validCodes, err := r.GetAllPermissionCodes(ctx)
	if err != nil {
		return err
	}

	// Build a map for fast lookup
	validMap := make(map[string]bool)
	for _, code := range validCodes {
		validMap[code] = true
	}

	// Check each permission (allow wildcards)
	for _, perm := range permissions {
		// Allow wildcards
		if perm == "*:*:*" {
			continue
		}

		// Check for partial wildcards
		resource, subScope, action := models.ParsePermission(perm)
		if resource == "*" || subScope == "*" || action == "*" {
			// Wildcard permissions are always valid
			continue
		}

		// Exact permission must exist
		if !validMap[perm] {
			return fmt.Errorf("invalid permission code: %s", perm)
		}
	}

	return nil
}

// RoleExists checks if a role code already exists
func (r *PermissionRepository) RoleExists(ctx context.Context, roleCode string) (bool, error) {
	filter := bson.M{"roleCode": roleCode}
	count, err := r.rolesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}
	return count > 0, nil
}
