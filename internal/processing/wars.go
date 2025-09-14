package processing

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"

	"github.com/rs/zerolog/log"
)

// WarProcessor handles war detection and processing
type WarProcessor struct {
	tornClient         TornClientInterface
	sheetsClient       SheetsClientInterface
	config             *app.Config
	ourFactionID       int // Cached faction ID from API
	locationService    LocationServiceInterface
	travelTimeService  TravelTimeServiceInterface
	attackService      AttackProcessingServiceInterface
	summaryService     WarSummaryServiceInterface
	stateChangeService StateChangeDetectionServiceInterface
}

// NewWarProcessor creates a WarProcessor with interface dependencies for testability
func NewWarProcessor(
	tornClient TornClientInterface,
	sheetsClient SheetsClientInterface,
	locationService LocationServiceInterface,
	travelTimeService TravelTimeServiceInterface,
	attackService AttackProcessingServiceInterface,
	summaryService WarSummaryServiceInterface,
	stateChangeService StateChangeDetectionServiceInterface,
	config *app.Config,
) *WarProcessor {
	return &WarProcessor{
		tornClient:         tornClient,
		sheetsClient:       sheetsClient,
		config:             config,
		ourFactionID:       0, // Will be initialized on first use
		locationService:    locationService,
		travelTimeService:  travelTimeService,
		attackService:      attackService,
		summaryService:     summaryService,
		stateChangeService: stateChangeService,
	}
}

// NewOptimizedWarProcessorWithConcreteDependencies creates an OptimizedWarProcessor with concrete implementations
// This is the recommended constructor for production use with state-based optimization
func NewOptimizedWarProcessorWithConcreteDependencies(tornClient *torn.Client, sheetsClient *sheets.Client, config *app.Config) *OptimizedWarProcessor {
	// Create the attack processing service
	attackService := NewAttackProcessingService(config.OurFactionID)
	summaryService := NewWarSummaryService(attackService)
	stateChangeService := NewStateChangeDetectionService(sheetsClient)

	return NewOptimizedWarProcessor(
		tornClient,
		sheetsClient,
		NewLocationService(),
		NewTravelTimeService(),
		attackService,
		summaryService,
		stateChangeService,
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
		attacks, err = wp.tornClient.GetAllAttacksForWar(ctx, war)
	} else {
		// Incremental update mode
		log.Debug().
			Int("war_id", war.ID).
			Int("existing_records", existingInfo.RecordCount).
			Int64("latest_timestamp", existingInfo.LatestTimestamp).
			Msg("Using incremental update mode - existing records found")
		attacks, err = wp.tornClient.GetAttacksForTimeRange(ctx, war, war.Start, &existingInfo.LatestTimestamp)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch attacks for war: %w", err)
	}

	log.Debug().
		Int("war_id", war.ID).
		Int("attacks_count", len(attacks)).
		Msg("Fetched attacks for war")

	// Process attack data into records
	records := wp.attackService.ProcessAttacksIntoRecords(attacks, war)

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
	summary := wp.summaryService.GenerateWarSummary(war, attacks)

	// Update sheets
	if err := wp.sheetsClient.UpdateWarSummary(ctx, wp.config.SpreadsheetID, sheetConfig, summary); err != nil {
		return fmt.Errorf("failed to update war summary: %w", err)
	}

	if err := wp.sheetsClient.UpdateAttackRecords(ctx, wp.config.SpreadsheetID, sheetConfig, records); err != nil {
		return fmt.Errorf("failed to update attack records: %w", err)
	}

	// Process travel status for both factions - fetch data upfront to avoid lag
	ourFactionID := wp.getOurFactionID(war)
	enemyFactionID := wp.getEnemyFactionID(war)

	// Fetch faction data upfront for both factions to ensure synchronized snapshots
	var ourFactionData, enemyFactionData *app.FactionBasicResponse

	if ourFactionID > 0 {
		var err error
		ourFactionData, err = wp.fetchFactionData(ctx, ourFactionID)
		if err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Int("our_faction_id", ourFactionID).
				Msg("Failed to fetch our faction data - skipping travel status")
		}
	}

	if enemyFactionID > 0 {
		var err error
		enemyFactionData, err = wp.fetchFactionData(ctx, enemyFactionID)
		if err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Int("enemy_faction_id", enemyFactionID).
				Msg("Failed to fetch enemy faction data - skipping travel status")
		}
	}

	// Process our faction travel status with pre-fetched data
	if ourFactionID > 0 && ourFactionData != nil {
		if err := wp.processTravelStatusWithData(ctx, war, ourFactionID, wp.config.SpreadsheetID, ourFactionData); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Int("our_faction_id", ourFactionID).
				Msg("Failed to process our faction travel status - continuing with war processing")
		}
	}

	// Process enemy faction travel status with pre-fetched data
	if enemyFactionID > 0 && enemyFactionData != nil {
		if err := wp.processTravelStatusWithData(ctx, war, enemyFactionID, wp.config.SpreadsheetID, enemyFactionData); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Int("enemy_faction_id", enemyFactionID).
				Msg("Failed to process enemy faction travel status - continuing with war processing")
		}
	}

	log.Info().
		Int("war_id", war.ID).
		Int("attacks_processed", len(attacks)).
		Int("records_created", len(records)).
		Msg("=== EXITING processWar - Successfully processed war ===")

	return nil
}

