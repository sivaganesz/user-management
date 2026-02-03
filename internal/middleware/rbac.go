package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/white/user-management/internal/services"
)

// RBACContext loads DB-backed permissions + data scope for the user's role
// and stores them in request context. This makes backend authZ authoritative
// (JWT permissions are treated as non-authoritative).
func RBACContext(rbacService *services.RBACService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rbacService == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			roleCode, _ := ctx.Value(RoleKey).(string)
			if roleCode == "" {
				if roles, ok := ctx.Value("roles").([]string); ok && len(roles) > 0 {
					roleCode = roles[0]
				}
			}

			if roleCode == "" {
				log.Printf("RBAC: 403 %s %s - role not in context", r.Method, r.URL.Path)
				respondWithJSON(w, http.StatusForbidden, ErrorResponse{
					Error: ErrorDetail{
						Code:    "ROLE_REQUIRED",
						Message: "User role not found",
					},
				})
				return
			}

			perms, dataScope, err := rbacService.GetPermissionsForRole(ctx, roleCode)
			if err != nil {
				log.Printf("RBAC: failed to load permissions for role %s: %v (path: %s %s)", roleCode, err, r.Method, r.URL.Path)
				respondWithJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INTERNAL_ERROR",
						Message: "Failed to load permissions",
					},
				})
				return
			}

			// If DB returns no permissions (e.g. admin role not in role_permissions yet),
			// keep JWT permissions so admin/super-user tokens still work
			jwtPerms, _ := r.Context().Value(PermissionsKey).([]string)
			if len(perms) == 0 {
				if len(jwtPerms) > 0 {
					perms = jwtPerms
				}
			} else {
				// When DB returns permissions, preserve JWT super-admin: if JWT had *:*:*,
				// ensure it stays in the list so full-access tokens are not downgraded by DB data.
				hasSuperAdmin := false
				for _, p := range jwtPerms {
					if p == "*:*:*" {
						hasSuperAdmin = true
						break
					}
				}
				if hasSuperAdmin {
					found := false
					for _, p := range perms {
						if p == "*:*:*" {
							found = true
							break
						}
					}
					if !found {
						perms = append([]string{"*:*:*"}, perms...)
					}
				}
			}

			ctx = context.WithValue(ctx, PermissionsKey, perms)
			if dataScope != nil {
				ctx = context.WithValue(ctx, DataScopeKey, *dataScope)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

