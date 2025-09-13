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
	processor    *WarProcessor
	cachedClient *CachedTornClient
	optimizer    *APIOptimizer
	tracker      *APICallTracker
	stateManager *WarStateManager
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
	stateManager := NewWarStateManager()

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
		processor:    processor,
		cachedClient: cachedClient,
		optimizer:    optimizer,
		tracker:      tracker,
		stateManager: stateManager,
	}
}

// ProcessActiveWars processes wars with sophisticated state-based optimization
func (owp *OptimizedWarProcessor) ProcessActiveWars(ctx context.Context) error {
	// Log current state at start of processing loop
	stateInfo := owp.stateManager.GetStateInfo()
	log.Info().
		Str("current_state", stateInfo.State.String()).
		Str("state_description", stateInfo.Description).
		Dur("time_in_state", stateInfo.TimeInState).
		Dur("time_until_next_check", stateInfo.TimeUntilCheck).
		Msg("Starting war processor loop")

	// Check if we should process now based on current state
	if !owp.stateManager.ShouldProcessNow() {
		log.Info().
			Str("current_state", stateInfo.State.String()).
			Dur("time_until_next_check", stateInfo.TimeUntilCheck).
			Msg("Skipping processing - not time for next check")
		return nil
	}

	// Log pre-processing stats
	preStats := owp.tracker.GetSessionStats()
	log.Debug().
		Int64("session_calls_before", preStats.SessionCalls).
		Msg("API calls before processing")

	// Fetch war data to determine current state
	warResponse, err := owp.cachedClient.GetFactionWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch wars for state analysis: %w", err)
	}

	// Update war state based on current data
	previousState := owp.stateManager.GetCurrentState()
	currentState := owp.stateManager.UpdateState(warResponse)

	// Log state information
	stateInfo = owp.stateManager.GetStateInfo()
	log.Info().
		Str("war_state", currentState.String()).
		Str("description", stateInfo.Description).
		Dur("time_in_state", stateInfo.TimeInState).
		Dur("next_check_in", stateInfo.TimeUntilCheck).
		Bool("state_changed", previousState != currentState).
		Msg("War state analysis complete")

	// Handle different states
	switch currentState {
	case NoWars:
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("No active wars - processor will pause until next Tuesday matchmaking")
		return nil

	case PostWar:
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("War completed - processor will pause until next week's matchmaking")
		return nil

	case PreWar:
		log.Info().
			Dur("update_interval", stateInfo.UpdateInterval).
			Msg("Pre-war reconnaissance mode - monitoring opponent")
		// Continue to processing for reconnaissance data

	case ActiveWar:
		log.Info().
			Dur("update_interval", stateInfo.UpdateInterval).
			Msg("Active war detected - real-time monitoring enabled")
		// Continue to full processing
	}

	// Only process if we have wars that need attention (PreWar or ActiveWar)
	if currentState == PreWar || currentState == ActiveWar {
		// Process wars using existing logic but with optimized client
		owp.processor.ourFactionID = 0 // Reset to ensure faction ID is fetched if needed
		err = owp.processor.ProcessActiveWars(ctx)
		if err != nil {
			return fmt.Errorf("failed to process wars: %w", err)
		}
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
