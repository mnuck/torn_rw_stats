package app

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	TornAPIKey      string
	SpreadsheetID   string
	CredentialsFile string
	UpdateInterval  time.Duration
	DeployURL       string

	// BigQuery integration (all optional; empty ProjectID disables BigQuery)
	BigQueryProjectID string
	BigQueryDatasetID string
	BigQueryTableID   string
}

// LogLevel is the package-level log level variable used by SetupEnvironment.
var LogLevel slog.LevelVar

// LevelDisabled is a sentinel level that disables all logging.
const LevelDisabled = slog.Level(math.MaxInt)

// SetupEnvironment loads .env file and configures slog output and log level.
func SetupEnvironment() {
	err := LoadDotEnv(".env")

	var unknownLevel bool
	levelStr := strings.ToLower(os.Getenv("LOGLEVEL"))
	switch levelStr {
	case "debug":
		LogLevel.Set(slog.LevelDebug)
	case "info":
		LogLevel.Set(slog.LevelInfo)
	case "warn", "warning":
		LogLevel.Set(slog.LevelWarn)
	case "error":
		LogLevel.Set(slog.LevelError)
	case "fatal", "panic":
		LogLevel.Set(slog.LevelError)
	case "disabled":
		LogLevel.Set(LevelDisabled)
	case "":
		if os.Getenv("ENV") == "production" {
			LogLevel.Set(slog.LevelWarn)
		} else {
			LogLevel.Set(slog.LevelInfo)
		}
	default:
		LogLevel.Set(slog.LevelInfo)
		unknownLevel = true
	}

	opts := &slog.HandlerOptions{Level: &LogLevel}
	var handler slog.Handler
	if os.Getenv("ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))

	if unknownLevel {
		slog.Warn("Unknown LOGLEVEL, defaulting to info", "loglevel", levelStr)
	}
	if err == nil {
		slog.Debug("Loaded environment variables from .env file")
	} else {
		slog.Debug("No .env file found; proceeding with existing environment variables")
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

	deployURL := os.Getenv("DEPLOY_URL")

	bigQueryProjectID := os.Getenv("BIGQUERY_PROJECT_ID")
	bigQueryDatasetID := os.Getenv("BIGQUERY_DATASET_ID")
	bigQueryTableID := os.Getenv("BIGQUERY_TABLE_ID")
	if bigQueryTableID == "" {
		bigQueryTableID = "state_changes"
	}

	return &Config{
		TornAPIKey:        apiKey,
		SpreadsheetID:     spreadsheetID,
		CredentialsFile:   credentialsFile,
		DeployURL:         deployURL,
		BigQueryProjectID: bigQueryProjectID,
		BigQueryDatasetID: bigQueryDatasetID,
		BigQueryTableID:   bigQueryTableID,
	}, nil
}

// GetRequiredEnv gets an environment variable or exits if not found
func GetRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Required environment variable not set", "key", key)
		os.Exit(1)
	}
	return value
}
