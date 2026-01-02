package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	OpenAPI   OpenAPIConfig
	Collector CollectorConfig
	Logging   LoggingConfig
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port int
	Mode string
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Type     string // "sqlite" or "mysql"
	FilePath string // for SQLite
	Host     string // for MySQL
	Port     int    // for MySQL
	Username string // for MySQL
	Password string // for MySQL
	Database string // for MySQL
}

// OpenAPIConfig represents the external API configuration
type OpenAPIConfig struct {
	BaseURL    string
	ServiceKey string
}

// CollectorConfig represents the data collector configuration
type CollectorConfig struct {
	IntervalMs       int
	RetryMaxAttempts int
	RetryBackoffMs   int
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// DSN returns the database connection string
func (d *DatabaseConfig) DSN() string {
	if d.Type == "sqlite" {
		return d.FilePath
	}
	// MySQL DSN
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database)
}

// DriverName returns the SQL driver name
func (d *DatabaseConfig) DriverName() string {
	if d.Type == "sqlite" {
		return "sqlite3"
	}
	return "mysql"
}

// LoadFromSettings loads configuration from AppSettings
func LoadFromSettings(settings *AppSettings) *Config {
	dbPath := filepath.Join(settings.StoragePath, "bus_history.db")

	interval := settings.IntervalMs
	if interval <= 0 {
		interval = 30000 // Default 30s
	}

	return &Config{
		Database: DatabaseConfig{
			Type:     "sqlite",
			FilePath: dbPath,
		},
		OpenAPI: OpenAPIConfig{
			BaseURL:    "https://apis.data.go.kr/6410000/busarrivalservice/v2",
			ServiceKey: settings.ServiceKey,
		},
		Collector: CollectorConfig{
			IntervalMs:       interval,
			RetryMaxAttempts: 3,
			RetryBackoffMs:   1000,
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}
}

// Load loads configuration from environment variables (Legacy/Dev)
func Load(envPath string) (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if envPath == "" {
		envPath = ".env"
	}
	_ = godotenv.Load(envPath)

	dbType := getEnv("DB_TYPE", "sqlite")

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("SERVER_PORT", 8080),
			Mode: getEnv("SERVER_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Type:     dbType,
			FilePath: getEnv("DB_FILE_PATH", "./bus_history.db"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 3306),
			Username: getEnv("DB_USERNAME", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_DATABASE", "bus_history"),
		},
		OpenAPI: OpenAPIConfig{
			BaseURL:    getEnv("API_BASE_URL", "https://apis.data.go.kr/6410000/busarrivalservice/v2"),
			ServiceKey: getEnv("API_SERVICE_KEY", ""),
		},
		Collector: CollectorConfig{
			IntervalMs:       getEnvAsInt("COLLECTOR_INTERVAL_MS", 30000),
			RetryMaxAttempts: getEnvAsInt("COLLECTOR_RETRY_MAX_ATTEMPTS", 3),
			RetryBackoffMs:   getEnvAsInt("COLLECTOR_RETRY_BACKOFF_MS", 1000),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate required fields
	if cfg.OpenAPI.ServiceKey == "" {
		return nil, fmt.Errorf("API_SERVICE_KEY environment variable is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
