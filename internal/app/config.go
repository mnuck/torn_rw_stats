package app

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds application configuration
type Config struct {
	TornAPIKey      string
	SpreadsheetID   string
	CredentialsFile string
	OurFactionID    int
	UpdateInterval  time.Duration
}

// SetupEnvironment initializes logging and loads environment variables
func SetupEnvironment() {
	// Set up logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load .env file if it exists
	_ = godotenv.Load()
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	apiKey := os.Getenv("TORN_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TORN_API_KEY environment variable is required")
	}

	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	if spreadsheetID == "" {
		return nil, fmt.Errorf("SPREADSHEET_ID environment variable is required")
	}

	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if credentialsFile == "" {
		credentialsFile = "credentials.json"
	}

	return &Config{
		TornAPIKey:      apiKey,
		SpreadsheetID:   spreadsheetID,
		CredentialsFile: credentialsFile,
	}, nil
}

// GetRequiredEnv gets an environment variable or panics if not found
func GetRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatal().Str("key", key).Msg("Required environment variable not set")
	}
	return value
}