package app

import (
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalTornAPIKey := os.Getenv("TORN_API_KEY")
	originalSpreadsheetID := os.Getenv("SPREADSHEET_ID")
	originalCredentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")

	// Cleanup function
	defer func() {
		setOrUnset("TORN_API_KEY", originalTornAPIKey)
		setOrUnset("SPREADSHEET_ID", originalSpreadsheetID)
		setOrUnset("GOOGLE_CREDENTIALS_FILE", originalCredentialsFile)
	}()

	t.Run("ValidConfiguration", func(t *testing.T) {
		os.Setenv("TORN_API_KEY", "test_api_key")
		os.Setenv("SPREADSHEET_ID", "test_spreadsheet_id")
		os.Setenv("GOOGLE_CREDENTIALS_FILE", "test_credentials.json")

		config, err := LoadConfig()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.TornAPIKey != "test_api_key" {
			t.Errorf("Expected TornAPIKey to be 'test_api_key', got '%s'", config.TornAPIKey)
		}

		if config.SpreadsheetID != "test_spreadsheet_id" {
			t.Errorf("Expected SpreadsheetID to be 'test_spreadsheet_id', got '%s'", config.SpreadsheetID)
		}

		if config.CredentialsFile != "test_credentials.json" {
			t.Errorf("Expected CredentialsFile to be 'test_credentials.json', got '%s'", config.CredentialsFile)
		}
	})

	t.Run("DefaultCredentialsFile", func(t *testing.T) {
		os.Setenv("TORN_API_KEY", "test_api_key")
		os.Setenv("SPREADSHEET_ID", "test_spreadsheet_id")
		os.Unsetenv("GOOGLE_CREDENTIALS_FILE")

		config, err := LoadConfig()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.CredentialsFile != "credentials.json" {
			t.Errorf("Expected CredentialsFile to default to 'credentials.json', got '%s'", config.CredentialsFile)
		}
	})

	t.Run("MissingTornAPIKey", func(t *testing.T) {
		os.Unsetenv("TORN_API_KEY")
		os.Setenv("SPREADSHEET_ID", "test_spreadsheet_id")

		_, err := LoadConfig()

		if err == nil {
			t.Fatal("Expected error for missing TORN_API_KEY, got nil")
		}

		if !strings.Contains(err.Error(), "TORN_API_KEY") {
			t.Errorf("Expected error message to contain 'TORN_API_KEY', got '%s'", err.Error())
		}
	})

	t.Run("MissingSpreadsheetID", func(t *testing.T) {
		os.Setenv("TORN_API_KEY", "test_api_key")
		os.Unsetenv("SPREADSHEET_ID")

		_, err := LoadConfig()

		if err == nil {
			t.Fatal("Expected error for missing SPREADSHEET_ID, got nil")
		}

		if !strings.Contains(err.Error(), "SPREADSHEET_ID") {
			t.Errorf("Expected error message to contain 'SPREADSHEET_ID', got '%s'", err.Error())
		}
	})
}

func TestSetupEnvironment(t *testing.T) {
	// Save original environment
	originalENV := os.Getenv("ENV")
	originalLOGLEVEL := os.Getenv("LOGLEVEL")
	originalLevel := zerolog.GlobalLevel()

	// Cleanup function
	defer func() {
		setOrUnset("ENV", originalENV)
		setOrUnset("LOGLEVEL", originalLOGLEVEL)
		zerolog.SetGlobalLevel(originalLevel)
	}()

	testCases := []struct {
		name           string
		env            string
		logLevel       string
		expectedLevel  zerolog.Level
	}{
		{"ProductionDebug", "production", "debug", zerolog.DebugLevel},
		{"ProductionInfo", "production", "info", zerolog.InfoLevel},
		{"ProductionWarn", "production", "warn", zerolog.WarnLevel},
		{"ProductionWarning", "production", "warning", zerolog.WarnLevel},
		{"ProductionError", "production", "error", zerolog.ErrorLevel},
		{"ProductionFatal", "production", "fatal", zerolog.FatalLevel},
		{"ProductionPanic", "production", "panic", zerolog.PanicLevel},
		{"ProductionDisabled", "production", "disabled", zerolog.Disabled},
		{"ProductionDefault", "production", "", zerolog.WarnLevel},
		{"ProductionUnknown", "production", "unknown", zerolog.InfoLevel},
		{"DevelopmentDebug", "development", "debug", zerolog.DebugLevel},
		{"DevelopmentDefault", "development", "", zerolog.InfoLevel},
		{"DevelopmentUnknown", "", "unknown", zerolog.InfoLevel},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset("ENV", tc.env)
			setOrUnset("LOGLEVEL", tc.logLevel)

			SetupEnvironment()

			if zerolog.GlobalLevel() != tc.expectedLevel {
				t.Errorf("Expected log level %v, got %v", tc.expectedLevel, zerolog.GlobalLevel())
			}
		})
	}
}

func TestGetRequiredEnv(t *testing.T) {
	// Save original environment
	originalValue := os.Getenv("TEST_REQUIRED_VAR")

	// Cleanup function
	defer func() {
		setOrUnset("TEST_REQUIRED_VAR", originalValue)
	}()

	t.Run("ExistingVariable", func(t *testing.T) {
		os.Setenv("TEST_REQUIRED_VAR", "test_value")

		value := GetRequiredEnv("TEST_REQUIRED_VAR")

		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}
	})

	t.Run("MissingVariable", func(t *testing.T) {
		os.Unsetenv("TEST_REQUIRED_VAR")

		// This function calls log.Fatal() which would exit the process
		// We can't easily test this without complex setup, so we skip it
		// In a real scenario, you might use dependency injection for the logger
		t.Skip("Cannot test log.Fatal() without complex test setup")
	})
}

// Helper function to set environment variable or unset if value is empty
func setOrUnset(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}