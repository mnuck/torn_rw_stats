package services

import (
	"context"
	"fmt"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/attack"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/processing"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"

	"github.com/rs/zerolog/log"
)

// WarProcessor handles war detection and processing
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

// NewOptimizedProcessor creates an OptimizedWarProcessor with concrete implementations
// This is the recommended constructor for production use with state-based optimization
func NewOptimizedProcessor(tornClient *torn.Client, sheetsClient *sheets.Client, config *app.Config) *OptimizedWarProcessor {
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
	)
}

// ensureOurFactionID fetches and caches our faction ID if not already set
func (wp *WarProcessor) ensureOurFactionID(ctx context.Context) error {
	if wp.ourFactionID == 0 {
		log.Debug().Msg("Fetching our faction ID from API")

		factionInfo, err := wp.tornClient.GetOwnFaction(ctx)
		if err != nil {
			return fmt.Errorf("failed to get own faction info: %w", err)
		}

		wp.ourFactionID = factionInfo.ID
		log.Info().
			Int("faction_id", wp.ourFactionID).
			Str("faction_name", factionInfo.Name).
			Str("faction_tag", factionInfo.Tag).
			Msg("Detected our faction ID")
	}
	return nil
}

// ProcessActiveWars fetches current wars and processes each one
func (wp *WarProcessor) ProcessActiveWars(ctx context.Context) error {
	log.Info().Msg("Processing active wars")

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
		log.Info().
			Int("war_id", warResponse.Wars.Ranked.ID).
			Msg("Processing ranked war")

		if err := wp.processWar(ctx, warResponse.Wars.Ranked); err != nil {
			log.Error().
				Err(err).
				Int("war_id", warResponse.Wars.Ranked.ID).
				Msg("Failed to process ranked war")
		} else {
			processedWars++
		}
	}

	// Process raid wars
	for _, war := range warResponse.Wars.Raids {
		log.Info().
			Int("war_id", war.ID).
			Msg("Processing raid war")

		if err := wp.processWar(ctx, &war); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Msg("Failed to process raid war")
		} else {
			processedWars++
		}
	}

	// Process territory wars
	for _, war := range warResponse.Wars.Territory {
		log.Info().
			Int("war_id", war.ID).
			Msg("Processing territory war")

		if err := wp.processWar(ctx, &war); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Msg("Failed to process territory war")
		} else {
			processedWars++
		}
	}

	log.Info().
		Int("processed_wars", processedWars).
		Msg("Completed processing active wars")

	return nil
}

// processWar handles processing a single war
func (wp *WarProcessor) processWar(ctx context.Context, war *app.War) error {
	log.Info().
		Int("war_id", war.ID).
		Int("factions_count", len(war.Factions)).
		Int64("start_time", war.Start).
		Msg("=== ENTERING processWar ===")

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

	// Fetch attacks based on existing data
	var attacks []app.Attack
	if existingInfo.RecordCount == 0 {
		// Full population mode
		log.Debug().Int("war_id", war.ID).Msg("Using full population mode - no existing records")
		processor := torn.NewAttackProcessor(wp.tornClient)
		attacks, err = processor.GetAllAttacksForWar(ctx, war)
	} else {
		// Incremental update mode
		log.Debug().
			Int("war_id", war.ID).
			Int("existing_records", existingInfo.RecordCount).
			Int64("latest_timestamp", existingInfo.LatestTimestamp).
			Msg("Using incremental update mode - existing records found")
		processor := torn.NewAttackProcessor(wp.tornClient)
		attacks, err = processor.GetAttacksForTimeRange(ctx, war, war.Start, &existingInfo.LatestTimestamp)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch attacks for war: %w", err)
	}

	log.Debug().
		Int("war_id", war.ID).
		Int("attacks_count", len(attacks)).
		Msg("Fetched attacks for war")

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
		log.Error().
			Int("total_records", len(records)).
			Int("duplicate_codes", len(duplicateRecords)).
			Strs("duplicate_records", duplicateRecords).
			Msg("=== DUPLICATES DETECTED IN PROCESSED RECORDS ===")
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

	log.Info().
		Int("war_id", war.ID).
		Int("attacks_processed", len(attacks)).
		Int("records_created", len(records)).
		Msg("=== EXITING processWar - Successfully processed war ===")

	return nil
}

// getOurFactionID determines which faction is "ours" in the war
func (wp *WarProcessor) getOurFactionID(war *app.War) int {
	return wp.ourFactionID
}
