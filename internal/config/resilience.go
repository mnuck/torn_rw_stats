package config

import "time"

// Retry configuration constants
const (
	// API Request retry configuration
	APIRequestMaxAttempts       = 3
	APIRequestInitialWait       = 1 * time.Second
	APIRequestMaxWait           = 10 * time.Second
	APIRequestBackoffMultiplier = 2.0
	APIRequestTimeout           = 30 * time.Second

	// Sheet Read retry configuration
	SheetReadMaxAttempts       = 3
	SheetReadInitialWait       = 500 * time.Millisecond
	SheetReadMaxWait           = 5 * time.Second
	SheetReadBackoffMultiplier = 2.0
	SheetReadTimeout           = 30 * time.Second

	// Sheet Write retry configuration
	SheetWriteMaxAttempts       = 3
	SheetWriteInitialWait       = 1 * time.Second
	SheetWriteMaxWait           = 10 * time.Second
	SheetWriteBackoffMultiplier = 2.0
	SheetWriteTimeout           = 30 * time.Second
)

// RetryConfig defines retry behavior for operations
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
	Timeout     time.Duration
}

// ResilienceConfig contains all retry configurations
type ResilienceConfig struct {
	APIRequest RetryConfig
	SheetRead  RetryConfig
	SheetWrite RetryConfig
}

// DefaultResilienceConfig provides sensible defaults
var DefaultResilienceConfig = ResilienceConfig{
	APIRequest: RetryConfig{
		MaxAttempts: APIRequestMaxAttempts,
		InitialWait: APIRequestInitialWait,
		MaxWait:     APIRequestMaxWait,
		Multiplier:  APIRequestBackoffMultiplier,
		Timeout:     APIRequestTimeout,
	},
	SheetRead: RetryConfig{
		MaxAttempts: SheetReadMaxAttempts,
		InitialWait: SheetReadInitialWait,
		MaxWait:     SheetReadMaxWait,
		Multiplier:  SheetReadBackoffMultiplier,
		Timeout:     SheetReadTimeout,
	},
	SheetWrite: RetryConfig{
		MaxAttempts: SheetWriteMaxAttempts,
		InitialWait: SheetWriteInitialWait,
		MaxWait:     SheetWriteMaxWait,
		Multiplier:  SheetWriteBackoffMultiplier,
		Timeout:     SheetWriteTimeout,
	},
}
