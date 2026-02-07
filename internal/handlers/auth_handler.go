package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/white/user-management/config"
	"github.com/white/user-management/internal/events"
	"github.com/white/user-management/internal/middleware"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/internal/services"
	"github.com/white/user-management/internal/utils"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/smtp"
	"github.com/white/user-management/pkg/uuid"
)

type AuthHandler struct {
	authService    *services.AuthService
	producer       *kafka.Producer
	config         *config.Config
	settingsRepo   *repositories.SettingsRepository
	otpService     *services.OTPService
	smtpClient     *smtp.SMTPClient
	db             *mongodb.Client
	emailRepo      *repositories.MongoEmailRepository
	userRepo       *repositories.MongoUserRepository
	auditPublisher *events.AuditPublisher
}

func NewAuthHandler(db *mongodb.Client, config *config.Config, producer *kafka.Producer) *AuthHandler {
	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	sessionRepo := repositories.NewSessionRepository(db)
	passwordResetRepo := repositories.NewPasswordResetRepository(db)
	permissionRepo := repositories.NewPermissionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	emailRepo := repositories.NewMongoEmailRepository(db)

	// Initialize JWT service
	jwtService, err := utils.NewJWTService(config.JWT)
	if err != nil {
		panic("Failed to initialize JWT service: " + err.Error())
	}

	authService := services.NewAuthService(userRepo, sessionRepo, passwordResetRepo, permissionRepo, jwtService)
	otpService := services.NewOTPService()
	smtpClient, err := smtp.NewSMTPClientFromEnv()

	return &AuthHandler{
		authService:  authService,
		producer:     producer,
		config:       config,
		settingsRepo: settingsRepo,
		otpService:   otpService,
		smtpClient:   smtpClient,
		db:           db,
		emailRepo:    emailRepo,
		userRepo:     userRepo,
	}
}
// SetAuditPublisher sets the audit publisher for logging auth events
func (h *AuthHandler) SetAuditPublisher(publisher *events.AuditPublisher) {
	h.auditPublisher = publisher
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response body
type LoginResponse struct {
	User                  interface{} `json:"user,omitempty"`
	Tokens                interface{} `json:"tokens,omitempty"`
	Requires2FA           bool        `json:"requires_2fa,omitempty"`
	RequiresPasswordReset bool        `json:"requiresPasswordReset,omitempty"`
	TempToken             string      `json:"temp_token,omitempty"`
	Message               string      `json:"message,omitempty"`
}

// Helper function to get client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// Login godoc
// @Summary User login
// @Description Authenticates a user with email and password, returns JWT tokens or requires 2FA verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param loginRequest body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse "Login successful or 2FA required"
// @Failure 400 {object} ErrorResponse "Invalid request body or missing fields"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	// Basic validation
	if req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Authenticate user (validates password but doesn't create session yet for 2FA flow)
	user, tokens, err := h.authService.Login(req.Email, req.Password, getClientIP(r), r.UserAgent())

	if err != nil {
		if h.auditPublisher != nil {
			h.auditPublisher.PublishAuthEvent(r, "", req.Email, req.Email, events.ActionLoginFailed, false, fmt.Sprintf("Login failed for %s: invalid credentials", req.Email))
		}
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check if user must reset password (master admin first login)
	if user.MustResetPassword {
		// Generate temporary token for password reset (15-minute validity encoded in token)
		tokenData := fmt.Sprintf("reset:%s:%d", user.ID, time.Now().Unix())
		tempToken := base64.URLEncoding.EncodeToString([]byte(tokenData))

		respondWithJSON(w, http.StatusOK, LoginResponse{
			RequiresPasswordReset: true,
			TempToken:             tempToken,
			Message:               "Password reset required. Please set a new password.",
		})
		return
	}

	// Check if user has 2FA enabled
	ctx := context.Background()
	securitySettings, err := h.settingsRepo.GetSecuritySettings(ctx, user.ID)
	fmt.Printf("DEBUG 2FA: userID=%s, err=%v, settings=%+v\n", user.ID, err, securitySettings)
	if err == nil && securitySettings != nil && securitySettings.TwoFactorEnabled {
		//Generate OTP
		otp := h.otpService.GenerateOTP()
		otpHash, err := h.otpService.HashOTP(otp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate OTP")
			return
		}
		// Store OTP in database (reuse password_reset collection with a type field)
		otpExpiry := h.otpService.GetExpiryTime()
		tempToken := uuid.MustNewUUID()

		// Store the 2FA OTP
		err = h.store2FAOTP(ctx, user.ID, tempToken, otpHash, otpExpiry)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to store OTP")
			return
		}

		// Send OTP via email
		fmt.Printf("DEBUG 2FA: OTP code for %s is: %s\n", user.Email, otp)
		err = h.send2FAEmail(user.Email, user.Name, otp)
		if err != nil {
			fmt.Printf("Warning: Failed to send 2FA email: %v\n", err)
			// Continue anyway - OTP is logged in dev mode
		}

		respondWithJSON(w, http.StatusOK, LoginResponse{
			Requires2FA: true,
			TempToken:   tempToken,
			Message:     "2FA verification required. Please check your email for the OTP code.",
		})
	}
	// No 2FA - proceed with normal login
	// Publish login event to Kafka (async, fire-and-forget)
	go h.publishLoginEvent(user, getClientIP(r), r.UserAgent())

	// Publish audit event for successful login
	if h.auditPublisher != nil {
		h.auditPublisher.PublishAuthEvent(r, user.ID, user.Name, user.Email, events.ActionLogin, true, "User logged in successfully")
	}

	// Return response
	respondWithJSON(w, http.StatusOK, LoginResponse{
		User:   user.ToProfile(),
		Tokens: tokens,
	})
}

// LogoutRequest represents the logout request body
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout godoc
// @Summary User logout
// @Description Revokes user session and invalidates refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param logoutRequest body LogoutRequest true "Refresh token to revoke"
// @Success 200 {object} map[string]string "Logout successful"
// @Failure 400 {object} ErrorResponse "Invalid request body or missing refresh token"
// @Failure 401 {object} ErrorResponse "Invalid or expired refresh token"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	// Basic validation
	if req.RefreshToken == "" {
		respondWithError(w, http.StatusBadRequest, "Refresh token is required")
		return
	}

	// Revoke session and invalidate refresh token
	user, err := h.authService.Logout(req.RefreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	// Publish logout event to Kafka (async, fire-and-forget)
	go h.publishLogoutEvent(user, getClientIP(r), r.UserAgent())

	// Publish audit event for logout
	if h.auditPublisher != nil {
		h.auditPublisher.PublishAuthEvent(r, user.ID, user.Name, user.Email, events.ActionLogout, true, "User logged out")
	}

	// Return response
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Logout successful",
	})
}