// processAttacksIntoRecords converts API attack data into sheet records
func (wp *WarProcessor) processAttacksIntoRecords(attacks []app.Attack, war *app.War) []app.AttackRecord {
	var records []app.AttackRecord

	// Determine our faction ID (we'll need to configure this)
	ourFactionID := wp.getOurFactionID(war)

	for _, attack := range attacks {
		record := app.AttackRecord{
			AttackID:            attack.ID,
			Code:                attack.Code,
			Started:             time.Unix(attack.Started, 0),
			Ended:               time.Unix(attack.Ended, 0),
			AttackerID:          attack.Attacker.ID,
			AttackerName:        attack.Attacker.Name,
			AttackerLevel:       attack.Attacker.Level,
			DefenderID:          attack.Defender.ID,
			DefenderName:        attack.Defender.Name,
			DefenderLevel:       attack.Defender.Level,
			Result:              attack.Result,
			RespectGain:         attack.RespectGain,
			RespectLoss:         attack.RespectLoss,
			Chain:               attack.Chain,
			IsInterrupted:       attack.IsInterrupted,
			IsStealthed:         attack.IsStealthed,
			IsRaid:              attack.IsRaid,
			IsRankedWar:         attack.IsRankedWar,
			ModifierFairFight:   attack.Modifiers.FairFight,
			ModifierWar:         attack.Modifiers.War,
			ModifierRetaliation: attack.Modifiers.Retaliation,
			ModifierGroup:       attack.Modifiers.Group,
			ModifierOverseas:    attack.Modifiers.Overseas,
			ModifierChain:       attack.Modifiers.Chain,
			ModifierWarlord:     attack.Modifiers.Warlord,
		}

		// Handle attacker faction
		if attack.Attacker.Faction != nil {
			record.AttackerFactionID = &attack.Attacker.Faction.ID
			record.AttackerFactionName = attack.Attacker.Faction.Name
		}

		// Handle defender faction
		if attack.Defender.Faction != nil {
			record.DefenderFactionID = &attack.Defender.Faction.ID
			record.DefenderFactionName = attack.Defender.Faction.Name
		}

		// Handle finishing hit effects (take the first one if it exists)
		if len(attack.FinishingHitEffects) > 0 {
			record.FinishingHitName = attack.FinishingHitEffects[0].Name
			record.FinishingHitValue = attack.FinishingHitEffects[0].Value
		}

		// Determine attack direction
		if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
			record.Direction = "Outgoing"
		} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
			record.Direction = "Incoming"
		} else {
			record.Direction = "Unknown"
		}

		records = append(records, record)
	}

	log.Debug().
		Int("total_attacks", len(attacks)).
		Int("records_created", len(records)).
		Int("our_faction_id", ourFactionID).
		Msg("Processed attacks into records")

	return records
}

