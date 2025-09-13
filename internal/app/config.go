package app

import (
	"fmt"
	"os"
	"strings"
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

// SetupEnvironment loads .env file and configures zerolog output and log level.
func SetupEnvironment() {
	// Load .env file if it exists
	err := godotenv.Load()

	// Configure logging
	if os.Getenv("ENV") == "production" {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Output(os.Stderr)
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	levelStr := strings.ToLower(os.Getenv("LOGLEVEL"))
	switch levelStr {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "disabled":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	case "":
		// Default based on environment
		if os.Getenv("ENV") == "production" {
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unknown LOGLEVEL '%s', defaulting to info.", levelStr)
	}

	// wait until now to report on the .env file so we have the chance to set up logging first
	if err == nil {
		log.Debug().Msg("Loaded environment variables from .env file.")
	} else {
		log.Debug().Msg("No .env file found or error loading .env file; proceeding with existing environment variables.")
	}
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
