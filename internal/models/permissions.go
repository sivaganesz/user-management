package models

import (
	"strings"
	"time"
)

// ================================
// Permission Resource Definitions
// ================================

// PermissionAction defines a single action within a sub-scope
// Example: { code: "view", name: "View Prospects", frontendRoutes: ["/funnel/prospects"] }
type PermissionAction struct {
	Code            string   `bson:"code" json:"code"`
	Name            string   `bson:"name" json:"name"`
	Description     string   `bson:"description" json:"description"`
	FrontendRoutes  []string `bson:"frontendRoutes" json:"frontendRoutes"`
	BackendPatterns []string `bson:"backendPatterns" json:"backendPatterns"`
}

// PermissionSubScope defines a sub-scope within a resource
// Example: { code: "prospects", name: "Prospects Stage", actions: [...] }
type PermissionSubScope struct {
	Code        string             `bson:"code" json:"code"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Order       int                `bson:"order" json:"order"`
	Actions     []PermissionAction `bson:"actions" json:"actions"`
}

// Stored in 'permission_resources' collection
// Example: { code: "funnel", name: "Sales Funnel", subScopes: [prospects, mqls, ...] }
type PermissionResource struct {
	ID          string   `bson:"_id,omitempty" json:"id"`
	Code        string               `bson:"code" json:"code"`        // e.g., "funnel", "campaigns"
	Name        string               `bson:"name" json:"name"`        // e.g., "Campaign Funnel"
	Description string               `bson:"description" json:"description"`
	Category    string               `bson:"category" json:"category"` // UI grouping
	Icon        string               `bson:"icon" json:"icon"`         // Lucide icon name
	Order       int                  `bson:"order" json:"order"`
	SubScopes   []PermissionSubScope `bson:"subScopes" json:"subScopes"`
	IsActive    bool                 `bson:"isActive" json:"isActive"`
	CreatedAt   time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt" json:"updatedAt"`
}

// ================================
// Role Permission Mappings
// ================================

// DataScope defines data visibility for different resources
// Determines what records a role can see (own, team, region, all)
type DataScope struct {
	Customers string `bson:"customers" json:"customers"` // own | team | region | all
	Campaigns string `bson:"campaigns" json:"campaigns"` // own | team | region | all
}