// generateWarSummary creates a summary of the war statistics
func (wp *WarProcessor) generateWarSummary(war *app.War, attacks []app.Attack) *app.WarSummary {
	ourFactionID := wp.getOurFactionID(war)

	summary := &app.WarSummary{
		WarID:       war.ID,
		StartTime:   time.Unix(war.Start, 0),
		Status:      "Active",
		LastUpdated: time.Now(),
	}

	if war.End != nil {
		endTime := time.Unix(*war.End, 0)
		summary.EndTime = &endTime
		summary.Status = "Completed"
	}

	// Find our faction and enemy faction
	for _, faction := range war.Factions {
		if faction.ID == ourFactionID {
			summary.OurFaction = faction
		} else {
			summary.EnemyFaction = faction
		}
	}

	// Calculate attack statistics
	var attacksWon, attacksLost int
	var respectGained, respectLost float64

	for _, attack := range attacks {
		if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
			// Our attack
			summary.TotalAttacks++
			respectGained += attack.RespectGain
			respectLost += attack.RespectLoss

			// Determine if we won (simplified - you may need more sophisticated logic)
			if attack.Result == "Hospitalized" || attack.Result == "Mugged" || attack.Result == "Left" {
				attacksWon++
			} else {
				attacksLost++
			}
		} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
			// Attack against us
			summary.TotalAttacks++

			// For defensive stats, respect gain/loss is inverted
			respectLost += attack.RespectGain
			respectGained += attack.RespectLoss

			// We "won" if we defended successfully
			if attack.Result == "Stalemate" || attack.Result == "Escape" || attack.Result == "Assist" {
				attacksWon++
			} else {
				attacksLost++
			}
		}
	}

	summary.AttacksWon = attacksWon
	summary.AttacksLost = attacksLost
	summary.RespectGained = respectGained
	summary.RespectLost = respectLost

	// Set war name based on factions
	summary.WarName = fmt.Sprintf("%s vs %s", summary.OurFaction.Name, summary.EnemyFaction.Name)

	log.Debug().
		Int("war_id", war.ID).
		Int("total_attacks", summary.TotalAttacks).
		Int("attacks_won", summary.AttacksWon).
		Int("attacks_lost", summary.AttacksLost).
		Float64("respect_gained", summary.RespectGained).
		Float64("respect_lost", summary.RespectLost).
		Msg("Generated war summary")

	return summary
}

// getOurFactionID determines which faction is "ours" in the war
func (wp *WarProcessor) getOurFactionID(war *app.War) int {
	return wp.ourFactionID
}

// getEnemyFactionID determines which faction is the enemy in the war
func (wp *WarProcessor) getEnemyFactionID(war *app.War) int {
	ourFactionID := wp.getOurFactionID(war)

	// Return the faction ID that is not ours
	for _, faction := range war.Factions {
		if faction.ID != ourFactionID {
			return faction.ID
		}
	}

	return 0
}

// getFactionName returns the faction name for a given ID from war data
func (wp *WarProcessor) getFactionName(war *app.War, factionID int) string {
	for _, faction := range war.Factions {
		if faction.ID == factionID {
			return faction.Name
		}
	}
	return fmt.Sprintf("Faction %d", factionID) // fallback
}

// parseHospitalCountdown parses hospital countdown from status description
func (wp *WarProcessor) parseHospitalCountdown(description string) string {
	if description == "" {
		return ""
	}

	// Log the input for debugging
	log.Debug().
		Str("description", description).
		Msg("Parsing hospital countdown")

	var hours, minutes int

	// Look for hours pattern: "2h", "12h", "1hrs", "2 hrs", etc.
	hoursRe := regexp.MustCompile(`(\d+)\s*hrs?`)
	if hoursMatch := hoursRe.FindStringSubmatch(description); len(hoursMatch) > 1 {
		if h, err := strconv.Atoi(hoursMatch[1]); err == nil {
			hours = h
		}
	}

	// Look for minutes pattern: "15m", "45m", "35mins", etc.
	minutesRe := regexp.MustCompile(`(\d+)\s*mins?`)
	if minutesMatch := minutesRe.FindStringSubmatch(description); len(minutesMatch) > 1 {
		if m, err := strconv.Atoi(minutesMatch[1]); err == nil {
			minutes = m
		}
	}

	// Only return a countdown if we found at least hours or minutes
	if hours > 0 || minutes > 0 {
		result := fmt.Sprintf("%02d:%02d:00", hours, minutes)
		log.Debug().
			Str("description", description).
			Int("hours", hours).
			Int("minutes", minutes).
			Str("result", result).
			Msg("Parsed hospital countdown")
		return result
	}

	log.Debug().
		Str("description", description).
		Msg("No hospital countdown found")
	return ""
}

