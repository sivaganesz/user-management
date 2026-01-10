package services

import (
	"fmt"
	"time"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo          *repositories.MongoUserRepository
	sessionRepo       *repositories.SessionRepository
	passwordResetRepo *repositories.PasswordResetRepository
	jwtService        *utils.JWTService
}

func NewAuthService(
	userRepo *repositories.MongoUserRepository,
	sessionRepo *repositories.SessionRepository,
	passwordResetRepo *repositories.PasswordResetRepository,
	jwtService *utils.JWTService,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		sessionRepo:       sessionRepo,
		passwordResetRepo: passwordResetRepo,
		jwtService:        jwtService,
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(email, password, ipAddress, userAgent string) (*models.User, *models.TokenPair, error) {
	user, err := s.userRepo.GetByEmailCompat(email)
	if err != nil {
		return nil, nil, fmt.Errorf("Invalid Credentials")
	}

	if !user.IsActive {
		return nil, nil, fmt.Errorf("Account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		println("Password mismatch:", err.Error())
		return nil, nil, fmt.Errorf("Invalid Credentials")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate access token: %v", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate refresh token: %v", err)
	}

	//create session
	session := models.Session{
		TokenID:      primitive.NewObjectID(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
	}

	if err := s.sessionRepo.CreateSessionCompat(session); err != nil {
		return nil, nil, fmt.Errorf("Failed to create session: %v", err)
	}

	// update last login time
	s.userRepo.UpdateLastLoginCompat(user.ID, time.Now())

	token := &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    7200,
	}

	return user, token, nil
}

// Logout revokes a user's session and returns the user info for event publishing
func (s *AuthService) Logout(refreshToken string) (*models.User, error) {
	userID, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get session to verify it exists
	session, err := s.sessionRepo.GetByRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session is already revoked
	if session.IsRevoked {
		return nil, fmt.Errorf("session already revoked")
	}

	// Get user info before revoking session
	user, err := s.userRepo.GetByIDCompat(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Revoke session
	if err := s.sessionRepo.Revoke(refreshToken); err != nil {
		return nil, fmt.Errorf("failed to revoke session: %v", err)
	}

	return user, nil

}
// RefreshToken generates a new access token using a refresh token
func (s *AuthService) RefreshToken(refreshToken string) (*models.TokenPair, error) {
	// Validate refresh token
	userID, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get session
	session, err := s.sessionRepo.GetByRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session is valid
	if !session.IsVaild() {
		return nil, fmt.Errorf("session expired or revoked")
	}

	// Get user
	user, err := s.userRepo.GetByIDCompat(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active (default to true if not set)
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Generate new access token
	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Return tokens
	tokens := &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Keep the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
	}

	return tokens, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ChangePassword changes a user's password (requires old password verification)
func (s *AuthService) ChangePassword(userID primitive.ObjectID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByIDCompat(userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid current password")
	}

	// Validate new password (minimum 8 characters)
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePasswordCompat(userID, string(newHash)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ForgotPassword creates a password reset token for a user
func (s *AuthService) ForgotPassword(email,ipAddress, userAgent string) (primitive.ObjectID, error) {
	user, err := s.userRepo.GetByEmailCompat(email)
	if err != nil {
		// Don't reveal if user exists or not (security best practice)
		// Return a dummy token to prevent user enumeration
		return primitive.NewObjectID(), nil
	}
	// Check if user is active (default to true if not set)
	if !user.IsActive {
		// Don't reveal account status
		return primitive.NewObjectID(), nil
	}

		// Create reset token
	resetToken := primitive.NewObjectID()
	reset := models.PasswordReset{
		ResetToken: resetToken,
		UserID:     user.ID,
		Email:      user.Email,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		IsUsed:     false,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	if err := s.passwordResetRepo.CreateReset(reset); err != nil {
		return primitive.ObjectID{}, fmt.Errorf("failed to create reset token: %w", err)
	}

	return resetToken, nil
}

// ResetPassword resets a user's password using a reset token
func (s *AuthService) ResetPassword(resetToken primitive.ObjectID, newPassword string) error {
	// Get reset token
	reset, err := s.passwordResetRepo.GetByToken(resetToken)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Validate reset token
	if !reset.IsValid() {
		return fmt.Errorf("reset token has expired or already been used")
	}

	// Validate new password (minimum 8 characters)
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePasswordCompat(reset.UserID, string(newHash)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.passwordResetRepo.MarkAsUsed(resetToken); err != nil {
		// Log error but don't fail the password reset
		fmt.Printf("Warning: failed to mark reset token as used: %v\n", err)
	}

	return nil
}

// CreateSessionForUser creates a session for a user (used for 2FA flow)
func (s *AuthService) CreateSessionForUser(user *models.User, ipAddress, userAgent string) (*models.TokenPair, error) {
	// Generate tokens
	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := models.Session{
		TokenID:      primitive.NewObjectID(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsRevoked:    false,
	}

	if err := s.sessionRepo.CreateSessionCompat(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login time
	s.userRepo.UpdateLastLoginCompat(user.ID, time.Now())

	// Return tokens
	tokens := &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
	}

	return tokens, nil
}

func (s *AuthService) CreateForHandler(user *models.MongoUser) error {
	// Create user in database
	if err := s.userRepo.CreateForHandler(user); err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	return nil
}