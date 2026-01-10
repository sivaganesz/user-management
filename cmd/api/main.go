// Package main is the entry point for the User Management API.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/white/user-management/config"
	"github.com/white/user-management/internal/handlers"
	"github.com/white/user-management/internal/middleware"
	// "github.com/white/user-management/internal/repositories"
	// "github.com/white/user-management/internal/services"
	"github.com/white/user-management/internal/utils"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/mongodb"
	"github.com/white/user-management/pkg/smtp"
)

func main() {
	// Load environment variables (ignore error in dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	mongoURI := os.Getenv("MONGODB_URL")
	if mongoURI == "" {
		log.Fatal("FATAL: MONGODB_URI environment variable is required but not set. Please configure MongoDB connection.")
	}

	log.Printf("Connecting to MongoDB...")
	// Initialize MongoDB client
	mongoConfig := mongodb.Config{
		URI: mongoURI,
		Database: getEnvWithDefault("MONGODB_DATABASE", "white-dev"),
		MaxPoolSize: uint64(getEnvIntWithDefault("MONGODB_MAX_POOL_SIZE", 100)),
		MinPoolSize: uint64(getEnvIntWithDefault("MONGODB_MIN_POOL_SIZE", 10)),
		MaxRetries:  getEnvIntWithDefault("MONGODB_MAX_RETRIES", 5),	
	}

	mongoClient, err := mongodb.NewClient(mongoConfig)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to MongoDB: %v. Application cannot start without database.", err)
	}
	defer mongoClient.Close()
	log.Println("Successfully connected to MongoDB")


	// Initialize Kafka producer (optional - gracefully handle if not available)
	var kafkaProducer *kafka.Producer

	// Load Kafka configuration from environment
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092" // Default for local development
	}

	kafkaConfig := config.KafkaConfig{
		Brokers:       []string{kafkaBrokers},
		ClientID:      getEnvWithDefault("KAFKA_CLIENT_ID", "white-backend-producer"),
		Username:      os.Getenv("KAFKA_USER_NAME"),
		Password:      os.Getenv("KAFKA_PASSWORD"),
		SSL:           os.Getenv("KAFKA_SSL") == "true",
		SASLMechanism: getEnvWithDefault("KAFKA_SASL_MECHANISM", "plain"),
	}

	kafkaProducer, err = kafka.NewProducer(kafkaConfig)
	if err != nil {
		log.Printf("Warning: Kafka producer not available: %v. Events will not be published.", err)
		kafkaProducer = nil // Set to nil so handlers can check
	} else {
		log.Println("Connected to Kafka Producer")
		defer kafkaProducer.Close()
	}

	// Initialize SMTP client for email sending (Office 365 or other SMTP servers)
	var smtpClient *smtp.SMTPClient
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost != "" {
		var err error
		smtpClient, err = smtp.NewSMTPClientFromEnv()
		if err != nil {
			log.Printf("Warning: SMTP client initialization failed: %v. SMTP email will not be available.", err)
			smtpClient = nil
		} else {
			log.Printf("SMTP email client initialized (host: %s, from: %s)", smtpHost, smtpClient.GetFromEmail())
		}
	} else {
		log.Println("Warning: SMTP_HOST not configured. SMTP email will not be available.")
	}

	
	// =====================================================
	// MONGODB REPOSITORIES
	// =====================================================
	// Initialize MongoDB repositories (MongoDB-based)

	// mongoUserRepo := repositories.NewMongoUserRepository(mongoClient)
	// Settings repository (User Settings, Company Settings, Notifications, Audit Logs)
	// settingsRepo := repositories.NewSettingsRepository(mongoClient)
	// log.Println("MongoDB repositories initialized (all modules including Phase 3)")
	// emailRepo := repositories.NewMongoEmailRepository(mongoClient)

	// =====================================================
	// COMMENTED OUT: Legacy services (removed)
	// =====================================================

	// emailThreadingService := services.NewEmailThreadingService(emailRepo)
	// Event Publisher Service (Communications Hub - Kafka Event Publishing)
	// _ = services.NewEventPublisher(kafkaProducer)
	// log.Println("Event publisher service initialized")


	// =====================================================
	// MONGODB HANDLERS (TASK GROUP 1: MongoDB Migration Complete)
	// =====================================================
	// Initialize MongoDB handlers (using MongoDB repositories)


	// Initialize router
	router := mux.NewRouter()

	// Add CORS middleware (must be first to handle preflight OPTIONS requests)
	router.Use(corsMiddleware)

	// Custom NotFoundHandler with CORS headers (for routes that don't exist)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Viewer-ID, X-Viewer-Type, X-Session-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"Endpoint not found"}}`))
	})

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Health check endpoints
	router.HandleFunc("/health", handlers.GetOverallHealth).Methods("GET", "OPTIONS")

	//Swagger ui endpoint - API documentation
	router.PathPrefix("swagger").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	// Initialize JWT config
	cfg := &config.Config{
		JWT: config.JWTConfig{
			PrivateKeyPath:     "./secrets/jwt/private.pem",
			PublicKeyPath:      "./secrets/jwt/public.pem",
			AccessTokenExpiry:  2880, // 2 days (48 hours) in minutes for development
			RefreshTokenExpiry: 7,    // 7 days
		},
		Server: config.ServerConfig{
			Environment: "development",
		},
	}
	
	// Initialize JWT service
	jwtService, err := utils.NewJWTService(cfg.JWT)
	if err != nil {
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}
	log.Println("JWT service initialized")

	// Initialize JWT middleware
	authMiddleware := middleware.JWTAuth(jwtService)

	// Prevent "declared and not used" errors (Task Group 1)
	_ = api            // Will be used in future task groups for route registration
	_ = authMiddleware // Will be used in future task groups for protected routes

	// api.HandleFunc("auth/register")
	// HTTP server configuration
	srv := &http.Server{
		Addr:         ":" + getEnvWithDefault("PORT", "8080"),
		Handler:      corsMiddleware(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// =====================================================
	// Authentication Routes (MongoDB-based)
	// =====================================================
	authHandler := handlers.NewAuthHandler(mongoClient,cfg, kafkaProducer)
	// authHandler.SetAuditPublisher(auditPublisher)
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/verify-2fa", authHandler.Verify2FA).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", authHandler.Logout).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/password/change", authHandler.ChangePassword).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/password/forgot", authHandler.ForgotPassword).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/password/reset", authHandler.ResetPassword).Methods("POST", "OPTIONS")

	api.HandleFunc("/create/new-user", authHandler.CreateUser).Methods("POST", "OPTIONS")



	// Start server
	go func() {
		log.Printf("Server running on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// getEnvIntWithDefault gets an environment variable as int or returns a default value
func getEnvIntWithDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue := 0
	_, err := fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s: %v, using default: %d", key, value, defaultValue)
		return defaultValue
	}
	return intValue
}

// getEnvWithDefault returns an environment variable or a default value.
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// corsMiddleware adds CORS headers to responses.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:5174",
			"http://localhost:5175",
		}

		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				originAllowed = true
				break
			}
		}

		if !originAllowed && origin != "" {
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