// calculateTravelTimes calculates travel departure, arrival and countdown for a user
func (wp *WarProcessor) calculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time) *TravelTimeData {
	return wp.travelTimeService.CalculateTravelTimes(ctx, userID, destination, travelType, currentTime, wp.config.UpdateInterval)
}

// calculateTravelTimesFromDeparture calculates arrival and countdown based on existing departure time
func (wp *WarProcessor) calculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr, existingArrivalStr string, travelType string, currentTime time.Time) *TravelTimeData {
	return wp.travelTimeService.CalculateTravelTimesFromDeparture(ctx, userID, destination, departureStr, existingArrivalStr, travelType, currentTime, wp.locationService)
}

// readExistingTravelData reads existing travel records from sheet to preserve departure times
func (wp *WarProcessor) readExistingTravelData(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.TravelRecord, error) {
	// Read all data from the sheet (starting from row 2 to skip headers)
	range_ := fmt.Sprintf("'%s'!A2:G", sheetName)
	values, err := wp.sheetsClient.ReadSheet(ctx, spreadsheetID, range_)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing travel data: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Str("range", range_).
		Int("rows_returned", len(values)).
		Msg("Raw sheet data read from API")

	existingData := make(map[string]app.TravelRecord)

	for i, row := range values {
		log.Debug().
			Str("sheet_name", sheetName).
			Int("row_number", i+2). // +2 because we start from A2
			Int("columns_count", len(row)).
			Interface("row_data", row).
			Msg("Processing sheet row")

		if len(row) < 4 { // Need at least 4 columns: Name, Level, Location, State
			log.Debug().
				Str("sheet_name", sheetName).
				Int("row_number", i+2).
				Int("columns_count", len(row)).
				Msg("Skipping row - insufficient columns")
			continue
		}

		// Parse the row data
		name, ok := row[0].(string)
		if !ok || name == "" {
			continue
		}

		record := app.TravelRecord{
			Name: name,
		}

		// Parse level
		if levelVal, ok := row[1].(float64); ok {
			record.Level = int(levelVal)
		}

		// Parse other string fields (column layout: Name, Level, Status, Location, Countdown, Departure, Arrival)
		if state, ok := row[2].(string); ok {
			record.State = state
		}
		if location, ok := row[3].(string); ok {
			record.Location = location
		}
		// Parse optional columns safely
		if len(row) > 4 {
			if countdown, ok := row[4].(string); ok {
				record.Countdown = countdown
			}
		}
		if len(row) > 5 {
			if departure, ok := row[5].(string); ok {
				record.Departure = departure
			}
		}
		if len(row) > 6 {
			if arrival, ok := row[6].(string); ok {
				record.Arrival = arrival
			}
		}

		existingData[name] = record
	}

	log.Debug().
		Int("existing_records", len(existingData)).
		Str("sheet_name", sheetName).
		Msg("Read existing travel data")

	// Debug log existing travel data for troubleshooting
	for name, record := range existingData {
		log.Debug().
			Str("player", name).
			Str("departure", record.Departure).
			Str("arrival", record.Arrival).
			Str("countdown", record.Countdown).
			Str("state", record.State).
			Msg("Found existing travel data record")
	}

	return existingData, nil
}

// getFactionTypeForLogging determines faction type string for logging
func (wp *WarProcessor) getFactionTypeForLogging(factionID int) string {
	if factionID == wp.ourFactionID {
		return "our faction"
	}
	return "enemy faction"
}

