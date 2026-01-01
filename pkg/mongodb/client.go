package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Config struct {
	URI         string
	Database    string
	MaxPoolSize uint64
	MinPoolSize uint64
	MaxRetries  int
	TLSCAFile   string // Path to CA certificate file for TLS
}

type Client struct {
	Client *mongo.Client
	DB     *mongo.Database
	config Config
}

type HealthStatus struct {
	ConnectedNodes int
	MaxPoolSize    uint64
	MinPoolSize    uint64
	DatabaseName   string
	ServerVersion  string
}

// NewClient creates a new MongoDB client with connection pooling and retry logic
// It implements exponential backoff retry strategy: 1s, 2s, 4s, 8s, 16s (max)
func NewClient(config Config) (*Client, error) {
	// Set default values if not provided
	if config.MaxPoolSize == 0 {
		config.MaxPoolSize = 100
	}
	if config.MinPoolSize == 0 {
		config.MinPoolSize = 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 5
	}

	// Validate configuration
	if config.URI == "" {
		return nil, fmt.Errorf("MongoDB URI cannot be empty")
	}
	if config.Database == "" {
		return nil, fmt.Errorf("MongoDB database name cannot be empty")
	}
	if config.MinPoolSize > config.MaxPoolSize {
		return nil, fmt.Errorf("MinPoolSize (%d) cannot be greater than MaxPoolSize (%d)", config.MinPoolSize, config.MaxPoolSize)
	}

	clientOpts := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetMaxConnIdleTime(60 * time.Second).
		SetServerSelectionTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetRetryWrites(true).
		SetRetryReads(true)

	if config.TLSCAFile != "" {
		tlsConfig, err := loadTLSConfig(config.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS CA file: %w", err)
		}
		clientOpts.SetTLSConfig(tlsConfig)
		fmt.Printf("TLS configured with CA file: %s\n", config.TLSCAFile)
	}
	var client *mongo.Client
	var err error

	//Implement retry logic with exponential backoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoffDuration > 16*time.Second {
				backoffDuration = 16 * time.Second
			}

			fmt.Printf("MongoDB connection attempt %d/%d failed, retrying in %v... (error: %v)\n",
				attempt, config.MaxRetries, backoffDuration, err)
			time.Sleep(backoffDuration)
		}

		//create context with timeout for connection attempt
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		//attempt to connect
		client, err = mongo.Connect(ctx, clientOpts)
		if err != nil {
			cancel()
			continue
		}

		err = client.Ping(ctx, readpref.Primary())
		cancel()

		if err == nil {
			break //successful connection
		}

		// If this was last attempt, disconnect and return error
		if attempt == config.MaxRetries {
			if client != nil {
				_ = client.Disconnect(context.Background())
			}
			return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts: %w", config.MaxRetries, err)
		}
	}

	//get database handle
	database := client.Database(config.Database)
	fmt.Printf("Successfully connected to MongoDB database: %s\n", config.Database)

	return &Client{
		Client: client,
		DB:     database,
		config: config,
	}, nil
}

// Ping performs a simple ping to check if the connection is alive
func (c *Client) Ping() error {
	if c.Client == nil {
		return fmt.Errorf("MongoDB client is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Client.Ping(ctx, readpref.Primary())
}

// Collection returns a collection handle
func (c *Client) Collection(name string) *mongo.Collection {
	return c.DB.Collection(name)
}

// Startsession strarts a new client session
func (c *Client) Startsession() (mongo.Session, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("MongoDB client is nil")
	}

	return c.Client.StartSession()
}

// CreateCollection creates a new collection with optional options
func (c *Client) CreateCollection(name string) error {
	if c.DB == nil {
		return fmt.Errorf("MongoDB database is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.DB.CreateCollection(ctx, name);
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", name, err)
	}

	return nil
}

//ListCollections lists all collection names in the database
func (c *Client) ListCollections() ([]string, error) {
	if c.DB == nil {
		return nil, fmt.Errorf("MongoDB database is nil")
	}	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := c.DB.ListCollectionNames(ctx, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	
	return collections, nil
}

//DropCollection drops an existing collection
func (c *Client) DropCollection(name string) error {
	if c.DB == nil {
		return fmt.Errorf("MongoDB database is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.DB.Collection(name).Drop(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop collection %s: %w", name, err)
	}
	return nil
}
// loadTLSConfig loads a TLS configuration with a custom CA certificate
func loadTLSConfig(caFile string) (*tls.Config, error) {
	CaCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(CaCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
	}
	// Create TLS config with the CA pool
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	return tlsConfig, nil
}