// RefreshTokenRequest represents the refresh token request body
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents the refresh token response
type RefreshTokenResponse struct {
	Tokens interface{} `json:"tokens"`
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generates new access and refresh tokens using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param refreshRequest body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} RefreshTokenResponse "Token refreshed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or missing refresh token"
// @Failure 401 {object} ErrorResponse "Invalid or expired refresh token"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate refresh token
	if req.RefreshToken == "" {
		respondWithError(w, http.StatusBadRequest, "Refresh token is required")
		return
	}

	// Refresh token
	tokens, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	respondWithJSON(w, http.StatusOK, RefreshTokenResponse{
		Tokens: tokens,
	})
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword godoc
// @Summary Change password
// @Description Changes password for authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param changePasswordRequest body ChangePasswordRequest true "Old and new passwords"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or password validation failed"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate passwords
	if req.OldPassword == "" || req.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, "Old password and new password are required")
		return
	}

	// Get user ID from context (set by auth middleware)
	// The JWT middleware sets user_id in the request context after validating the token
	// For backwards compatibility, we also support X-User-ID header
	// but the middleware approach is preferred and more secure

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// userID, err := primitive.ObjectIDFromHex(userID)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Invalid user ID")
	// 	return
	// }

	// Change password
	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		// Publish audit event for failed password change
		if h.auditPublisher != nil {
			userName, _ := r.Context().Value(middleware.NameKey).(string)
			h.auditPublisher.PublishAuthEvent(r, userID, userName, "", events.ActionPasswordChanged, false, fmt.Sprintf("Password change failed: %v", err))
		}
		respondWithError(w, http.StatusBadRequest, "")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})

}