// setupTravelTracking sets up travel tracking infrastructure (sheets and existing data)
func (wp *WarProcessor) setupTravelTracking(ctx context.Context, spreadsheetID string, factionID int) (string, map[string]app.TravelRecord, error) {
	// Ensure travel status sheet exists
	travelSheetName, err := wp.sheetsClient.EnsureTravelStatusSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to ensure travel status sheet: %w", err)
	}

	// Read existing travel data to preserve departure times
	existingTravelData, err := wp.readExistingTravelData(ctx, spreadsheetID, travelSheetName)
	if err != nil {
		log.Warn().
			Err(err).
			Str("sheet_name", travelSheetName).
			Msg("Failed to read existing travel data, will create fresh records")
		existingTravelData = make(map[string]app.TravelRecord) // Empty map as fallback
	}

	return travelSheetName, existingTravelData, nil
}

// fetchFactionData retrieves current faction data from API
func (wp *WarProcessor) fetchFactionData(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	factionData, err := wp.tornClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch faction basic data: %w", err)
	}
	return factionData, nil
}

// handleStateChangeDetection manages state change detection and persistence
func (wp *WarProcessor) handleStateChangeDetection(ctx context.Context, war *app.War, factionID int, factionData *app.FactionBasicResponse, spreadsheetID string) (map[string]app.FactionMember, error) {
	factionName := wp.getFactionName(war, factionID)

	// Load previous member states from persistent storage
	var previousMembers map[string]app.FactionMember
	previousStateSheetName, err := wp.sheetsClient.EnsurePreviousStateSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		log.Error().
			Err(err).
			Int("faction_id", factionID).
			Msg("Failed to ensure previous state sheet - skipping state change detection")
		previousMembers = make(map[string]app.FactionMember) // Empty map as fallback
	} else {
		// Load previous states
		loadedPreviousMembers, err := wp.sheetsClient.LoadPreviousMemberStates(ctx, spreadsheetID, previousStateSheetName)
		if err != nil {
			log.Warn().
				Err(err).
				Int("faction_id", factionID).
				Str("sheet_name", previousStateSheetName).
				Msg("Failed to load previous member states - skipping state change detection")
			previousMembers = make(map[string]app.FactionMember) // Empty map as fallback
		} else {
			previousMembers = loadedPreviousMembers
			if len(previousMembers) > 0 {
				log.Debug().
					Int("faction_id", factionID).
					Int("previous_members", len(previousMembers)).
					Int("current_members", len(factionData.Members)).
					Msg("Processing state changes")

				if err := wp.stateChangeService.ProcessStateChanges(ctx, factionID, factionName, previousMembers, factionData.Members, spreadsheetID); err != nil {
					log.Error().
						Err(err).
						Int("faction_id", factionID).
						Msg("Failed to process state changes - continuing")
				}
			} else {
				log.Debug().
					Int("faction_id", factionID).
					Msg("No previous member states found - first run for this faction")
			}
		}

		// Store current member states for next comparison
		if err := wp.sheetsClient.StorePreviousMemberStates(ctx, spreadsheetID, previousStateSheetName, factionData.Members); err != nil {
			log.Error().
				Err(err).
				Int("faction_id", factionID).
				Str("sheet_name", previousStateSheetName).
				Msg("Failed to store current member states")
		} else {
			log.Debug().
				Int("faction_id", factionID).
				Int("members_stored", len(factionData.Members)).
				Str("sheet_name", previousStateSheetName).
				Msg("Stored current member states for next comparison")
		}
	}

	return previousMembers, nil
}

// convertMembersToTravelRecords processes faction members into travel records
func (wp *WarProcessor) convertMembersToTravelRecords(ctx context.Context, members map[string]app.FactionMember, existingTravelData map[string]app.TravelRecord, previousMembers map[string]app.FactionMember) []app.TravelRecord {
	currentTime := time.Now().UTC()
	var travelRecords []app.TravelRecord

	for userIDStr, member := range members {
		// Parse user ID for travel time calculations
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Warn().
				Str("user_id_str", userIDStr).
				Str("member_name", member.Name).
				Msg("Failed to parse user ID, skipping travel tracking")
			userID = 0
		}

		// Parse standardized location
		location := wp.locationService.ParseLocation(member.Status.Description)

		// Determine corrected state for special cases
		state := member.Status.State
		if strings.Contains(strings.ToLower(member.Status.Description), "returning to torn from") {
			state = "Traveling"
		}
		if strings.EqualFold(member.Status.State, "Abroad") {
			state = "Okay"
		}

		record := app.TravelRecord{
			Name:     member.Name,
			Level:    member.Level,
			Location: location,
			State:    state,
		}

		// Process travel record based on member state
		wp.processTravelRecordByState(ctx, &record, member, userID, userIDStr, existingTravelData, previousMembers, currentTime)

		travelRecords = append(travelRecords, record)
	}

	// Sort records by level (descending), then alphabetically by name
	sort.Slice(travelRecords, func(i, j int) bool {
		if travelRecords[i].Level == travelRecords[j].Level {
			return travelRecords[i].Name < travelRecords[j].Name
		}
		return travelRecords[i].Level > travelRecords[j].Level
	})

	return travelRecords
}

