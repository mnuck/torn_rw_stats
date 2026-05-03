package services

import (
	"context"
	"fmt"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/attack"
	"torn_rw_stats/internal/domain/travel"
	wardomain "torn_rw_stats/internal/domain/war"
	"log/slog"

	"torn_rw_stats/internal/processing"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"
)

// WarProcessor handles war detection and processing, coordinating attack collection,
// travel tracking, and sheet management for faction wars.
type WarProcessor struct {
	tornClient        processing.TornClientInterface
	sheetsClient      processing.SheetsClientInterface
	config            *app.Config
	ourFactionID      int // Cached faction ID from API
	locationService   processing.LocationServiceInterface
	travelTimeService processing.TravelTimeServiceInterface
	attackService     processing.AttackProcessingServiceInterface
	summaryService    processing.WarSummaryServiceInterface
}

// NewWarProcessor creates a WarProcessor with interface dependencies for testability
func NewWarProcessor(
	tornClient processing.TornClientInterface,
	sheetsClient processing.SheetsClientInterface,
	locationService processing.LocationServiceInterface,
	travelTimeService processing.TravelTimeServiceInterface,
	attackService processing.AttackProcessingServiceInterface,
	summaryService processing.WarSummaryServiceInterface,
	config *app.Config,
) *WarProcessor {
	return &WarProcessor{
		tornClient:        tornClient,
		sheetsClient:      sheetsClient,
		config:            config,
		ourFactionID:      0, // Will be initialized on first use
		locationService:   locationService,
		travelTimeService: travelTimeService,
		attackService:     attackService,
		summaryService:    summaryService,
	}
}

// NewOptimizedProcessor creates an OptimizedWarProcessor with concrete implementations.
// This is the recommended constructor for production use with state-based optimization.
// bqClient may be nil to disable BigQuery integration.
func NewOptimizedProcessor(tornClient *torn.Client, sheetsClient *sheets.Client, config *app.Config, bqClient processing.BigQueryClientInterface) *OptimizedWarProcessor {
	// Create the attack processing service
	attackService := attack.NewAttackProcessingService()
	summaryService := NewWarSummaryService(attackService)

	return NewOptimizedWarProcessor(
		tornClient,
		sheetsClient,
		travel.NewLocationService(),
		travel.NewTravelTimeService(),
		attackService,
		summaryService,
		config,
		bqClient,
	)
}

// ensureOurFactionID fetches and caches our faction ID if not already set
func (wp *WarProcessor) ensureOurFactionID(ctx context.Context) error {
	if wp.ourFactionID == 0 {
		slog.Debug("Fetching our faction ID from API")

		factionInfo, err := wp.tornClient.GetOwnFaction(ctx)
		if err != nil {
			return fmt.Errorf("failed to get own faction info: %w", err)
		}

		wp.ourFactionID = factionInfo.ID
		slog.Info("Detected our faction ID",
			"faction_id", wp.ourFactionID,
			"faction_name", factionInfo.Name,
			"faction_tag", factionInfo.Tag)
	}
	return nil
}

// ProcessActiveWars fetches current wars and processes each one
func (wp *WarProcessor) ProcessActiveWars(ctx context.Context) error {
	slog.Info("Processing active wars")

	// Ensure our faction ID is loaded
	if err := wp.ensureOurFactionID(ctx); err != nil {
		return fmt.Errorf("failed to initialize faction ID: %w", err)
	}

	warResponse, err := wp.tornClient.GetFactionWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch faction wars: %w", err)
	}

	var processedWars int

	// Process ranked war if it exists
	if warResponse.Wars.Ranked != nil {
		slog.Info("Processing ranked war",
			"war_id", warResponse.Wars.Ranked.ID)

		if err := wp.processWar(ctx, warResponse.Wars.Ranked); err != nil {
			slog.Error("Failed to process ranked war",
				"err", err,
				"war_id", warResponse.Wars.Ranked.ID)
		} else {
			processedWars++
		}
	}

	// Process raid wars
	for _, war := range warResponse.Wars.Raids {
		slog.Info("Processing raid war",
			"war_id", war.ID)

		if err := wp.processWar(ctx, &war); err != nil {
			slog.Error("Failed to process raid war",
				"err", err,
				"war_id", war.ID)
		} else {
			processedWars++
		}
	}

	// Process territory wars
	for _, war := range warResponse.Wars.Territory {
		slog.Info("Processing territory war",
			"war_id", war.ID)

		if err := wp.processWar(ctx, &war); err != nil {
			slog.Error("Failed to process territory war",
				"err", err,
				"war_id", war.ID)
		} else {
			processedWars++
		}
	}

	slog.Info("Completed processing active wars",
		"processed_wars", processedWars)

	return nil
}

