package attack

import (
	"time"
)

// FetchStrategy describes how to fetch attacks from the Torn API
type FetchStrategy struct {
	Method     FetchMethod
	TimeRange  TimeRange
	Pagination PaginationConfig
}

// FetchMethod describes the approach for fetching attacks
type FetchMethod string

const (
	// FetchMethodSimple uses a single API call for fetching attacks
	FetchMethodSimple FetchMethod = "simple"
	// FetchMethodPaginated uses backwards pagination for large time ranges
	FetchMethodPaginated FetchMethod = "paginated"
)

// TimeRange represents a time range for fetching
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// PaginationConfig contains configuration for paginated fetching
type PaginationConfig struct {
	Enabled      bool
	MaxPages     int
	StopOnGap    bool
	GapThreshold time.Duration
}

// DetermineFetchStrategy decides how to fetch attacks based on time range
func DetermineFetchStrategy(startTime, endTime time.Time) FetchStrategy {
	strategy := FetchStrategy{
		TimeRange: TimeRange{Start: startTime, End: endTime},
	}

	// Use simple approach for time ranges under 24 hours for incremental updates
	if ShouldUseSimpleApproach(startTime, endTime) {
		strategy.Method = FetchMethodSimple
		strategy.Pagination = PaginationConfig{Enabled: false}
	} else {
		strategy.Method = FetchMethodPaginated
		strategy.Pagination = PaginationConfig{
			Enabled:      true,
			MaxPages:     100,
			StopOnGap:    true,
			GapThreshold: 5 * time.Minute,
		}
	}

	return strategy
}

// ShouldUseSimpleApproach determines if simple fetching is appropriate
func ShouldUseSimpleApproach(startTime, endTime time.Time) bool {
	duration := endTime.Sub(startTime)

	// For time ranges 24 hours or less, use simple approach
	const maxSimpleRange = 24 * time.Hour
	return duration <= maxSimpleRange
}

// EstimateAPICallsRequired estimates how many API calls will be needed
func EstimateAPICallsRequired(strategy FetchStrategy) int {
	switch strategy.Method {
	case FetchMethodSimple:
		return 1
	case FetchMethodPaginated:
		// Conservative estimate based on typical war activity
		duration := strategy.TimeRange.End.Sub(strategy.TimeRange.Start)
		hoursInRange := int(duration.Hours())
		// Estimate ~10 attacks per hour, 100 attacks per page
		estimatedPages := (hoursInRange * 10) / 100
		if estimatedPages < 1 {
			estimatedPages = 1
		}
		return estimatedPages
	default:
		return 0
	}
}
