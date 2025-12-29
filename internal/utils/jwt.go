package utils

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/white/user-management/config"
	"github.com/white/user-management/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	config     config.JWTConfig
}

// AccessTokenClaims represents the claims in an access token
type AccessTokenClaims struct {
	UserId      string   `json:"sub"` // Subject - User ID
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Roles       []string `json:"roles"` // Array of roles (admin, sales_rep, manager)
	Team        string   `json:"team"`
	Permissions []string `json:"permissions"` // Array of permissions (read, write, delete)
	jwt.RegisteredClaims
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg config.JWTConfig) (*JWTService, error) {
	// Read private key
	privateKeyData, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Read public key
	publicKeyData, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &JWTService{
		privateKey: privateKey,
		publicKey:  publicKey,
		config:     cfg,
	}, nil
}

// GenerateRefreshToken generates a new refresh token
func (s *JWTService) GenerateAccessToken(user *models.User) (string, error) {

	expiryMinutes := time.Duration(s.config.AccessTokenExpiry) * time.Minute

	claims := AccessTokenClaims{
		UserId:      user.ID.Hex(),
		Email:       user.Email,
		Name:        user.Name,
		Role:        user.Role,
		Team:        user.Team,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryMinutes)),
			Issuer:    "white-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// GenerateRefreshToken generates a new refresh token
func (s *JWTService) GenerateRefreshToken(user *models.User) (string, error) {
	expiryDays := time.Duration(s.config.RefreshTokenExpiry) * 24 * time.Hour

	claims := jwt.RegisteredClaims{
		Subject:   user.ID.Hex(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDays)),
		Issuer:    "white-api",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// ValidateAccessToken validates an access token and returns the claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *JWTService) validateRefreshToken(tokenString string) (primitive.ObjectID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return primitive.ObjectID{}, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		userId, err := primitive.ObjectIDFromHex(claims.Subject)
		if err != nil {
			return primitive.ObjectID{}, fmt.Errorf("invalid user ID in token: %w", err)
		}
		return userId, nil
	}
	return primitive.ObjectID{}, fmt.Errorf("invalid token")
}

func (s *JWTService) validateTokenAndGetUserID(tokenString string) (primitive.ObjectID, error) {

	claims, err := s.ValidateAccessToken(tokenString)
	if err != nil {
		return primitive.ObjectID{}, fmt.Errorf("failed to validate access token: %w", err)
	}

	userId, err := primitive.ObjectIDFromHex(claims.UserId)
	if err != nil {
		return primitive.ObjectID{}, fmt.Errorf("invalid user ID in token: %w", err)
	}
	return userId, nil
}