// processWar handles processing a single war
func (wp *WarProcessor) processWar(ctx context.Context, war *app.War) error {
	slog.Info("=== ENTERING processWar ===",
		"war_id", war.ID,
		"factions_count", len(war.Factions),
		"start_time", war.Start)

	// Ensure sheets exist for this war
	sheetConfig, err := wp.sheetsClient.EnsureWarSheets(ctx, wp.config.SpreadsheetID, war)
	if err != nil {
		return fmt.Errorf("failed to ensure war sheets: %w", err)
	}

	// Check if we have existing records to determine update mode
	existingInfo, err := wp.sheetsClient.ReadExistingRecords(ctx, wp.config.SpreadsheetID, sheetConfig.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Use domain function to determine fetch mode
	fetchDecision := wardomain.DetermineAttackFetchMode(existingInfo.RecordCount, existingInfo.LatestTimestamp)
	slog.Debug("Determined attack fetch mode",
		"war_id", war.ID,
		"use_full_mode", fetchDecision.UseFullMode,
		"use_incremental", fetchDecision.UseIncremental,
		"reason", fetchDecision.Reason)

	// Fetch attacks based on decision
	var attacks []app.Attack
	processor := torn.NewAttackProcessor(wp.tornClient)
	if fetchDecision.UseFullMode {
		attacks, err = processor.GetAllAttacksForWar(ctx, war)
	} else {
		attacks, err = processor.GetAttacksForTimeRange(ctx, war, war.Start, &fetchDecision.LatestTimestamp)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch attacks for war: %w", err)
	}

	slog.Debug("Fetched attacks for war",
		"war_id", war.ID,
		"attacks_count", len(attacks))

	// Get our faction ID for processing
	ourFactionID := wp.getOurFactionID(war)

	// Process attack data into records
	records := wp.attackService.ProcessAttacksIntoRecords(attacks, war, ourFactionID)

	// Check for duplicates in processed records
	codeCount := make(map[string]int)
	var duplicateRecords []string
	for _, record := range records {
		codeCount[record.Code]++
		if codeCount[record.Code] == 2 {
			duplicateRecords = append(duplicateRecords, fmt.Sprintf("ID:%d Code:%s", record.AttackID, record.Code))
		}
	}

	if len(duplicateRecords) > 0 {
		slog.Error("=== DUPLICATES DETECTED IN PROCESSED RECORDS ===",
			"total_records", len(records),
			"duplicate_codes", len(duplicateRecords),
			"duplicate_records", duplicateRecords)
	}

	// Generate war summary
	summary := wp.summaryService.GenerateWarSummary(war, attacks, ourFactionID)

	// Update sheets
	if err := wp.sheetsClient.UpdateWarSummary(ctx, wp.config.SpreadsheetID, sheetConfig, summary); err != nil {
		return fmt.Errorf("failed to update war summary: %w", err)
	}

	if err := wp.sheetsClient.UpdateAttackRecords(ctx, wp.config.SpreadsheetID, sheetConfig, records); err != nil {
		return fmt.Errorf("failed to update attack records: %w", err)
	}

	slog.Info("=== EXITING processWar - Successfully processed war ===",
		"war_id", war.ID,
		"attacks_processed", len(attacks),
		"records_created", len(records))

	return nil
}

// getOurFactionID determines which faction is "ours" in the war
func (wp *WarProcessor) getOurFactionID(war *app.War) int {
	return wp.ourFactionID
}