// processTravelRecordByState handles travel record processing based on member state
func (wp *WarProcessor) processTravelRecordByState(ctx context.Context, record *app.TravelRecord, member app.FactionMember, userID int, userIDStr string, existingTravelData map[string]app.TravelRecord, previousMembers map[string]app.FactionMember, currentTime time.Time) {
	// Check existing travel record for departure/arrival time preservation
	existingRecord, hasExistingTravel := existingTravelData[member.Name]

	// Debug log for travel record lookup
	if strings.EqualFold(record.State, "Traveling") {
		log.Debug().
			Str("player", member.Name).
			Bool("has_existing_travel", hasExistingTravel).
			Str("existing_departure", func() string {
				if hasExistingTravel {
					return existingRecord.Departure
				}
				return "N/A"
			}()).
			Str("existing_countdown", func() string {
				if hasExistingTravel {
					return existingRecord.Countdown
				}
				return "N/A"
			}()).
			Int("total_existing_records", len(existingTravelData)).
			Msg("Travel record lookup debug")
	}

	// Get previous state from member state data (not travel data)
	previousState := ""
	if previousMember, hasPreviousMember := previousMembers[userIDStr]; hasPreviousMember {
		previousState = previousMember.Status.State
		// Handle same state corrections that we apply to current state
		if strings.Contains(strings.ToLower(previousMember.Status.Description), "returning to torn from") {
			previousState = "Traveling"
		}
		if strings.EqualFold(previousMember.Status.State, "Abroad") {
			previousState = "Okay"
		}
	}

	// Handle different states with proper state transition logic
	if strings.EqualFold(member.Status.State, "Hospital") {
		// Hospital countdown
		record.Countdown = wp.parseHospitalCountdown(member.Status.Description)

		// Clear travel times if transitioning from Traveling to Hospital
		if strings.EqualFold(previousState, "Traveling") {
			log.Debug().
				Str("player", member.Name).
				Str("previous_state", previousState).
				Str("new_state", record.State).
				Msg("State transition from Traveling - clearing departure and countdown")
		}
		// Departure and Countdown remain empty for Hospital state

	} else if strings.EqualFold(record.State, "Traveling") {
		if userID > 0 {
			if strings.EqualFold(previousState, "Traveling") && hasExistingTravel && existingRecord.Departure != "" {
				// Still traveling - preserve departure/arrival, only update countdown
				record.Departure = existingRecord.Departure
				// Recalculate only the countdown using existing departure time
				if travelData := wp.calculateTravelTimesFromDeparture(ctx, userID, record.Location, existingRecord.Departure, existingRecord.Arrival, member.Status.TravelType, currentTime); travelData != nil {
					record.Countdown = travelData.Countdown
					record.Arrival = travelData.Arrival
				}

				log.Debug().
					Str("player", member.Name).
					Str("departure", record.Departure).
					Str("countdown", record.Countdown).
					Msg("Still traveling - preserved departure, updated countdown only")

			} else if previousState != "" && !strings.EqualFold(previousState, "Traveling") {
				// State transition TO Traveling - set new departure and countdown times
				travelDestination := wp.locationService.GetTravelDestinationForCalculation(member.Status.Description, record.Location)
				if travelData := wp.calculateTravelTimes(ctx, userID, travelDestination, member.Status.TravelType, currentTime); travelData != nil {
					record.Departure = travelData.Departure
					record.Countdown = travelData.Countdown
					record.Arrival = travelData.Arrival

					log.Info().
						Str("player", member.Name).
						Str("previous_state", previousState).
						Str("new_state", record.State).
						Str("destination", travelDestination).
						Str("departure", record.Departure).
						Msg("State transition to Traveling - set new departure and countdown times")
				}
			} else {
				// No previous state data (fresh start) or already traveling with no departure time
				// Check if existing travel data has departure time - preserve it if present
				if hasExistingTravel && existingRecord.Departure != "" {
					// Preserve manually curated departure time
					record.Departure = existingRecord.Departure
					// Recalculate countdown from departure
					if travelData := wp.calculateTravelTimesFromDeparture(ctx, userID, record.Location, existingRecord.Departure, existingRecord.Arrival, member.Status.TravelType, currentTime); travelData != nil {
						record.Countdown = travelData.Countdown
						record.Arrival = travelData.Arrival
					}

					log.Debug().
						Str("player", member.Name).
						Str("previous_state", previousState).
						Str("new_state", record.State).
						Str("departure", record.Departure).
						Str("countdown", record.Countdown).
						Bool("has_previous_member_state", previousState != "").
						Bool("has_existing_travel", hasExistingTravel).
						Msg("Player traveling but no observed state transition - preserved existing departure time")
				} else {
					// Leave departure blank since we didn't observe the transition and have no existing data
					log.Debug().
						Str("player", member.Name).
						Str("previous_state", previousState).
						Str("new_state", record.State).
						Bool("has_previous_member_state", previousState != "").
						Bool("has_existing_travel", hasExistingTravel).
						Msg("Player traveling but no observed state transition - leaving departure blank")
				}
			}
		}
	} else {
		// Not traveling and not in hospital - clear travel times if transitioning from Traveling
		if strings.EqualFold(previousState, "Traveling") {
			log.Debug().
				Str("player", member.Name).
				Str("previous_state", previousState).
				Str("new_state", record.State).
				Msg("State transition from Traveling - clearing departure and countdown")
		}
		// Departure and Countdown remain empty for non-traveling states
	}
}