// ForgotPasswordRequest represents the forgot password request body
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPasswordResponse represents the forgot password response
type ForgotPasswordResponse struct {
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"` // Only for testing/development
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Sends password reset token to user's email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param forgotPasswordRequest body ForgotPasswordRequest true "User email"
// @Success 200 {object} ForgotPasswordResponse "Password reset email sent"
// @Failure 400 {object} ErrorResponse "Invalid request body or email validation failed"
// @Failure 500 {object} ErrorResponse "Failed to create reset token"
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate email
	if req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}
	// Create reset token
	resetToken, err := h.authService.ForgotPassword(
		req.Email,
		getClientIP(r),
		r.UserAgent(),
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create reset token")
		return
	}

	user, err := h.userRepo.GetByEmailCompat(req.Email)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "This is email not exists")
		return
	}
	// Send invitation email via Kafka queue (or direct SMTP as fallback)
	emailSent := false
	inviteURL := fmt.Sprintf("%s/auth/password/reset?token=%s", getAppBaseURL(), resetToken)
	emailErr := h.sendForgetPasswordEmail(req.Email, user.Name, inviteURL)
	if emailErr != nil {
		// Log error but don't fail the request - user is already created
		fmt.Printf("Warning: Failed to send invitation email to %s: %v\n", req.Email, emailErr)
	} else {
		emailSent = true
	}

	// // In production, send email with reset link containing the token
	// // For now, return the token in the response for testing
	// response := ForgotPasswordResponse{
	// 	Message: "If the email exists, a password reset link has been sent",
	// }

	// // Only include token in development mode
	// if h.config.Server.Environment == "development" {
	// 	response.ResetToken = resetToken
	// }

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "If the email exists, a password reset link has been sent",
		"emailSent": emailSent,
	})
}

// ResetPasswordRequest represents the reset password request body
type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword godoc
// @Summary Reset password with token
// @Description Resets user password using reset token from email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param resetPasswordRequest body ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]string "Password reset successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body, token, or password validation failed"
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate inputs
	if req.ResetToken == "" || req.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, "Reset token and new password are required")
		return
	}

	// Parse reset token
	// resetToken, err := primitive.ObjectIDFromHex(req.ResetToken)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Invalid reset token format")
	// 	return
	// }
	// Reset password
	if err := h.authService.ResetPassword(req.ResetToken, req.NewPassword); err != nil {
		// Publish audit event for failed password reset
		if h.auditPublisher != nil {
			h.auditPublisher.PublishAuthEvent(r, "", "", "", events.ActionPasswordReset, false, fmt.Sprintf("Password reset failed: %v", err))
		}
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Publish audit event for successful password reset
	if h.auditPublisher != nil {
		h.auditPublisher.PublishAuthEvent(r, "", "", "", events.ActionPasswordReset, true, "Password reset completed successfully")
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Password reset successfully",
	})

}

type TwoFAOTP struct {
	ID        string    `bson:"_id,omitempty"`
	UserID    string    `bson:"user_id"`
	TempToken string    `bson:"temp_token"`
	OTPHash   string    `bson:"otp_hash"`
	ExpiresAt time.Time `bson:"expires_at"`
	Used      bool      `bson:"used"`
	CreatedAt time.Time `bson:"created_at"`
}

// store2FAOTP stores the 2FA OTP in the database
func (h *AuthHandler) store2FAOTP(ctx context.Context, userID, tempToken string, otpHash string, expiresAt time.Time) error {
	collection := h.db.Database().Collection("two_factor_otps")
	newUUID := uuid.MustNewUUID()
	otp := TwoFAOTP{
		ID:        newUUID,
		UserID:    userID,
		TempToken: tempToken,
		OTPHash:   otpHash,
		ExpiresAt: expiresAt,
		Used:      false,
		CreatedAt: time.Now(),
	}

	_, err := collection.InsertOne(ctx, otp)
	return err
}

