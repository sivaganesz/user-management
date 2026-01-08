package handlers

import (

	"github.com/white/user-management/config"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/internal/services"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/smtp"
	"github.com/white/user-management/pkg/mongodb"
)

type AuthHandler struct {
	authService *services.AuthService
	producer    *kafka.Producer
	config 		*config.Config
	settingsRepo *repositories.SettingsRepository
	otpService   *services.OTPService
	smtpClient   *smtp.SMTPClient
	db           *mongodb.Client
	// emailRepo    *repositories.MongoEmailRepository
}