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
	processor       *WarProcessor
	cachedClient    *CachedTornClient
	optimizer       *APIOptimizer
	tracker         *APICallTracker
	stateManager    *WarStateManager
	stateTracker    *StateTrackingService
	spreadsheetID   string
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
	stateManager := NewWarStateManager()
	cachedClient := NewCachedTornClientWithWarStateManager(tornClient, tracker, stateManager)
	optimizer := NewAPIOptimizer(cachedClient, tracker)

	// Create state tracking service with raw client for real-time faction data
	stateTracker := NewStateTrackingService(tornClient, sheetsClient)

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
		processor:     processor,
		cachedClient:  cachedClient,
		optimizer:     optimizer,
		tracker:       tracker,
		stateManager:  stateManager,
		stateTracker:  stateTracker,
		spreadsheetID: config.SpreadsheetID,
	}
}

// ProcessActiveWars processes wars with sophisticated state-based optimization
func (owp *OptimizedWarProcessor) ProcessActiveWars(ctx context.Context, force bool) error {
	// Always fetch war data first to determine actual current state
	log.Debug().
		Msg("Fetching war data to determine current state")

	warResponse, err := owp.cachedClient.GetFactionWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch wars for state analysis: %w", err)
	}

	// Update war state based on fresh data
	previousState := owp.stateManager.GetCurrentState()
	currentState := owp.stateManager.UpdateState(warResponse)

	// Log current state at start of processing loop
	stateInfo := owp.stateManager.GetStateInfo()
	log.Info().
		Str("current_state", stateInfo.State.String()).
		Str("state_description", stateInfo.Description).
		Dur("time_in_state", stateInfo.TimeInState).
		Dur("time_until_next_check", stateInfo.TimeUntilCheck).
		Msg("Starting war processor loop")

	// Now check if we should do full processing based on updated state
	if !force && !owp.stateManager.ShouldProcessNow() {
		log.Info().
			Str("current_state", stateInfo.State.String()).
			Dur("time_until_next_check", stateInfo.TimeUntilCheck).
			Msg("Skipping full processing - not time for next check")
		return nil
	}

	if force {
		log.Info().
			Str("current_state", stateInfo.State.String()).
			Msg("Force flag enabled - bypassing state-based optimization")
	}

	// Log pre-processing stats
	preStats := owp.tracker.GetSessionStats()
	log.Debug().
		Int64("session_calls_before", preStats.SessionCalls).
		Msg("API calls before processing")

	// Log state information
	stateInfo = owp.stateManager.GetStateInfo()
	log.Info().
		Str("war_state", currentState.String()).
		Str("description", stateInfo.Description).
		Dur("time_in_state", stateInfo.TimeInState).
		Dur("next_check_in", stateInfo.TimeUntilCheck).
		Bool("state_changed", previousState != currentState).
		Msg("War state analysis complete")

	// Ensure our faction ID is available for state tracking
	if err := owp.processor.ensureOurFactionID(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to ensure our faction ID - continuing without state tracking")
	}

	// Process state changes for all observed factions
	owp.processStateChanges(ctx, warResponse)

	// Handle different states
	switch currentState {
	case NoWars:
		if !force {
			log.Info().
				Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
				Msg("No active wars - processor will pause until next Tuesday matchmaking")
			return nil
		}
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("No active wars - but force flag enabled, processing our faction status only")

		// Process just our faction's status when no wars exist
		return owp.processOurFactionOnly(ctx)

	case PostWar:
		if !force {
			log.Info().
				Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
				Msg("War completed - processor will pause until next week's matchmaking")
			return nil
		}
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("War completed - but force flag enabled, continuing processing")

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

	// Only process if we have wars that need attention (PreWar or ActiveWar) or force flag is enabled
	if currentState == PreWar || currentState == ActiveWar || force {
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

// GetAPICallCount returns the current API call count
func (owp *OptimizedWarProcessor) GetAPICallCount() int64 {
	return owp.tracker.GetSessionStats().SessionCalls
}

// GetNextCheckTime returns when the next processing should occur based on current war state
func (owp *OptimizedWarProcessor) GetNextCheckTime() time.Time {
	return owp.stateManager.GetNextCheckTime()
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

// processOurFactionOnly processes just our faction's status when no wars exist
func (owp *OptimizedWarProcessor) processOurFactionOnly(ctx context.Context) error {
	log.Info().Msg("Processing our faction status only (no active wars)")

	// Ensure our faction ID is loaded
	if err := owp.processor.ensureOurFactionID(ctx); err != nil {
		return fmt.Errorf("failed to initialize faction ID: %w", err)
	}

	ourFactionID := owp.processor.ourFactionID
	if ourFactionID == 0 {
		return fmt.Errorf("our faction ID is not set")
	}

	// Create a dummy war structure for travel processing (needed for the existing processTravelStatus method)
	dummyWar := &app.War{
		ID: 0, // Use 0 to indicate "no war"
		Factions: []app.Faction{
			{ID: ourFactionID}, // Just our faction
		},
	}

	// Process our faction's travel status using existing method
	if err := owp.processor.processTravelStatus(ctx, dummyWar, ourFactionID, owp.processor.config.SpreadsheetID); err != nil {
		return fmt.Errorf("failed to process our faction travel status: %w", err)
	}

	log.Info().
		Int("faction_id", ourFactionID).
		Msg("Successfully processed our faction status")

	return nil
}

// processStateChanges handles state tracking for all observed factions
func (owp *OptimizedWarProcessor) processStateChanges(ctx context.Context, warResponse *app.WarResponse) {
	// Determine which factions to track based on current wars
	var factionIDs []int

	// Add our faction ID if available
	if owp.processor.ourFactionID != 0 {
		factionIDs = append(factionIDs, owp.processor.ourFactionID)
	}

	// Add faction IDs from active wars
	if warResponse.Wars.Ranked != nil {
		for _, faction := range warResponse.Wars.Ranked.Factions {
			factionIDs = append(factionIDs, faction.ID)
		}
	}

	// Add faction IDs from raid wars
	for _, war := range warResponse.Wars.Raids {
		for _, faction := range war.Factions {
			factionIDs = append(factionIDs, faction.ID)
		}
	}

	// Add faction IDs from territory wars
	for _, war := range warResponse.Wars.Territory {
		for _, faction := range war.Factions {
			factionIDs = append(factionIDs, faction.ID)
		}
	}

	// Remove duplicates
	factionIDs = owp.removeDuplicateFactionIDs(factionIDs)

	// If no factions to track, skip
	if len(factionIDs) == 0 {
		log.Debug().Msg("No factions to track for state changes")
		return
	}

	// Process state changes
	log.Debug().
		Ints("faction_ids", factionIDs).
		Msg("Processing state changes for factions")

	if err := owp.stateTracker.ProcessStateChanges(ctx, owp.spreadsheetID, factionIDs); err != nil {
		log.Error().
			Err(err).
			Ints("faction_ids", factionIDs).
			Msg("Failed to process state changes - continuing with main processing")
	} else {
		log.Debug().
			Ints("faction_ids", factionIDs).
			Msg("Successfully processed state changes")
	}
}

// removeDuplicateFactionIDs removes duplicate faction IDs from a slice
func (owp *OptimizedWarProcessor) removeDuplicateFactionIDs(factionIDs []int) []int {
	seen := make(map[int]bool)
	var result []int

	for _, id := range factionIDs {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
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