// get2FAOTP retrieves the 2FA OTP record by temp token
func (h *AuthHandler) get2FAOTP(ctx context.Context, tempToken string) (*TwoFAOTP, error) {
	collection := h.db.Database().Collection("two_factor_otps")
	var otp TwoFAOTP
	err := collection.FindOne(ctx, map[string]interface{}{
		"temp_token": tempToken,
		"used":       false,
	}).Decode(&otp)

	if err != nil {
		return nil, err
	}

	return &otp, nil
}

// mark2FAOTPUsed marks the OTP as used
func (h *AuthHandler) mark2FAOTPUsed(ctx context.Context, tempToken string) error {
	collection := h.db.Database().Collection("two_factor_otps")

	_, err := collection.UpdateOne(ctx,
		map[string]interface{}{"temp_token": tempToken},
		map[string]interface{}{"$set": map[string]interface{}{"used": true}},
	)
	return err
}

// send2FAEmail sends the 2FA OTP via email using Kafka queue (go-worker handles actual sending)
func (h *AuthHandler) send2FAEmail(email, name, otp string) error {
	subject := "White Platform - Your Login Verification Code"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4F46E5; color: white; padding: 20px; text-align: center; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 5px; margin-top: 20px; }
        .otp-code { font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #4F46E5; text-align: center; margin: 30px 0; padding: 20px; background-color: #fff; border-radius: 5px; border: 2px dashed #4F46E5; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #666; }
        .warning { color: #DC2626; font-weight: bold; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Login Verification</h1>
        </div>
        <div class="content">
            <p>Hello %s,</p>
            <p>You are attempting to login to White Platform. Please use the following verification code to complete your login:</p>

            <div class="otp-code">%s</div>

            <p><strong>This code is valid for 10 minutes.</strong></p>

            <p>If you didn't request this login, please ignore this email and ensure your account is secure.</p>

            <p class="warning">Never share this code with anyone. Our team will never ask for your verification code.</p>
        </div>
        <div class="footer">
            <p>&copy; 2025 White Platform. All rights reserved.</p>
            <p>This is an automated email. Please do not reply.</p>
        </div>
    </div>
</body>
</html>
	`, name, otp)

	plainBody := fmt.Sprintf(`
Login Verification

Hello %s,

You are attempting to login to White Platform. Please use the following verification code to complete your login:

Verification Code: %s

This code is valid for 10 minutes.

If you didn't request this login, please ignore this email and ensure your account is secure.

SECURITY WARNING: Never share this code with anyone. Our team will never ask for your verification code.

---
White Platform
This is an automated email. Please do not reply.
	`, name, otp)

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
		ToAddresses: []string{email},
		Subject:     subject,
		BodyHTML:    htmlBody,
		BodyText:    plainBody,
		Priority:    models.PriorityUrgent, // OTP emails are urgent
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store message in MongoDB
	if h.emailRepo != nil {
		if err := h.emailRepo.CreateMessageCompat(msg); err != nil {
			fmt.Printf("Warning: Failed to store 2FA email in database: %v\n", err)
			// Fall back to direct SMTP if available
			return h.send2FAEmailDirect(email, msg)
		}
	}

	// Future Enhancement: email queuing via Kafka

	// Queue via Kafka for go-worker to process
	// if h.producer != nil {
	// queueMessage := models.NewEmailQueueMessage(
	// 	messageID.Hex(),
	// 	"smtp", // Use SMTP for system emails
	// 	"urgent",
	// )
	// if err := h.producer.PublishJSON("email.queued", queueMessage); err != nil {
	// 	fmt.Printf("Warning: Failed to queue 2FA email to Kafka: %v\n", err)
	// Fall back to direct SMTP if available
	// return h.send2FAEmailDirect(email, msg)
	// }
	// fmt.Printf("2FA email queued successfully for: %s (message_id=%s)\n", email, messageID.Hex())
	// return nil
	// }

	// No Kafka available, fall back to direct SMTP
	return h.send2FAEmailDirect(email, msg)
}

// send2FAEmailDirect sends 2FA email directly via SMTP (fallback when Kafka unavailable)
func (h *AuthHandler) send2FAEmailDirect(email string, msg *models.CommMessage) error {
	if h.smtpClient == nil {
		fmt.Printf("SMTP not configured. 2FA email would be sent to: %s\n", email)
		fmt.Printf("Subject: %s\n", msg.Subject)
		return nil
	}
	if err := h.smtpClient.SendEmail(msg); err != nil {
		fmt.Printf("SMTP ERROR: Failed to send 2FA email to %s: %v\n", email, err)
		return err
	}
	fmt.Printf("2FA email sent successfully to: %s\n", email)
	return nil
}

// sendForgetPasswordEmail sends Forget email via.
// sendForgetPasswordEmail sends forgot password email
func (h *AuthHandler) sendForgetPasswordEmail(toEmail, name, resetLink string) error {
	subject := "Reset your password"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; background-color: #f6f6f6; padding: 20px;">
  <table width="100%%" cellpadding="0" cellspacing="0">
    <tr>
      <td align="center">
        <table width="600" style="background: #ffffff; padding: 30px; border-radius: 8px;">
          <tr>
            <td>
              <h2 style="color: #333;">Reset your password</h2>

              <p>Hello %s,</p>

              <p>
                We received a request to reset your password.
                Click the button below to choose a new one.
              </p>

              <p style="text-align: center; margin: 30px 0;">
                <a href="%s"
                   style="background: #4f46e5; color: #ffffff; padding: 12px 24px;
                          text-decoration: none; border-radius: 6px; font-weight: bold;">
                  Reset Password
                </a>
              </p>

              <p>
                This link will expire in <strong>30 minutes</strong>.
              </p>

              <p>
                If you did not request a password reset, you can safely ignore this email.
              </p>

              <hr style="margin: 30px 0; border: none; border-top: 1px solid #eee;">

              <p style="font-size: 12px; color: #888;">
                If the button doesn’t work, copy and paste this link into your browser:
                <br>
                <a href="%s">%s</a>
              </p>

              <p style="font-size: 12px; color: #888;">
                — The %s Security Team
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
		name,
		resetLink,
		resetLink,
		resetLink,
		"White",
	)

	textBody := fmt.Sprintf(
		`Hello %s,

We received a request to reset your password.

Reset your password using the link below:
%s

This link will expire in 30 minutes.

If you did not request this, you can ignore this email.

— %s Security Team
`,
		name,
		resetLink,
		"user",
	)

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
			return h.sendForgetPasswordEmailDirect(toEmail, msg)
		}
	}
	// Send email via your email service
	return h.sendForgetPasswordEmailDirect(toEmail, msg)
}

func (h *AuthHandler) sendForgetPasswordEmailDirect(toEmail string, msg *models.CommMessage) error {
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

type Verify2FARequest struct {
	TempToken string `json:"temp_token"`
	OTPCode   string `json:"otp_code"`
}

func (h *AuthHandler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Bad Request")
		return
	}

	if req.TempToken == "" || req.OTPCode == "" {
		respondWithError(w, http.StatusBadRequest, "TempToken and OTPCode are required")
		return
	}

	// Validate temp token format (UUID)
	if req.TempToken == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid temp token format")
		return
	}

	// verify tempToken and otp code
	ctx := context.Background()

	//get the OTP record
	storedOTP, err := h.get2FAOTP(ctx, req.TempToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired verification code")
		return
	}

	// Check if OTP is expired
	if h.otpService.IsOTPExpired(storedOTP.ExpiresAt) {
		respondWithError(w, http.StatusUnauthorized, "Verification code has expired")
		return
	}

	// Vaildate OTP code
	if h.otpService.ValidateOTP(req.OTPCode, storedOTP.OTPHash, storedOTP.ExpiresAt) {
		respondWithError(w, http.StatusUnauthorized, "Invalid verification code")
		return
	}

	// Marks OTP as used
	if err := h.mark2FAOTPUsed(ctx, storedOTP.ID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to mark OTP as used")
		return
	}

	userRepo := repositories.NewUserRepository(h.db)
	user, err := userRepo.GetByIDCompat(storedOTP.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve user")
		return
	}

	tokens, err := h.authService.CreateSessionForUser(user, getClientIP(r), r.UserAgent())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user session")
		return
	}

	// Publish login event to Kafka
	go h.publishLoginEvent(user, getClientIP(r), r.UserAgent())

	// Return response
	respondWithJSON(w, http.StatusOK, LoginResponse{
		User:   user.ToProfile(),
		Tokens: tokens,
	})

}

// publishLoginEvent publishes a user login event to Kafka
func (h *AuthHandler) publishLoginEvent(user *models.User, ipAddress, userAgent string) {
	// Skip if Kafka producer is not available
	if h.producer == nil {
		println("Kafka producer not available, skipping event publish")
		return
	}

	event := map[string]interface{}{
		"event_id":     uuid.MustNewUUID(),
		"user_id":      user.ID,
		"email":        user.Email,
		"role":         user.Role,
		"region":       user.Region,
		"team":         user.Team,
		"ip_address":   ipAddress,
		"user_agent":   userAgent,
		"logged_in_at": time.Now().Unix(),
	}

	ctx := context.Background() // or pass the request/context from handler

	if err := h.producer.PublishJSON(ctx, "users.logged_in", event); err != nil {
		// Log error but don't fail the login process
		// In production, use a proper logger
		fmt.Println("Failed to publish login event:", err.Error())
	}

}

// publishLogoutEvent publishes a user logout event to Kafka
func (h *AuthHandler) publishLogoutEvent(user *models.User, ipAddress, userAgent string) {
	// Skip if Kafka producer is not available
	if h.producer == nil {
		println("Kafka producer not available, skipping event publish")
		return
	}

	event := map[string]interface{}{
		"event_id":      uuid.MustNewUUID(),
		"user_id":       user.ID,
		"email":         user.Email,
		"role":          user.Role,
		"region":        user.Region,
		"team":          user.Team,
		"ip_address":    ipAddress,
		"user_agent":    userAgent,
		"logged_out_at": time.Now().Unix(),
	}
	ctx := context.Background() // or pass the request/context from handler

	if err := h.producer.PublishJSON(ctx, "users.logged_out", event); err != nil {
		// Log error but don't fail the logout process
		// In production, you'd use a proper logger
		println("Failed to publish logout event:", err.Error())
	}
}

type InviteUserRequest struct {
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Password    string   `json:"password"`
	Role        string   `json:"role"`
	Region      string   `json:"region"`
	Team        string   `json:"team"`
	Permissions []string `json:"permissions"`
}

// @Security BearerAuth
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Verify user is authenticated
	// invitedBy, ok := getUserIDFromContextUser(r.Context())
	// if !ok {
	// 	respondWithError(w, http.StatusUnauthorized, "Unauthorized")
	// 	return
	// }

	// Parse request body
	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate required fields
	if req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}

	if req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if req.Role == "" {
		respondWithError(w, http.StatusBadRequest, "Role is required")
		return
	}

	// // Check if user with email already exists
	// existingUser, err := h.userRepo.GetByEmailForHandler(req.Email)
	// if err == nil && existingUser != nil {
	// 	respondWithError(w, http.StatusConflict, "User with this email already exists")
	// 	return
	// }

	newUUID := uuid.MustNewUUID()

	// Create new user (inactive by default, pending activation)
	newUser := &models.MongoUser{
		ID:           newUUID,
		Email:        req.Email,
		PasswordHash: req.Password,
		Name:         req.Name,
		Role:         models.UserRole(req.Role),
		Region:       req.Region,
		Team:         req.Team,
		Permissions:  req.Permissions,
		IsActive:     true, // User is inactive until they verify email
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Call repository
	if err := h.authService.CreateForHandler(newUser); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user invitation")
		return
	}
	// Convert to safe user profile
	profile := newUser.ToProfile1()

	respondWithJSON(w, http.StatusCreated, profile)
}
