package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/white/user-management/internal/events"
	"github.com/white/user-management/internal/middleware"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/smtp"
	"github.com/white/user-management/pkg/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// TeamHandler handles team member management endpoints
type TeamHandler struct {
	client         *mongodb.Client
	smtpClient     *smtp.SMTPClient
	kafkaProducer  *kafka.Producer
	emailRepo      *repositories.MongoEmailRepository
	permissionRepo *repositories.PermissionRepository
	auditPublisher *events.AuditPublisher
}

// NewTeamHandler creates a new TeamHandler
func NewTeamHandler(client *mongodb.Client, smtpClient *smtp.SMTPClient, kafkaProducer *kafka.Producer, auditPublisher *events.AuditPublisher) *TeamHandler {
	return &TeamHandler{
		client:         client,
		smtpClient:     smtpClient,
		kafkaProducer:  kafkaProducer,
		emailRepo:      repositories.NewMongoEmailRepository(client),
		permissionRepo: repositories.NewPermissionRepository(client),
		auditPublisher: auditPublisher,
	}
}

// TeamMember represents a team member response
type TeamMember struct {
	ID          string     `json:"id"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Region      string     `json:"region"`
	Team        string     `json:"team"`
	Status      string     `json:"status"`
	Permissions []string   `json:"permissions"`
	Avatar      string     `json:"avatar,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	JobTitle    string     `json:"jobTitle,omitempty"`
	InviteToken string     `json:"inviteToken,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
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
			ID:          getIDField(user, "_id"),
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

	id, err := uuid.ValidateUUID(idStr)
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
		ID:          getIDField(user, "_id"),
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

// generateInviteToken generates a secure random token for invitation
func generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// InviteTeamMember invites a new team member
func (h *TeamHandler) InviteTeamMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Name      string `json:"name"` // Fallback for backward compatibility
		Role      string `json:"role"`
		Region    string `json:"region"`
		Team      string `json:"team"`
		JobTitle  string `json:"jobTitle"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Handle firstName/lastName or fallback to name
	firstName := req.FirstName
	lastName := req.LastName
	if firstName == "" && lastName == "" && req.Name != "" {
		// Split name into first and last name
		parts := splitName(req.Name)
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	if req.Email == "" || (firstName == "" && lastName == "") {
		respondWithError(w, http.StatusBadRequest, "Email and name are required")
		return
	}

	ctx := r.Context()
	collection := h.client.Collection("users")

	// Check if user already exists
	var existing bson.M
	err := collection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		respondWithError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// Generate invite token
	inviteToken, err := generateInviteToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate invite token")
		return
	}

	inviteTokenHash := hashToken(inviteToken)
	// Create new user with invited status
	now := time.Now()
	userID := uuid.MustNewUUID()
	fullName := firstName + " " + lastName
	newUser := bson.M{
		"_id":               userID,
		"email":             req.Email,
		"first_name":        firstName,
		"last_name":         lastName,
		"name":              fullName,
		"role":              getValueOrDefault(req.Role, "sales_rep"),
		"region":            getValueOrDefault(req.Region, "pan_india"),
		"team":              getValueOrDefault(req.Team, "sales"),
		"job_title":         req.JobTitle,
		"status":            "invited",
		"permissions":       []string{},
		"invite_token":      inviteTokenHash,
		"invite_sent_at":    now,
		"invite_expires_at": now.Add(7 * 24 * time.Hour), // 7 days expiry
		"created_at":        now,
		"updated_at":        now,
	}

	_, err = collection.InsertOne(ctx, newUser)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create team member")
		return
	}

	// Send invitation email via Kafka queue (or direct SMTP as fallback)
	emailSent := false
	inviteURL := fmt.Sprintf("%s/signup?token=%s", getAppBaseURL(), inviteTokenHash)
	emailErr := h.sendInvitationEmail(req.Email, firstName, inviteURL)
	if emailErr != nil {
		// Log error but don't fail the request - user is already created
		fmt.Printf("Warning: Failed to send invitation email to %s: %v\n", req.Email, emailErr)
	} else {
		emailSent = true
	}

	// Publish audit event via Kafka (fire-and-forget)
	if h.auditPublisher != nil {
		actorID := middleware.GetUserID(r)
		actorName, _ := ctx.Value(middleware.NameKey).(string)
		h.auditPublisher.PublishTeamEvent(r, actorID, actorName, events.ActionTeamMemberAdded,
			userID, fmt.Sprintf("Team member invited: %s (%s) - role: %s", fullName, req.Email, req.Role))
	}
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success":   true,
		"message":   "Team member invited successfully",
		"emailSent": emailSent,
		"data": map[string]interface{}{
			"id":        userID,
			"email":     req.Email,
			"firstName": firstName,
			"lastName":  lastName,
			"name":      fullName,
		},
	})
}

