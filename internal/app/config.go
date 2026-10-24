package app

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"torn_rw_stats/internal/domain/war"
)

// Config holds application configuration
type Config struct {
	TornAPIKey      string
	SpreadsheetID   string
	CredentialsFile string
	UpdateInterval  time.Duration
	DeployURL       string

	// Cloudflare R2 deployment for the Status v2 JSON (all optional; empty
	// R2AccountID disables R2). Can run alongside DeployURL during migration.
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string

	// SheetRetention is how long a concluded war's sheets are kept before they
	// are pruned. Zero disables pruning.
	SheetRetention time.Duration

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

	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	r2Bucket := os.Getenv("R2_BUCKET")
	if r2AccountID != "" || r2AccessKeyID != "" || r2SecretAccessKey != "" || r2Bucket != "" {
		if r2AccountID == "" || r2AccessKeyID == "" || r2SecretAccessKey == "" || r2Bucket == "" {
			return nil, fmt.Errorf("R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY and R2_BUCKET must all be set to enable R2 deployment")
		}
	}

	bigQueryProjectID := os.Getenv("BIGQUERY_PROJECT_ID")
	bigQueryDatasetID := os.Getenv("BIGQUERY_DATASET_ID")
	bigQueryTableID := os.Getenv("BIGQUERY_TABLE_ID")
	if bigQueryTableID == "" {
		bigQueryTableID = "state_changes"
	}

	sheetRetention := war.DefaultSheetRetention
	if v := os.Getenv("SHEET_RETENTION_DAYS"); v != "" {
		days, err := strconv.Atoi(v)
		if err != nil || days < 0 {
			return nil, fmt.Errorf("SHEET_RETENTION_DAYS must be a non-negative integer, got %q", v)
		}
		sheetRetention = time.Duration(days) * 24 * time.Hour
	}

	return &Config{
		TornAPIKey:        apiKey,
		SpreadsheetID:     spreadsheetID,
		CredentialsFile:   credentialsFile,
		DeployURL:         deployURL,
		R2AccountID:       r2AccountID,
		R2AccessKeyID:     r2AccessKeyID,
		R2SecretAccessKey: r2SecretAccessKey,
		R2Bucket:          r2Bucket,
		BigQueryProjectID: bigQueryProjectID,
		BigQueryDatasetID: bigQueryDatasetID,
		BigQueryTableID:   bigQueryTableID,
		SheetRetention:    sheetRetention,
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