// processTravelStatus fetches and processes travel status for a faction
func (wp *WarProcessor) processTravelStatus(ctx context.Context, war *app.War, factionID int, spreadsheetID string) error {
	// Fetch faction basic data
	factionData, err := wp.fetchFactionData(ctx, factionID)
	if err != nil {
		return err
	}

	return wp.processTravelStatusWithData(ctx, war, factionID, spreadsheetID, factionData)
}

// processTravelStatusWithData processes travel status for a faction using pre-fetched faction data
// This eliminates data lag by using synchronized faction snapshots
func (wp *WarProcessor) processTravelStatusWithData(ctx context.Context, war *app.War, factionID int, spreadsheetID string, factionData *app.FactionBasicResponse) error {
	factionType := wp.getFactionTypeForLogging(factionID)

	log.Debug().
		Int("faction_id", factionID).
		Str("faction_type", factionType).
		Msg("Processing travel status with pre-fetched data")

	// Setup travel tracking infrastructure
	travelSheetName, existingTravelData, err := wp.setupTravelTracking(ctx, spreadsheetID, factionID)
	if err != nil {
		return fmt.Errorf("failed to setup travel tracking: %w", err)
	}

	// Handle state change detection and persistence using provided faction data
	previousMembers, err := wp.handleStateChangeDetection(ctx, war, factionID, factionData, spreadsheetID)
	if err != nil {
		// Log error but continue processing
		log.Error().
			Err(err).
			Int("faction_id", factionID).
			Msg("Failed to handle state change detection - continuing")
		previousMembers = make(map[string]app.FactionMember) // Fallback
	}

	// Convert faction members to travel records with travel time tracking
	travelRecords := wp.convertMembersToTravelRecords(ctx, factionData.Members, existingTravelData, previousMembers)

	// Update travel status sheet
	if err := wp.sheetsClient.UpdateTravelStatus(ctx, spreadsheetID, travelSheetName, travelRecords); err != nil {
		return fmt.Errorf("failed to update travel status sheet: %w", err)
	}

	log.Info().
		Int("faction_id", factionID).
		Str("faction_type", factionType).
		Int("members_processed", len(travelRecords)).
		Str("sheet_name", travelSheetName).
		Msg("Successfully processed travel status")

	return nil
}
