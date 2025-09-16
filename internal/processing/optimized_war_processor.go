package processing

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// OptimizedWarProcessor wraps WarProcessor with war state management
type OptimizedWarProcessor struct {
	processor         *WarProcessor
	tornClient        TornClientInterface
	tracker           *APICallTracker
	stateManager      *WarStateManager
	stateTracker      *StateTrackingService
	statusV2Processor *StatusV2Processor
	spreadsheetID     string
	config            *app.Config
}

// NewOptimizedWarProcessor creates a WarProcessor with war state management
func NewOptimizedWarProcessor(
	tornClient TornClientInterface,
	sheetsClient SheetsClientInterface,
	locationService LocationServiceInterface,
	travelTimeService TravelTimeServiceInterface,
	attackService AttackProcessingServiceInterface,
	warSummaryService WarSummaryServiceInterface,
	config *app.Config,
) *OptimizedWarProcessor {

	// Create war state management
	tracker := NewAPICallTracker()
	stateManager := NewWarStateManager()

	// Create state tracking service with raw client
	stateTracker := NewStateTrackingService(tornClient, sheetsClient)

	// Create Status v2 processor
	statusV2Processor := NewStatusV2Processor(tornClient, sheetsClient, config.OurFactionID)

	// Create processor with raw client
	processor := NewWarProcessor(
		tornClient,
		sheetsClient,
		locationService,
		travelTimeService,
		attackService,
		warSummaryService,
		config,
	)

	return &OptimizedWarProcessor{
		processor:         processor,
		tornClient:        tornClient,
		tracker:           tracker,
		stateManager:      stateManager,
		stateTracker:      stateTracker,
		statusV2Processor: statusV2Processor,
		spreadsheetID:     config.SpreadsheetID,
		config:            config,
	}
}

// ProcessActiveWars processes wars with continuous monitoring
func (owp *OptimizedWarProcessor) ProcessActiveWars(ctx context.Context) error {
	// Always fetch war data first to determine actual current state
	log.Debug().
		Msg("Fetching war data to determine current state")

	warResponse, err := owp.tornClient.GetFactionWars(ctx)
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

	// Continuous monitoring enabled - always process all states
	log.Debug().
		Str("current_state", stateInfo.State.String()).
		Msg("Continuous monitoring enabled - processing all states")

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
	owp.processStateChanges(ctx, warResponse, stateInfo)

	// Handle different states
	switch currentState {
	case NoWars:
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("No active wars - processing our faction status only")

		// Process just our faction's status when no wars exist
		return owp.processOurFactionOnly(ctx)

	case PostWar:
		log.Info().
			Time("next_matchmaking", owp.stateManager.GetNextCheckTime()).
			Msg("War completed - continuing processing for post-war analysis")

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

	// Process wars for PreWar and ActiveWar states (NoWars and PostWar are handled above)
	if currentState == PreWar || currentState == ActiveWar {
		// Process wars using existing logic but with optimized client
		owp.processor.ourFactionID = 0 // Reset to ensure faction ID is fetched if needed
		err = owp.processor.ProcessActiveWars(ctx)
		if err != nil {
			return fmt.Errorf("failed to process wars: %w", err)
		}
	}

	// Log processing results
	owp.LogProcessingResults(ctx)

	return nil
}

// LogProcessingResults logs the processing session results
func (owp *OptimizedWarProcessor) LogProcessingResults(ctx context.Context) {
	// Get current session stats
	owp.tracker.LogSessionSummary(ctx)
}

// GetAPICallCount returns the current API call count
func (owp *OptimizedWarProcessor) GetAPICallCount() int64 {
	return owp.tracker.GetSessionStats().SessionCalls
}

// GetNextCheckTime returns when the next processing should occur based on current war state
func (owp *OptimizedWarProcessor) GetNextCheckTime() time.Time {
	return owp.stateManager.GetNextCheckTime()
}

// GetProcessingSummary returns a summary of processing session
func (owp *OptimizedWarProcessor) GetProcessingSummary() ProcessingSummary {
	stats := owp.tracker.GetSessionStats()

	return ProcessingSummary{
		SessionAPICalls: stats.SessionCalls,
		TotalAPICalls:   stats.TotalCalls,
		CallsPerMinute:  stats.CallsPerMinute,
		SessionDuration: stats.SessionDuration,
	}
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

	log.Info().
		Int("faction_id", ourFactionID).
		Msg("Successfully processed our faction status")

	return nil
}

// processStateChanges handles state tracking for all observed factions
func (owp *OptimizedWarProcessor) processStateChanges(ctx context.Context, warResponse *app.WarResponse, stateInfo WarStateInfo) {
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

	// Process Status v2 sheets for all factions
	log.Debug().
		Ints("faction_ids", factionIDs).
		Msg("Processing Status v2 for factions")

	if err := owp.statusV2Processor.ProcessStatusV2ForFactions(ctx, owp.spreadsheetID, factionIDs, owp.config.UpdateInterval); err != nil {
		log.Error().
			Err(err).
			Ints("faction_ids", factionIDs).
			Msg("Failed to process Status v2 - continuing with main processing")
	} else {
		log.Debug().
			Ints("faction_ids", factionIDs).
			Msg("Successfully processed Status v2")
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

// ProcessingSummary provides a summary of processing session
type ProcessingSummary struct {
	SessionAPICalls int64
	TotalAPICalls   int64
	CallsPerMinute  float64
	SessionDuration time.Duration
}
