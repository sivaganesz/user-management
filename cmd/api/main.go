// Package main is the entry point for the User Management API.
package main

import (
	"context"
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
	"github.com/white/user-management/internal/utils"
)

func main() {
	// Load environment variables (ignore error in dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

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
