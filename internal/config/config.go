package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	APIKey string
	Model  string
	Url    string
}

// LoadConfig loads configuration from .env file and environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// If .env file doesn't exist, that's fine, we'll use environment variables
		fmt.Printf("Warning: could not load .env file: %v\n", err)
	}

	config := &Config{
		APIKey: getEnvOrDefault("API_KEY", ""),
		Model:  getEnvOrDefault("MODEL", "gpt-3.5-turbo"),
		Url:    getEnvOrDefault("URL", "https://api.groq.com/openai/v1"),
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required but not set in environment or .env file")
	}

	return config, nil
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
