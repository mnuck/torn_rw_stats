package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/war"
	"log/slog"

	"torn_rw_stats/internal/processing"
)

// OptimizedWarProcessor wraps WarProcessor with intelligent war state management,
// adapting API call frequency based on war phases and Tuesday matchmaking schedules.
type OptimizedWarProcessor struct {
	processor         *WarProcessor
	tornClient        processing.TornClientInterface
	tracker           *APICallTracker
	stateManager      *war.WarStateManager
	stateTracker      *StateTrackingService
	statusV2Processor *StatusV2Processor
	spreadsheetID     string
	config            *app.Config
}

// NewOptimizedWarProcessor creates a WarProcessor with war state management
func NewOptimizedWarProcessor(
	tornClient processing.TornClientInterface,
	sheetsClient processing.SheetsClientInterface,
	locationService processing.LocationServiceInterface,
	travelTimeService processing.TravelTimeServiceInterface,
	attackService processing.AttackProcessingServiceInterface,
	warSummaryService processing.WarSummaryServiceInterface,
	config *app.Config,
	bqClient processing.BigQueryClientInterface,
) *OptimizedWarProcessor {

	// Create war state management
	tracker := NewAPICallTracker()
	stateManager := war.NewWarStateManager()

	// Create state tracking service
	stateTracker := NewStateTrackingService(tornClient, bqClient)

	// Create Status v2 processor
	statusV2Processor := NewStatusV2Processor(tornClient, sheetsClient, bqClient, config.DeployURL)

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
	slog.Debug("Fetching war data to determine current state")

	warResponse, err := owp.tornClient.GetFactionWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch wars for state analysis: %w", err)
	}

	// Update war state based on fresh data
	previousState := owp.stateManager.GetCurrentState()
	currentState := owp.stateManager.UpdateState(warResponse)

	// Log current state at start of processing loop
	stateInfo := owp.stateManager.GetStateInfo()
	slog.Info("Starting war processor loop",
		"current_state", stateInfo.State.String(),
		"state_description", stateInfo.Description,
		"time_in_state", stateInfo.TimeInState,
		"time_until_next_check", stateInfo.TimeUntilCheck)

	// Continuous monitoring enabled - always process all states
	slog.Debug("Continuous monitoring enabled - processing all states",
		"current_state", stateInfo.State.String())

	// Log pre-processing stats
	preStats := owp.tracker.GetSessionStats()
	slog.Debug("API calls before processing",
		"session_calls_before", preStats.SessionCalls)

	// Log state information
	stateInfo = owp.stateManager.GetStateInfo()
	slog.Info("War state analysis complete",
		"war_state", currentState.String(),
		"description", stateInfo.Description,
		"time_in_state", stateInfo.TimeInState,
		"next_check_in", stateInfo.TimeUntilCheck,
		"state_changed", previousState != currentState)

	// Ensure our faction ID is available for state tracking
	if err := owp.processor.ensureOurFactionID(ctx); err != nil {
		slog.Error("Failed to ensure our faction ID - continuing without state tracking", "err", err)
	}

	// Process state changes for all observed factions
	owp.processStateChanges(ctx, warResponse, stateInfo)

	// Handle different states
	switch currentState {
	case war.NoWars:
		slog.Info("No active wars - processing our faction status only",
			"next_matchmaking", owp.stateManager.GetNextCheckTime())

		// Process just our faction's status when no wars exist
		return owp.processOurFactionOnly(ctx)

	case war.PostWar:
		slog.Info("War completed - continuing processing for post-war analysis",
			"next_matchmaking", owp.stateManager.GetNextCheckTime())

	case war.PreWar:
		slog.Info("Pre-war reconnaissance mode - monitoring opponent",
			"update_interval", stateInfo.UpdateInterval)
		// Continue to processing for reconnaissance data

	case war.ActiveWar:
		slog.Info("Active war detected - real-time monitoring enabled",
			"update_interval", stateInfo.UpdateInterval)
		// Continue to full processing
	}

	// Process wars for PreWar and ActiveWar states (NoWars and PostWar are handled above)
	if currentState == war.PreWar || currentState == war.ActiveWar {
		// Process wars using existing logic but with optimized client
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
	slog.Info("Processing our faction status only (no active wars)")

	// Ensure our faction ID is loaded
	if err := owp.processor.ensureOurFactionID(ctx); err != nil {
		return fmt.Errorf("failed to initialize faction ID: %w", err)
	}

	ourFactionID := owp.processor.ourFactionID
	if ourFactionID == 0 {
		return fmt.Errorf("our faction ID is not set")
	}

	slog.Info("Successfully processed our faction status",
		"faction_id", ourFactionID)

	return nil
}

// processStateChanges handles state tracking for all observed factions
func (owp *OptimizedWarProcessor) processStateChanges(ctx context.Context, warResponse *app.WarResponse, stateInfo war.WarStateInfo) {
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
		slog.Debug("No factions to track for state changes")
		return
	}

	// Process state changes for all factions (ranked, raids, territory)
	slog.Debug("Processing state changes for factions",
		"faction_ids", factionIDs)

	if err := owp.stateTracker.ProcessStateChanges(ctx, owp.spreadsheetID, factionIDs); err != nil {
		slog.Error("Failed to process state changes - continuing with main processing",
			"err", err,
			"faction_ids", factionIDs)
	} else {
		slog.Debug("Successfully processed state changes",
			"faction_ids", factionIDs)
	}

	// Build faction list scoped to ranked war only for the tactical dashboard.
	// When a ranked war is active alongside raids/territory wars, we only want
	// the ranked war opponent in the Status v2 sheets and travel_data.json.
	var dashboardFactionIDs []int
	if owp.processor.ourFactionID != 0 {
		dashboardFactionIDs = append(dashboardFactionIDs, owp.processor.ourFactionID)
	}
	if warResponse.Wars.Ranked != nil {
		for _, faction := range warResponse.Wars.Ranked.Factions {
			dashboardFactionIDs = append(dashboardFactionIDs, faction.ID)
		}
	}
	// Fall back to all factions if there is no ranked war
	if len(dashboardFactionIDs) <= 1 {
		dashboardFactionIDs = factionIDs
	}
	dashboardFactionIDs = owp.removeDuplicateFactionIDs(dashboardFactionIDs)

	// Process Status v2 sheets for ranked war factions only (tactical dashboard)
	slog.Debug("Processing Status v2 for ranked war factions",
		"faction_ids", dashboardFactionIDs)

	if err := owp.statusV2Processor.ProcessStatusV2ForFactions(ctx, owp.spreadsheetID, dashboardFactionIDs, stateInfo.UpdateInterval); err != nil {
		slog.Error("Failed to process Status v2 - continuing with main processing",
			"err", err,
			"faction_ids", dashboardFactionIDs)
	} else {
		slog.Debug("Successfully processed Status v2",
			"faction_ids", dashboardFactionIDs)
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

// ProcessingSummary provides a summary of processing session including API call
// statistics, duration, and call rate metrics.
type ProcessingSummary struct {
	SessionAPICalls int64
	TotalAPICalls   int64
	CallsPerMinute  float64
	SessionDuration time.Duration
}
