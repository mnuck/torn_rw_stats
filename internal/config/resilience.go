package config

import "time"

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
	APIRequest   RetryConfig
	SheetRead    RetryConfig
	SheetWrite   RetryConfig
}

// DefaultResilienceConfig provides sensible defaults
var DefaultResilienceConfig = ResilienceConfig{
	APIRequest: RetryConfig{
		MaxAttempts: 3,
		InitialWait: 1 * time.Second,
		MaxWait:     10 * time.Second,
		Multiplier:  2.0,
		Timeout:     30 * time.Second,
	},
	SheetRead: RetryConfig{
		MaxAttempts: 3,
		InitialWait: 500 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
		Timeout:     30 * time.Second,
	},
	SheetWrite: RetryConfig{
		MaxAttempts: 3,
		InitialWait: 1 * time.Second,
		MaxWait:     10 * time.Second,
		Multiplier:  2.0,
		Timeout:     30 * time.Second,
	},
}