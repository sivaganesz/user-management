package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server		ServerConfig
	MongoDB		MongoDBConfig
	Kafka 		KafkaConfig
	JWT			JWTConfig
	ProcessorPort int
}

type ServerConfig struct {
	Port string
	Environment string
	Version string
}

type MongoDBConfig struct {
	URI         string
	Database    string
	MaxPoolSize uint64
	MinPoolSize uint64
	MaxRetries  int
}

type KafkaConfig struct {
	Brokers         []string
	ProducerTimeout int
	ConsumerGroup   string
	ClientID        string
	Username        string
	Password        string
	SSL             bool
	SASLMechanism   string
	Topics          KafkaTopics
}

type KafkaTopics struct {
	UserLoggedIn     string
	UserLoggedOut    string
	EmailSent        string
}

type JWTConfig struct {
	PrivateKeyPath     string
	PublicKeyPath      string
	AccessTokenExpiry  int    // in minutes
	RefreshTokenExpiry int    // in days
	JWKSEndpoint       string // JWKS endpoint for RS256 validation
	SharedSecret       string // Shared secret for HS256 validation
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	// Set default values
	setDefaults()

	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Try to read config file (optional)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/white-backend")

	// Reading config file is optional
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use environment variables and defaults
	}

	var config Config

	// Server configuration
	config.Server = ServerConfig{
		Port:        viper.GetString("server.port"),
		Environment: viper.GetString("server.environment"),
		Version:     viper.GetString("server.version"),
	}

	// MongoDB configuration
	config.MongoDB = MongoDBConfig{
		URI:         viper.GetString("mongodb.uri"),
		Database:    viper.GetString("mongodb.database"),
		MaxPoolSize: viper.GetUint64("mongodb.max_pool_size"),
		MinPoolSize: viper.GetUint64("mongodb.min_pool_size"),
		MaxRetries:  viper.GetInt("mongodb.max_retries"),
	}

	// Kafka configuration
	config.Kafka = KafkaConfig{
		Brokers:         viper.GetStringSlice("kafka.brokers"),
		ProducerTimeout: viper.GetInt("kafka.producer_timeout"),
		ConsumerGroup:   viper.GetString("kafka.consumer_group"),
		ClientID:        viper.GetString("kafka.client_id"),
		Username:        viper.GetString("kafka.username"),
		Password:        viper.GetString("kafka.password"),
		SSL:             viper.GetBool("kafka.ssl"),
		SASLMechanism:   viper.GetString("kafka.sasl_mechanism"),
		Topics: KafkaTopics{
			UserLoggedIn:     viper.GetString("kafka.topics.user_logged_in"),
			UserLoggedOut:    viper.GetString("kafka.topics.user_logged_out"),
			EmailSent:        viper.GetString("kafka.topics.email_sent"),
		},
	}

	// JWT configuration
	config.JWT = JWTConfig{
		PrivateKeyPath:     viper.GetString("jwt.private_key_path"),
		PublicKeyPath:      viper.GetString("jwt.public_key_path"),
		AccessTokenExpiry:  viper.GetInt("jwt.access_token_expiry"),
		RefreshTokenExpiry: viper.GetInt("jwt.refresh_token_expiry"),
		JWKSEndpoint:       viper.GetString("jwt.jwks_endpoint"),
		SharedSecret:       viper.GetString("jwt.shared_secret"),
	}

	// Processor port configuration
	config.ProcessorPort = viper.GetInt("processor.port")

	return &config, nil
}
func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.version", "1.0.0")
	
	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "white_user_db")
	viper.SetDefault("mongodb.max_pool_size", 100)
	viper.SetDefault("mongodb.min_pool_size", 10)
	viper.SetDefault("mongodb.max_retries", 5)

	// Kafka defaults
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.producer_timeout", 5000)
	viper.SetDefault("kafka.consumer_group", "white-backend")
	viper.SetDefault("kafka.client_id", "white-backend-producer")
	viper.SetDefault("kafka.username", "")
	viper.SetDefault("kafka.password", "")
	viper.SetDefault("kafka.ssl", false)
	viper.SetDefault("kafka.sasl_mechanism", "plain")

	// Kafka topic defaults
	viper.SetDefault("kafka.topics.user_logged_in", "users.logged_in")
	viper.SetDefault("kafka.topics.user_logged_out", "users.logged_out")
	viper.SetDefault("kafka.topics.email_sent", "communications.email_sent")

	// JWT defaults
	viper.SetDefault("jwt.private_key_path", "./secrets/jwt/private.pem")
	viper.SetDefault("jwt.public_key_path", "./secrets/jwt/public.pem")
	viper.SetDefault("jwt.access_token_expiry", 15) // 15 minutes
	viper.SetDefault("jwt.refresh_token_expiry", 7) // 7 days
	viper.SetDefault("jwt.jwks_endpoint", "")       // JWKS endpoint (optional)
	viper.SetDefault("jwt.shared_secret", "")       // Shared secret for HS256 (optional)

	// Processor defaults	
	viper.SetDefault("processor.port", 8081) // Health check port for processor
}