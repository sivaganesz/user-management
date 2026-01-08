package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// OTPService handles OTP generation and validation
type OTPService struct {
	otpLength     int
	expiryMinutes int
	bcryptCost    int
}

// NewOTPService creates a new OTP service
func NewOTPService() *OTPService {
	return &OTPService{
		otpLength:     6,
		expiryMinutes: 10,
		bcryptCost:    12, // Cost factor of 12 as per requirements
	}
}

// GenerateOTP generates a random 6-digit OTP
func (s *OTPService) GenerateOTP() string {
	// Generate a random 6-digit number (000000-999999)
	max := big.NewInt(1000000) // 10^6
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to timestamp-based OTP if random generation fails
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}

	// Format as 6-digit string with leading zeros
	return fmt.Sprintf("%06d", n.Int64())
}

// HashOTP hashes an OTP using bcrypt with cost factor 12
func (s *OTPService) HashOTP(otp string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(otp), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash OTP: %w", err)
	}
	return string(hash), nil
}

// ValidateOTP validates an OTP against its hash and expiry time
func (s *OTPService) ValidateOTP(otp, hash string, expiresAt time.Time) bool {
	// Check if OTP has expired
	if time.Now().After(expiresAt) {
		return false
	}

	// Check if OTP or hash is empty
	if otp == "" || hash == "" {
		return false
	}

	// Validate OTP against hash
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(otp))
	return err == nil
}

// GetExpiryTime returns the expiry time for a new OTP (10 minutes from now)
func (s *OTPService) GetExpiryTime() time.Time {
	return time.Now().Add(time.Duration(s.expiryMinutes) * time.Minute)
}

// IsOTPExpired checks if an OTP has expired
func (s *OTPService) IsOTPExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}
