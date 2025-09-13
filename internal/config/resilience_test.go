package config

import (
	"testing"
	"time"
)

func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 5,
		InitialWait: 2 * time.Second,
		MaxWait:     30 * time.Second,
		Multiplier:  3.0,
		Timeout:     60 * time.Second,
	}

	if config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", config.MaxAttempts)
	}

	if config.InitialWait != 2*time.Second {
		t.Errorf("Expected InitialWait 2s, got %v", config.InitialWait)
	}

	if config.MaxWait != 30*time.Second {
		t.Errorf("Expected MaxWait 30s, got %v", config.MaxWait)
	}

	if config.Multiplier != 3.0 {
		t.Errorf("Expected Multiplier 3.0, got %f", config.Multiplier)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout 60s, got %v", config.Timeout)
	}
}

func TestResilienceConfig(t *testing.T) {
	config := ResilienceConfig{
		APIRequest: RetryConfig{MaxAttempts: 3},
		SheetRead:  RetryConfig{MaxAttempts: 2},
		SheetWrite: RetryConfig{MaxAttempts: 4},
	}

	if config.APIRequest.MaxAttempts != 3 {
		t.Errorf("Expected APIRequest MaxAttempts 3, got %d", config.APIRequest.MaxAttempts)
	}

	if config.SheetRead.MaxAttempts != 2 {
		t.Errorf("Expected SheetRead MaxAttempts 2, got %d", config.SheetRead.MaxAttempts)
	}

	if config.SheetWrite.MaxAttempts != 4 {
		t.Errorf("Expected SheetWrite MaxAttempts 4, got %d", config.SheetWrite.MaxAttempts)
	}
}

func TestDefaultResilienceConfig(t *testing.T) {
	// Test that DefaultResilienceConfig has expected values
	if DefaultResilienceConfig.APIRequest.MaxAttempts != 3 {
		t.Errorf("Expected default APIRequest MaxAttempts 3, got %d", DefaultResilienceConfig.APIRequest.MaxAttempts)
	}

	if DefaultResilienceConfig.APIRequest.InitialWait != 1*time.Second {
		t.Errorf("Expected default APIRequest InitialWait 1s, got %v", DefaultResilienceConfig.APIRequest.InitialWait)
	}

	if DefaultResilienceConfig.APIRequest.MaxWait != 10*time.Second {
		t.Errorf("Expected default APIRequest MaxWait 10s, got %v", DefaultResilienceConfig.APIRequest.MaxWait)
	}

	if DefaultResilienceConfig.APIRequest.Multiplier != 2.0 {
		t.Errorf("Expected default APIRequest Multiplier 2.0, got %f", DefaultResilienceConfig.APIRequest.Multiplier)
	}

	if DefaultResilienceConfig.APIRequest.Timeout != 30*time.Second {
		t.Errorf("Expected default APIRequest Timeout 30s, got %v", DefaultResilienceConfig.APIRequest.Timeout)
	}

	// Test SheetRead defaults
	if DefaultResilienceConfig.SheetRead.MaxAttempts != 3 {
		t.Errorf("Expected default SheetRead MaxAttempts 3, got %d", DefaultResilienceConfig.SheetRead.MaxAttempts)
	}

	if DefaultResilienceConfig.SheetRead.InitialWait != 500*time.Millisecond {
		t.Errorf("Expected default SheetRead InitialWait 500ms, got %v", DefaultResilienceConfig.SheetRead.InitialWait)
	}

	if DefaultResilienceConfig.SheetRead.MaxWait != 5*time.Second {
		t.Errorf("Expected default SheetRead MaxWait 5s, got %v", DefaultResilienceConfig.SheetRead.MaxWait)
	}

	if DefaultResilienceConfig.SheetRead.Multiplier != 2.0 {
		t.Errorf("Expected default SheetRead Multiplier 2.0, got %f", DefaultResilienceConfig.SheetRead.Multiplier)
	}

	if DefaultResilienceConfig.SheetRead.Timeout != 30*time.Second {
		t.Errorf("Expected default SheetRead Timeout 30s, got %v", DefaultResilienceConfig.SheetRead.Timeout)
	}

	// Test SheetWrite defaults
	if DefaultResilienceConfig.SheetWrite.MaxAttempts != 3 {
		t.Errorf("Expected default SheetWrite MaxAttempts 3, got %d", DefaultResilienceConfig.SheetWrite.MaxAttempts)
	}

	if DefaultResilienceConfig.SheetWrite.InitialWait != 1*time.Second {
		t.Errorf("Expected default SheetWrite InitialWait 1s, got %v", DefaultResilienceConfig.SheetWrite.InitialWait)
	}

	if DefaultResilienceConfig.SheetWrite.MaxWait != 10*time.Second {
		t.Errorf("Expected default SheetWrite MaxWait 10s, got %v", DefaultResilienceConfig.SheetWrite.MaxWait)
	}

	if DefaultResilienceConfig.SheetWrite.Multiplier != 2.0 {
		t.Errorf("Expected default SheetWrite Multiplier 2.0, got %f", DefaultResilienceConfig.SheetWrite.Multiplier)
	}

	if DefaultResilienceConfig.SheetWrite.Timeout != 30*time.Second {
		t.Errorf("Expected default SheetWrite Timeout 30s, got %v", DefaultResilienceConfig.SheetWrite.Timeout)
	}
}

func TestDefaultResilienceConfigImmutability(t *testing.T) {
	// Test that modifying the returned config doesn't affect the default
	original := DefaultResilienceConfig

	// Create a copy and modify it
	modified := DefaultResilienceConfig
	modified.APIRequest.MaxAttempts = 999

	// Verify original is unchanged
	if DefaultResilienceConfig.APIRequest.MaxAttempts != original.APIRequest.MaxAttempts {
		t.Error("DefaultResilienceConfig was unexpectedly modified")
	}

	if DefaultResilienceConfig.APIRequest.MaxAttempts == 999 {
		t.Error("DefaultResilienceConfig should not have been modified")
	}
}

func TestRetryConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		config RetryConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: RetryConfig{
				MaxAttempts: 3,
				InitialWait: 1 * time.Second,
				MaxWait:     10 * time.Second,
				Multiplier:  2.0,
				Timeout:     30 * time.Second,
			},
			valid: true,
		},
		{
			name: "zero max attempts",
			config: RetryConfig{
				MaxAttempts: 0,
				InitialWait: 1 * time.Second,
				MaxWait:     10 * time.Second,
				Multiplier:  2.0,
				Timeout:     30 * time.Second,
			},
			valid: false,
		},
		{
			name: "negative multiplier",
			config: RetryConfig{
				MaxAttempts: 3,
				InitialWait: 1 * time.Second,
				MaxWait:     10 * time.Second,
				Multiplier:  -1.0,
				Timeout:     30 * time.Second,
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic validation logic - checking for reasonable values
			isValid := tc.config.MaxAttempts > 0 &&
				tc.config.InitialWait >= 0 &&
				tc.config.MaxWait >= tc.config.InitialWait &&
				tc.config.Multiplier > 0 &&
				tc.config.Timeout > 0

			if isValid != tc.valid {
				t.Errorf("Expected validity %v, got %v for config %+v", tc.valid, isValid, tc.config)
			}
		})
	}
}