// RolePermission defines permissions assigned to a role
// Stored in 'role_permissions' collection
// Permissions use 3-part format: "resource:sub_scope:action"
type RolePermission struct {
	ID           string `bson:"_id,omitempty" json:"id"`
	RoleCode     string             `bson:"roleCode" json:"roleCode"`         // e.g., "admin","manager"
	RoleName     string             `bson:"roleName" json:"roleName"`         // e.g., "Super Admin"
	Description  string             `bson:"description" json:"description"`
	IsSystemRole bool               `bson:"isSystemRole" json:"isSystemRole"` // Cannot be deleted
	IsActive     bool               `bson:"isActive" json:"isActive"`

	// Permissions array using 3-part format: "resource:sub_scope:action"
	// Examples: ["campaign:overview:view", "campaign:schedule:view",]
	// Wildcards supported: ["settings:*:*", "*:*:*"]
	Permissions []string  `bson:"permissions" json:"permissions"`
	DataScope   DataScope `bson:"dataScope" json:"dataScope"`

	CreatedBy string `bson:"createdBy" json:"createdBy"`
	UpdatedBy string `bson:"updatedBy" json:"updatedBy"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ================================
// Permission Helper Functions
// ================================

// ParsePermission splits a 3-part permission string into its components
// Format: "resource:sub_scope:action"
// Returns: (resource, subScope, action)
// Examples:
//   - "campaign:schedule:create" → ("campaign", "schedule", "create")
//   - "settings:view" (legacy 2-part) → ("settings", "*", "view")
//   - "admin" (single part) → ("admin", "*", "*")
func ParsePermission(perm string) (resource, subScope, action string) {
	parts := strings.Split(perm, ":")
	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2]
	case 2:
		// Legacy 2-part format: treat as resource:*:action
		return parts[0], "*", parts[1]
	case 1:
		// Single part: treat as resource:*:*
		return parts[0], "*", "*"
	default:
		return perm, "*", "*"
	}
}


// MatchPermission checks if a granted permission matches a required permission
// Supports wildcard matching: "*" matches anything at that level
// Examples:
//   - MatchPermission("campaign:sequence:view", "campaign:sequence:view") → true (exact match)
//   - MatchPermission("campaign:*:view", "campaign:sequence:view") → true (sub-scope wildcard)
//   - MatchPermission("campaign:*:*", "campaign:squence:convert") → true (all actions wildcard)
//   - MatchPermission("*:*:*", "anything:here:works") → true (super admin)
//   - MatchPermission("campaign:email:view", "campaign:sms:view") → false (different sub-scope)

func MatchPermission(granted,required string) bool {
	gRes, gSub, gAct := ParsePermission(granted)
	rRes, rSub, rAct := ParsePermission(required)

	// Check resource level
	if gRes != "*" && gRes != rRes {
		return false
	}
	// Check sub-scope level
	if gSub != "*" && gSub != rSub {
		return false
	}
	// Check action level
	if gAct != "*" && gAct != rAct {
		return false
	}
	return true
}

// HasPermission checks if any granted permission in the list matches the required permission
// This is the main function used by middleware to check access
func HasPermission(grantedPermissions []string, requiredPermission string) bool {
	for _, granted := range grantedPermissions {
		if MatchPermission(granted, requiredPermission) {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if any of the required permissions is granted
func HasAnyPermission(grantedPermissions []string, requiredPermissions ...string) bool {
	for _, required := range requiredPermissions {
		if HasPermission(grantedPermissions, required) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if all required permissions are granted
func HasAllPermissions(grantedPermissions []string, requiredPermissions ...string) bool {
	for _, required := range requiredPermissions {
		if !HasPermission(grantedPermissions, required) {
			return false
		}
	}
	return true
}

// BuildPermission constructs a 3-part permission string
// Usage: BuildPermission("campaign", "schedule", "create") → "campaign:schedule:create"
func BuildPermission(resource, subScope, action string) string {
	return resource + ":" + subScope + ":" + action
}

// ================================
// API Request/Response Types
// ================================


// UpdateRolePermissionsRequest is the request body for updating role permissions
type UpdateRolePermissionsRequest struct {
	RoleName    string    `json:"roleName"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	DataScope   DataScope `json:"dataScope"`
}

// CreateCustomRoleRequest is the request body for creating a new custom role
type CreateCustomRoleRequest struct {
	RoleCode    string    `json:"roleCode"`
	RoleName    string    `json:"roleName"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	DataScope   DataScope `json:"dataScope"`
}

// MyPermissionsResponse is the response for /permissions/my endpoint
type MyPermissionsResponse struct {
	RoleCode    string    `json:"roleCode"`
	RoleName    string    `json:"roleName"`
	Permissions []string  `json:"permissions"`
	DataScope   DataScope `json:"dataScope"`
}

// PermissionResourceListResponse is the response for listing all resources
type PermissionResourceListResponse struct {
	Resources []PermissionResource `json:"resources"`
}

// RolePermissionListResponse is the response for listing all roles
type RolePermissionListResponse struct {
	Roles []RolePermission `json:"roles"`
}

// ================================
// Constants for Data Scope Values
// ================================

const (
	DataScopeOwn    = "own"    // Only records owned by the user
	DataScopeTeam   = "team"   // Records owned by team members
	DataScopeRegion = "region" // Records in user's region
	DataScopeAll    = "all"    // All records (admin level)
	DataScopeNone   = "none"   // No access to this resource
)

// ================================
// Constants for System Roles
// ================================

const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleHunting = "hunting"
	RoleFarming = "farming"
	RoleGenOps  = "genops"
)

// SystemRoles returns the list of system roles that cannot be deleted
func SystemRoles() []string {
	return []string{RoleAdmin, RoleManager, RoleHunting, RoleFarming, RoleGenOps}
}

// IsSystemRole checks if a role code is a system role
func IsSystemRole(roleCode string) bool {
	for _, r := range SystemRoles() {
		if r == roleCode {
			return true
		}
	}
	return false
}
