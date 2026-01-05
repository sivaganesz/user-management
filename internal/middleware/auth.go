package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/white/user-management/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	UserIDKey      = "user_id"
	EmailKey       = "email"
	NameKey        = "name"
	RoleKey        = "role"
	TeamKey        = "team"
	PermissionsKey = "permissions"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// JWTAuth is a middleware that validates JWT access tokens
func JWTAuth(jwtService *utils.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: ErrorDetail{
						Code:    "MISSING_TOKEN",
						Message: "Authorization header is required",
					},
				})
			}

			//check for Bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INVALID_TOKEN_FORMAT",
						Message: "Authorization header must be in format: Bearer <token>",
					},
				})
				return
			}

			accessToken := parts[1]

			// Validate access token
			claims, err := jwtService.ValidateAccessToken(accessToken)
			if err != nil {
				respondWithJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INVALID_TOKEN",
						Message: "Inalid or expired access token",
					},
				})
				return
			}

			userObjectID, err := primitive.ObjectIDFromHex(claims.UserID)
			if err != nil {
				log.Printf("Invalid user ID format in token: %v", err)
				respondWithJSON(w, http.StatusUnauthorized, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INVALID_TOKEN",
						Message: "Invalid user ID in token",
					},
				})
				return
			}

			// Set user claims in context for use in handlers
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, userObjectID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)
			ctx = context.WithValue(ctx, NameKey, claims.Name)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			ctx = context.WithValue(ctx, TeamKey, claims.Team)
			ctx = context.WithValue(ctx, PermissionsKey, claims.Permissions)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
