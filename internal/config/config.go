package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	CORS     CORSConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string
	Host string
	Addr string // Combined host:port for convenience
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Path string
}

// CORSConfig holds CORS-specific configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// Get Allowed Origins from environment
func getCORSOrigins() []string {
	// Check for explicit CORS config first
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		return strings.Split(origins, ",")
	}

	// Check DOMAIN variable (from Docker) second
	if domain := os.Getenv("DOMAIN"); domain != "" {
		return []string{
			fmt.Sprintf("https://%s", domain),
			fmt.Sprintf("http://%s", domain),
		}
	}
	// If nether, return development setting
	return []string{"http://localhost:3000"}
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not loaded: %v (this is OK if using env vars)", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "5001"),
			Host: getEnv("SERVER_HOST", "localhost"),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./data/portfolio_manager.db"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getCORSOrigins(),
		},
	}

	// Combine host and port
	config.Server.Addr = fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)

	return config, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
