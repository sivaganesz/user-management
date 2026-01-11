package handlers

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/smtp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TeamHandler handles team member management endpoints
type TeamHandler struct {
	client        *mongodb.Client
	smtpClient    *smtp.SMTPClient
	kafkaProducer *kafka.Producer
	emailRepo     *repositories.MongoEmailRepository
}

// NewTeamHandler creates a new TeamHandler
func NewTeamHandler(client *mongodb.Client, smtpClient *smtp.SMTPClient, kafkaProducer *kafka.Producer) *TeamHandler {
	return &TeamHandler{
		client:        client,
		smtpClient:    smtpClient,
		kafkaProducer: kafkaProducer,
		emailRepo:     repositories.NewMongoEmailRepository(client),
	}
}

// TeamMember represents a team member response
type TeamMember struct {
	ID          primitive.ObjectID `json:"id"`
	FirstName   string             `json:"firstName"`
	LastName    string             `json:"lastName"`
	Name        string             `json:"name"`
	Email       string             `json:"email"`
	Role        string             `json:"role"`
	Region      string             `json:"region"`
	Team        string             `json:"team"`
	Status      string             `json:"status"`
	Permissions []string           `json:"permissions"`
	Avatar      string             `json:"avatar,omitempty"`
	Phone       string             `json:"phone,omitempty"`
	JobTitle    string             `json:"jobTitle,omitempty"`
	InviteToken string             `json:"inviteToken,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	LastLogin   *time.Time         `json:"lastLogin,omitempty"`
}

// TeamMembersResponse represents the response for list team members
type TeamMembersResponse struct {
	Success bool         `json:"success"`
	Data    []TeamMember `json:"data"`
	Total   int64        `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}

// ListTeamMembers lists all team members with pagination
func (h *TeamHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination params
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100
			}
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get users from database
	collection := h.client.Collection("users")

	// Count total
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to count team members")
		return
	}

	// Find with pagination
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch team members")
		return
	}
	defer cursor.Close(ctx)

	var members []TeamMember
	for cursor.Next(ctx) {
		var user bson.M
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		firstName := getStringField(user, "first_name")
		lastName := getStringField(user, "last_name")
		name := getStringField(user, "name")
		// If name is empty, construct from firstName and lastName
		if name == "" && (firstName != "" || lastName != "") {
			name = firstName + " " + lastName
		}

		member := TeamMember{
			ID:          getObjectIDField(user, "_id"),
			FirstName:   firstName,
			LastName:    lastName,
			Name:        name,
			Email:       getStringField(user, "email"),
			Role:        getStringField(user, "role"),
			Region:      getStringField(user, "region"),
			Team:        getStringField(user, "team"),
			Status:      getStringFieldWithDefault(user, "status", "active"),
			Permissions: getStringArrayField(user, "permissions"),
			Avatar:      getStringField(user, "avatar"),
			Phone:       getStringField(user, "phone"),
			JobTitle:    getStringField(user, "job_title"),
			InviteToken: getStringField(user, "invite_token"),
		}

		if createdAt, ok := user["created_at"].(primitive.DateTime); ok {
			member.CreatedAt = createdAt.Time()
		}
		if updatedAt, ok := user["updated_at"].(primitive.DateTime); ok {
			member.UpdatedAt = updatedAt.Time()
		}
		if lastLogin, ok := user["last_login"].(primitive.DateTime); ok {
			t := lastLogin.Time()
			member.LastLogin = &t
		}

		members = append(members, member)
	}

	if members == nil {
		members = []TeamMember{}
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"members": members,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		},
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetTeamMember gets a single team member by ID
func (h *TeamHandler) GetTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid team member ID")
		return
	}

	collection := h.client.Collection("users")
	var user bson.M
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Team member not found")
		return
	}

	firstName := getStringField(user, "first_name")
	lastName := getStringField(user, "last_name")
	name := getStringField(user, "name")
	if name == "" && (firstName != "" || lastName != "") {
		name = firstName + " " + lastName
	}

	member := TeamMember{
		ID:          getObjectIDField(user, "_id"),
		FirstName:   firstName,
		LastName:    lastName,
		Name:        name,
		Email:       getStringField(user, "email"),
		Role:        getStringField(user, "role"),
		Region:      getStringField(user, "region"),
		Team:        getStringField(user, "team"),
		Status:      getStringFieldWithDefault(user, "status", "active"),
		Permissions: getStringArrayField(user, "permissions"),
		Avatar:      getStringField(user, "avatar"),
		Phone:       getStringField(user, "phone"),
		JobTitle:    getStringField(user, "job_title"),
		InviteToken: getStringField(user, "invite_token"),
	}

	if createdAt, ok := user["created_at"].(primitive.DateTime); ok {
		member.CreatedAt = createdAt.Time()
	}
	if updatedAt, ok := user["updated_at"].(primitive.DateTime); ok {
		member.UpdatedAt = updatedAt.Time()
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    member,
	})
}

// getAppBaseURL returns the frontend base URL from environment or default
func getAppBaseURL() string {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173" // Default development URL
	}
	return baseURL
}

// splitName splits a full name into first and last name
func splitName(name string) []string {
	parts := make([]string, 0, 2)
	words := []rune(name)
	var current []rune
	for _, c := range words {
		if c == ' ' {
			if len(current) > 0 {
				parts = append(parts, string(current))
				current = nil
			}
		} else {
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	if len(parts) == 0 {
		return []string{""}
	}
	if len(parts) == 1 {
		return parts
	}
	// Join middle names with last name
	return []string{parts[0], joinStrings(parts[1:])}
}
// joinStrings joins multiple strings with space
func joinStrings(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	return result
}

// Helper functions
func getStringField(m bson.M, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getStringFieldWithDefault(m bson.M, key, defaultVal string) string {
	if val, ok := m[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}

func getStringArrayField(m bson.M, key string) []string {
	if val, ok := m[key].(primitive.A); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}

func getObjectIDField(m bson.M, key string) primitive.ObjectID {
	if val, ok := m[key].(primitive.ObjectID); ok {
		return val
	}
	if val, ok := m[key].(string); ok {
		if oid, err := primitive.ObjectIDFromHex(val); err == nil {
			return oid
		}
	}
	return primitive.NilObjectID
}

func getValueOrDefault(val, defaultVal string) string {
	if val != "" {
		return val
	}
	return defaultVal
}