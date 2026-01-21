package middleware

import (
	"context"
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
				respondWithJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INTERNAL_ERROR",
						Message: "Failed to load permissions",
					},
				})
				return
			}

			ctx = context.WithValue(ctx, PermissionsKey, perms)
			if dataScope != nil {
				ctx = context.WithValue(ctx, DataScopeKey, *dataScope)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