// sendInvitationEmail sends an invitation email to the new team member using Kafka queue
func (h *TeamHandler) sendInvitationEmail(toEmail, firstName, inviteURL string) error {
	subject := "You're invited to join White Platform"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Team Invitation</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0;">Welcome to White Platform</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p>Hi %s,</p>
        <p>You've been invited to join the White Platform team! White is an AI-powered B2B sales engagement platform that helps teams manage their sales pipeline efficiently.</p>
        <p>Click the button below to complete your registration and get started:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background: #667eea; color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Complete Registration</a>
        </div>
        <p style="color: #666; font-size: 14px;">This invitation link will expire in 7 days.</p>
        <p style="color: #666; font-size: 14px;">If the button doesn't work, copy and paste this link into your browser:</p>
        <p style="color: #667eea; font-size: 12px; word-break: break-all;">%s</p>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 30px 0;">
        <p style="color: #999; font-size: 12px;">This email was sent by White Platform. If you didn't expect this invitation, please ignore this email.</p>
    </div>
</body>
</html>
`, firstName, inviteURL, inviteURL)

	textBody := fmt.Sprintf(`Hi %s,

You've been invited to join the White Platform team!

Click the link below to complete your registration:
%s

This invitation link will expire in 7 days.

Best regards,
White Platform Team
`, firstName, inviteURL)

	// Create email message with unique ID
	messageID := uuid.MustNewUUID()
	now := time.Now()

	msg := &models.CommMessage{
		MessageID:   messageID,
		Channel:     models.ChannelEmail,
		Direction:   models.DirectionOutbound,
		Status:      models.MessageStatusQueued,
		FromAddress: "sivaganesz7482@gmail.com",
		FromName:    "White Platform",
		ToAddresses: []string{toEmail},
		Subject:     subject,
		BodyHTML:    htmlBody,
		BodyText:    textBody,
		Priority:    models.PriorityHigh, // Invitations are high priority
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store message in MongoDB
	if h.emailRepo != nil {
		if err := h.emailRepo.CreateMessageCompat(msg); err != nil {
			fmt.Printf("Warning: Failed to store invitation email in database: %v\n", err)
			// Fall back to direct SMTP if available
			return h.sendInvitationEmailDirect(toEmail, msg)
		}
	}

	// Queue via Kafka for go-worker to process // future use
	// if h.kafkaProducer != nil {
	// 	queueMessage := models.NewEmailQueueMessage(
	// 		messageID.Hex(),
	// 		"smtp", // Use SMTP for system emails
	// 		"high",
	// 	)
	// 	if err := h.kafkaProducer.PublishJSON(kafka.TopicEmailQueue, queueMessage); err != nil {
	// 		fmt.Printf("Warning: Failed to queue invitation email to Kafka topic %s: %v\n", kafka.TopicEmailQueue, err)
	// 		// Fall back to direct SMTP if available
	// 		return h.sendInvitationEmailDirect(toEmail, msg)
	// 	}
	// 	fmt.Printf("Invitation email queued successfully for: %s (message_id=%s)\n", toEmail, messageID.Hex())
	// 	return nil
	// }

	// No Kafka available, fall back to direct SMTP
	return h.sendInvitationEmailDirect(toEmail, msg)
}

// sendInvitationEmailDirect sends invitation email directly via SMTP (fallback when Kafka unavailable)
func (h *TeamHandler) sendInvitationEmailDirect(toEmail string, msg *models.CommMessage) error {
	if h.smtpClient == nil {
		fmt.Printf("SMTP not configured. Invitation email would be sent to: %s\n", toEmail)
		fmt.Printf("Subject: %s\n", msg.Subject)
		return nil
	}

	err := h.smtpClient.SendEmail(msg)
	if err != nil {
		fmt.Printf("SMTP ERROR: Failed to send invitation email to %s: %v\n", toEmail, err)
		return err
	}

	fmt.Printf("Invitation email sent directly via SMTP to: %s\n", toEmail)
	return nil
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

func getIDField(m bson.M, key string) string {
	if val, ok := m[key].(string); ok && val != "" {
		return val
	}
	// Handle legacy ObjectID format for backward compatibility
	if val, ok := m[key].(string); ok {
		if val != "" {
			return val
		}
	}
	return ""
}

func getValueOrDefault(val, defaultVal string) string {
	if val != "" {
		return val
	}
	return defaultVal
}

// UpdateTeamMember updates a team member
func (h *TeamHandler) UpdateTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.ValidateUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid team member ID")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	collection := h.client.Collection("users")

	// Build update document
	update := bson.M{"updated_at": time.Now()}

	// Map frontend field names to database field names
	fieldMapping := map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
		"name":      "name",
		"role":      "role",
		"region":    "region",
		"team":      "team",
		"phone":     "phone",
		"jobTitle":  "job_title",
		"avatar":    "avatar",
	}
	for key, value := range fieldMapping {
		if val, ok := req[key]; ok {
			update[value] = val
		}
	}

	// If firstName or lastName is updated, also update the combined name field
	firstName, hasFirst := req["firstName"].(string)
	lastName, hasLast := req["lastName"].(string)
	if hasFirst || hasLast {
		// Get current values if not provided
		if !hasFirst || !hasLast {
			var existing bson.M
			err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&existing)
			if err == nil {
				if !hasFirst {
					firstName = getStringField(existing, "first_name")
				}
				if !hasLast {
					lastName = getStringField(existing, "last_name")
				}
			}
		}
		update["name"] = firstName + " " + lastName
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil || result.MatchedCount == 0 {
		respondWithError(w, http.StatusInternalServerError, "Failed to update team member")
		return
	}
	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		actorID := middleware.GetUserID(r)
		actorName, _ := ctx.Value(middleware.NameKey).(string)
		h.auditPublisher.PublishTeamEvent(
			r,
			actorID,
			actorName,
			events.ActionTeamMemberUpdated,
			idStr,
			fmt.Sprintf("Team member updated (ID: %s)", idStr),
		)
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Team member updated successfully",
	})
}

// DeactivateTeamMember deactivates a team member
func (h *TeamHandler) DeactivateTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.ValidateUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid team member ID")
		return
	}

	collection := h.client.Collection("users")
	_, err = collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":     "inactive",
			"updated_at": time.Now(),
		},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to deactivate team member")
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		actorID := middleware.GetUserID(r)
		actorName, _ := ctx.Value(middleware.NameKey).(string)
		h.auditPublisher.PublishTeamEvent(
			r,
			actorID,
			actorName,
			events.ActionTeamMemberDeactivated,
			idStr,
			fmt.Sprintf("Team member deactivated (ID: %s)", idStr),
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Team member deactivated successfully",
	})
}

// ReactivateTeamMember reactivates a deactivated team member
func (h *TeamHandler) ReactivateTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.ValidateUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid team member ID")
		return
	}

	collection := h.client.Collection("users")
	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":     "active",
				"updated_at": time.Now()},
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reactivate team member")
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		actorID := middleware.GetUserID(r)
		actorName, _ := ctx.Value(middleware.NameKey).(string)
		h.auditPublisher.PublishTeamEvent(
			r,
			actorID,
			actorName,
			events.ActionTeamMemberActivated,
			idStr,
			fmt.Sprintf("Team member reactivated (ID: %s)", idStr),
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Team member reactivated successfully",
	})
}

// DeleteTeamMember deletes a team member by ID
func (h *TeamHandler) DeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.ValidateUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid team member ID")
		return
	}

	collection := h.client.Collection("users")
	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":     "deleted",
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete team member")
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		actorID := middleware.GetUserID(r)
		actorName, _ := ctx.Value(middleware.NameKey).(string)
		h.auditPublisher.PublishTeamEvent(
			r,
			actorID,
			actorName,
			events.ActionTeamMemberRemoved,
			idStr,
			fmt.Sprintf("Team member deleted (ID: %s)", idStr),
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Team member deleted successfully",
	})
}

// VerifyInviteToken verifies an invite token and returns user info
func (h *TeamHandler) VerifyInviteToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondWithError(w, http.StatusBadRequest, "Token is required")
		return
	}

	ctx := r.Context()
	collection := h.client.Collection("users")

	// Find user by invite token
	var user bson.M
	err := collection.FindOne(ctx, bson.M{
		"invite_token": token,
		"status":       "invited",
	}).Decode(&user)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Invalid or expired invitation token")
		return
	}

	// Check if token has expired
	if expiresAt, ok := user["invite_expires_at"].(primitive.DateTime); ok {
		if time.Now().After(expiresAt.Time()) {
			respondWithError(w, http.StatusGone, "Invitation has expired")
			return
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"email":     getStringField(user, "email"),
			"firstName": getStringField(user, "first_name"),
			"lastName":  getStringField(user, "last_name"),
			"role":      getStringField(user, "role"),
			"region":    getStringField(user, "region"),
		},
	})
}

// CompleteSignup completes the signup process for an invited user
func (h *TeamHandler) CompleteSignup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invaild request body")
	}

	if req.Token == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Token and Password are required")
		return
	}

	if len(req.Password) < 6 {
		respondWithError(w, http.StatusBadRequest, "Password must be at least 6 characters long")
		return
	}
	ctx := r.Context()
	collection := h.client.Collection("users")

	var user bson.M
	err := collection.FindOne(ctx, bson.M{
		"invite_token": req.Token,
		"status":       "invited",
	}).Decode(&user)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Invalid or expired invitation token")
		return
	}

	// check if token has expired
	if expiresAt, ok := user["invite_expires_at"].(primitive.DateTime); ok {
		if time.Now().After(expiresAt.Time()) {
			respondWithError(w, http.StatusGone, "Invitation has expired")
			return
		}
	}

	//hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"password":     hashedPassword,
			"status":       "active",
			"is_active":    true, // Required for login authentication check
			"updated_at":   now,
			"activated_at": now,
		},
		"$unset": bson.M{
			"invite_token":      "",
			"invite_expires_at": "",
		},
	}

	if req.Phone != "" {
		update["$set"].(bson.M)["phone"] = req.Phone
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": user["_id"]}, update)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to complete signup")
		return
	}

	if result.ModifiedCount == 0 {
		respondWithError(w, http.StatusInternalServerError, "Failed to Update user")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Signup completed successfully",
		"data": map[string]interface{}{
			"email": getStringField(user, "email"),
		},
	})

}
