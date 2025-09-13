package processing

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// OptimizedWarProcessor wraps WarProcessor with API call optimizations
type OptimizedWarProcessor struct {
	processor      *WarProcessor
	cachedClient   *CachedTornClient
	optimizer      *APIOptimizer
	tracker        *APICallTracker
}

// NewOptimizedWarProcessor creates a WarProcessor with API optimizations enabled
func NewOptimizedWarProcessor(
	tornClient TornClientInterface,
	sheetsClient SheetsClientInterface,
	locationService LocationServiceInterface,
	travelTimeService TravelTimeServiceInterface,
	attackService AttackProcessingServiceInterface,
	warSummaryService WarSummaryServiceInterface,
	stateChangeService StateChangeDetectionServiceInterface,
	config *app.Config,
) *OptimizedWarProcessor {

	// Create optimization layer
	tracker := NewAPICallTracker()
	cachedClient := NewCachedTornClient(tornClient, tracker)
	optimizer := NewAPIOptimizer(cachedClient, tracker)

	// Create processor with optimized client
	processor := NewWarProcessor(
		cachedClient, // Use cached client instead of raw client
		sheetsClient,
		locationService,
		travelTimeService,
		attackService,
		warSummaryService,
		stateChangeService,
		config,
	)

	return &OptimizedWarProcessor{
		processor:      processor,
		cachedClient:   cachedClient,
		optimizer:      optimizer,
		tracker:        tracker,
	}
}

// ProcessActiveWars processes wars with API call optimizations
func (owp *OptimizedWarProcessor) ProcessActiveWars(ctx context.Context) error {
	log.Info().Msg("Processing active wars with API optimizations")

	// Log pre-processing stats
	preStats := owp.tracker.GetSessionStats()
	log.Debug().
		Int64("session_calls_before", preStats.SessionCalls).
		Msg("API calls before processing")

	// Use optimized war fetching
	warResponse, err := owp.optimizer.GetOptimizedWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch optimized wars: %w", err)
	}

	// Check if this was an optimized skip (empty response but no API call)
	postFetchStats := owp.tracker.GetSessionStats()
	if postFetchStats.SessionCalls == preStats.SessionCalls && owp.hasActiveWars(warResponse) == false {
		log.Info().Msg("Skipped war check due to optimization - no active wars expected")
		return nil
	}

	// Process wars using existing logic but with optimized client
	owp.processor.ourFactionID = 0 // Reset to ensure faction ID is fetched if needed
	err = owp.processor.ProcessActiveWars(ctx)
	if err != nil {
		return err
	}

	// Log optimization results
	owp.LogOptimizationResults(ctx)

	return nil
}

// hasActiveWars checks if the war response contains any active wars
func (owp *OptimizedWarProcessor) hasActiveWars(warResponse *app.WarResponse) bool {
	if warResponse.Wars.Ranked != nil {
		return true
	}
	if len(warResponse.Wars.Raids) > 0 {
		return true
	}
	if len(warResponse.Wars.Territory) > 0 {
		return true
	}
	return false
}

// LogOptimizationResults logs the impact of API optimizations
func (owp *OptimizedWarProcessor) LogOptimizationResults(ctx context.Context) {
	// Get current session stats
	owp.tracker.LogSessionSummary(ctx)

	// Get cache statistics
	cacheStats := owp.cachedClient.GetCacheStats()
	log.Info().
		Int("cache_valid_entries", cacheStats.ValidEntries).
		Int("cache_expired_entries", cacheStats.ExpiredEntries).
		Int("cache_total_entries", cacheStats.TotalEntries).
		Msg("API cache statistics")

	// Get optimizer statistics
	optimizerStats := owp.optimizer.GetOptimizationStats()
	log.Info().
		Int("known_active_wars", optimizerStats.KnownActiveWars).
		Int("consecutive_empty_checks", optimizerStats.ConsecutiveEmpty).
		Dur("next_check_interval", optimizerStats.NextCheckInterval).
		Msg("API optimizer statistics")

	// Estimate calls for next period
	estimate := owp.optimizer.EstimateCallsForPeriod(optimizerStats.NextCheckInterval)
	log.Info().
		Int64("estimated_war_checks", estimate.WarChecks).
		Int64("estimated_attack_calls", estimate.AttackCalls).
		Int64("estimated_total_calls", estimate.TotalEstimate).
		Dur("for_period", estimate.Period).
		Msg("API call estimate for next period")
}

// ResetSession resets optimization tracking for a new session
func (owp *OptimizedWarProcessor) ResetSession() {
	owp.tracker.ResetSession()
	log.Info().Msg("API optimization session reset")
}

// GetAPICallCount returns the current API call count
func (owp *OptimizedWarProcessor) GetAPICallCount() int64 {
	return owp.tracker.GetSessionStats().SessionCalls
}

// ClearCache manually clears all cached data
func (owp *OptimizedWarProcessor) ClearCache() {
	owp.cachedClient.ClearCache()
	log.Info().Msg("API cache manually cleared")
}

// GetOptimizationSummary returns a summary of optimization effectiveness
func (owp *OptimizedWarProcessor) GetOptimizationSummary() OptimizationSummary {
	stats := owp.tracker.GetSessionStats()
	cacheStats := owp.cachedClient.GetCacheStats()
	optimizerStats := owp.optimizer.GetOptimizationStats()

	return OptimizationSummary{
		SessionAPICalls:      stats.SessionCalls,
		TotalAPICalls:        stats.TotalCalls,
		CallsPerMinute:       stats.CallsPerMinute,
		SessionDuration:      stats.SessionDuration,
		CacheHitRate:         calculateCacheHitRate(stats.CallsByEndpoint, cacheStats),
		ActiveWarsDetected:   optimizerStats.KnownActiveWars,
		ConsecutiveEmptyRuns: optimizerStats.ConsecutiveEmpty,
		NextCheckIn:          optimizerStats.NextCheckInterval,
	}
}

// calculateCacheHitRate estimates cache effectiveness based on call patterns
func calculateCacheHitRate(callsByEndpoint map[string]int64, cacheStats CacheStats) float64 {
	if cacheStats.TotalEntries == 0 {
		return 0.0
	}

	// Rough estimation: valid cache entries vs total potential cacheable calls
	cacheableCalls := callsByEndpoint["GetOwnFaction"] + callsByEndpoint["GetFactionWars"] + callsByEndpoint["GetFactionBasic"]
	if cacheableCalls == 0 {
		return 0.0
	}

	// This is a simplified calculation - in reality we'd track cache hits vs misses
	return float64(cacheStats.ValidEntries) / float64(cacheableCalls+int64(cacheStats.ValidEntries)) * 100
}

// OptimizationSummary provides a summary of API optimization effectiveness
type OptimizationSummary struct {
	SessionAPICalls      int64
	TotalAPICalls        int64
	CallsPerMinute       float64
	SessionDuration      time.Duration
	CacheHitRate         float64
	ActiveWarsDetected   int
	ConsecutiveEmptyRuns int
	NextCheckIn          time.Duration
}