package processing

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"

	"github.com/rs/zerolog/log"
)

// WarProcessor handles war detection and processing
type WarProcessor struct {
	tornClient   *torn.Client
	sheetsClient *sheets.Client
	config       *app.Config
	ourFactionID int // Cached faction ID from API
}

func NewWarProcessor(tornClient *torn.Client, sheetsClient *sheets.Client, config *app.Config) *WarProcessor {
	return &WarProcessor{
		tornClient:   tornClient,
		sheetsClient: sheetsClient,
		config:       config,
		ourFactionID: 0, // Will be initialized on first use
	}
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
	records := wp.processAttacksIntoRecords(attacks, war)
	
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
	summary := wp.generateWarSummary(war, attacks)

	// Update sheets
	if err := wp.sheetsClient.UpdateWarSummary(ctx, wp.config.SpreadsheetID, sheetConfig, summary); err != nil {
		return fmt.Errorf("failed to update war summary: %w", err)
	}

	if err := wp.sheetsClient.UpdateAttackRecords(ctx, wp.config.SpreadsheetID, sheetConfig, records); err != nil {
		return fmt.Errorf("failed to update attack records: %w", err)
	}

	// Process travel status for both factions
	ourFactionID := wp.getOurFactionID(war)
	enemyFactionID := wp.getEnemyFactionID(war)
	
	// Process our faction travel status
	if ourFactionID > 0 {
		if err := wp.processTravelStatus(ctx, war, ourFactionID, wp.config.SpreadsheetID); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Int("our_faction_id", ourFactionID).
				Msg("Failed to process our faction travel status - continuing with war processing")
		}
	}
	
	// Process enemy faction travel status
	if enemyFactionID > 0 {
		if err := wp.processTravelStatus(ctx, war, enemyFactionID, wp.config.SpreadsheetID); err != nil {
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

// parseLocation extracts standardized location from status description
func (wp *WarProcessor) parseLocation(description string) string {
	if description == "" {
		return ""
	}

	// Hospital location mappings (adjective -> location name)
	hospitalMappings := map[string]string{
		"british":       "United Kingdom",
		"caymanian":     "Cayman Islands",
		"chinese":       "China",
		"mexican":       "Mexico",
		"swiss":         "Switzerland",
		"japanese":      "Japan",
		"canadian":      "Canada",
		"hawaiian":      "Hawaii",
		"emirati":       "UAE",
		"south african": "South Africa",
		"argentinian":   "Argentina",
	}

	descLower := strings.ToLower(description)

	// Check for hospital patterns first (handle both "a" and "an")
	for adjective, location := range hospitalMappings {
		if strings.Contains(descLower, "in a "+adjective+" hospital") || 
		   strings.Contains(descLower, "in an "+adjective+" hospital") {
			return location
		}
	}

	// Direct location patterns
	locations := []string{
		"Mexico", "Cayman Islands", "Canada", "Hawaii", "United Kingdom",
		"Argentina", "Switzerland", "Japan", "China", "UAE",
		"South Africa",
	}

	// Check "Traveling to X" pattern
	if strings.HasPrefix(descLower, "traveling to ") {
		for _, location := range locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				return location
			}
		}
	}

	// Check "In X" pattern
	if strings.HasPrefix(descLower, "in ") && !strings.Contains(descLower, "hospital") {
		for _, location := range locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				return location
			}
		}
	}

	// Check "Returning to Torn from X" pattern
	if strings.Contains(descLower, "returning to torn from") {
		return "Torn"
	}

	// Default cases
	if strings.Contains(descLower, "okay") || strings.Contains(descLower, "torn") {
		return "Torn"
	}

	// Hospital without location qualifier defaults to Torn
	if strings.Contains(descLower, "in hospital for") {
		// Check if it's NOT one of the specific country hospitals
		isCountryHospital := false
		for adjective := range hospitalMappings {
			if strings.Contains(descLower, "in a "+adjective+" hospital") {
				isCountryHospital = true
				break
			}
		}
		if !isCountryHospital {
			return "Torn"
		}
	}

	// Return original description if no pattern matches
	return description
}

// getTravelDestinationForCalculation returns the destination to use for travel time calculations
// For "Returning to Torn from X", returns X (the origin country)
// For other travel, returns the parsed location
func (wp *WarProcessor) getTravelDestinationForCalculation(description, parsedLocation string) string {
	if parsedLocation != "Torn" {
		return parsedLocation // Normal travel to foreign country
	}
	
	// For "Returning to Torn from X" cases, extract X for travel time calculation
	descLower := strings.ToLower(description)
	if strings.Contains(descLower, "returning to torn from") {
		// Extract the country name after "from "
		locations := []string{
			"Mexico", "Cayman Islands", "Canada", "Hawaii", "United Kingdom",
			"Argentina", "Switzerland", "Japan", "China", "UAE",
			"South Africa",
		}
		
		for _, location := range locations {
			if strings.Contains(descLower, strings.ToLower(location)) {
				log.Debug().
					Str("description", description).
					Str("origin_country", location).
					Msg("Detected return journey, using origin for travel time")
				return location
			}
		}
	}
	
	return parsedLocation
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

// getTravelTime returns travel duration based on destination and travel type
func (wp *WarProcessor) getTravelTime(destination string, travelType string) time.Duration {
	// Travel times in minutes
	regularTimes := map[string]int{
		"Mexico":          26,
		"Cayman Islands":  35,
		"Canada":          41,
		"Hawaii":          134, // 2h 14m
		"United Kingdom":  159, // 2h 39m
		"Argentina":       167, // 2h 47m
		"Switzerland":     175, // 2h 55m
		"Japan":           225, // 3h 45m
		"China":           242, // 4h 2m
		"UAE":             271, // 4h 31m
		"South Africa":    297, // 4h 57m
	}

	airstripTimes := map[string]int{
		"Mexico":          18,
		"Cayman Islands":  25,
		"Canada":          29,
		"Hawaii":          94,  // 1h 34m
		"United Kingdom":  111, // 1h 51m
		"Argentina":       117, // 1h 57m
		"Switzerland":     123, // 2h 3m
		"Japan":           158, // 2h 38m
		"China":           169, // 2h 49m
		"UAE":             190, // 3h 10m
		"South Africa":    208, // 3h 28m
	}

	var minutes int
	if travelType == "airstrip" {
		minutes = airstripTimes[destination]
	} else {
		minutes = regularTimes[destination]
	}

	if minutes == 0 {
		log.Warn().
			Str("destination", destination).
			Str("travel_type", travelType).
			Msg("Unknown travel destination, using default time")
		return 30 * time.Minute // Default fallback
	}

	return time.Duration(minutes) * time.Minute
}

// formatTravelTime formats duration as HH:MM:SS
func (wp *WarProcessor) formatTravelTime(d time.Duration) string {
	if d <= 0 {
		return "00:00:00"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// TravelTimeData holds calculated travel timing information
type TravelTimeData struct {
	Departure string
	Arrival   string
	Countdown string
}

// calculateTravelTimes calculates travel departure, arrival and countdown for a user
func (wp *WarProcessor) calculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time) *TravelTimeData {
	// Get travel duration based on destination and travel type
	travelDuration := wp.getTravelTime(destination, travelType)
	
	// Assume they departed 50% through the last cycle interval
	cycleInterval := wp.config.UpdateInterval
	estimatedDepartureTime := currentTime.Add(-cycleInterval / 2)
	arrivalTime := estimatedDepartureTime.Add(travelDuration)
	
	// Calculate countdown
	timeRemaining := arrivalTime.Sub(currentTime)
	countdown := wp.formatTravelTime(timeRemaining)
	
	// If they've already arrived, show as completed
	if timeRemaining <= 0 {
		countdown = "00:00:00"
	}
	
	log.Debug().
		Int("user_id", userID).
		Str("destination", destination).
		Str("travel_type", travelType).
		Dur("travel_duration", travelDuration).
		Str("countdown", countdown).
		Msg("Calculated travel times")
	
	return &TravelTimeData{
		Departure: estimatedDepartureTime.Format("2006-01-02 15:04:05"),
		Arrival:   arrivalTime.Format("2006-01-02 15:04:05"),
		Countdown: countdown,
	}
}

// calculateTravelTimesFromDeparture calculates arrival and countdown based on existing departure time
func (wp *WarProcessor) calculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr string, travelType string, currentTime time.Time) *TravelTimeData {
	// Parse existing departure time
	departureTime, err := time.Parse("2006-01-02 15:04:05", departureStr)
	if err != nil {
		log.Warn().
			Err(err).
			Str("departure_str", departureStr).
			Int("user_id", userID).
			Msg("Failed to parse existing departure time")
		return nil
	}

	// Get travel destination for calculation (handle return journeys)
	travelDestination := wp.getTravelDestinationForCalculation("", destination) // Don't have original description, use parsed location
	
	// Get travel duration based on destination and travel type
	travelDuration := wp.getTravelTime(travelDestination, travelType)
	arrivalTime := departureTime.Add(travelDuration)
	
	// Calculate countdown
	timeRemaining := arrivalTime.Sub(currentTime)
	countdown := wp.formatTravelTime(timeRemaining)
	
	// If they've already arrived, show as completed
	if timeRemaining <= 0 {
		countdown = "00:00:00"
	}
	
	log.Debug().
		Int("user_id", userID).
		Str("destination", travelDestination).
		Str("travel_type", travelType).
		Dur("travel_duration", travelDuration).
		Str("departure", departureStr).
		Str("countdown", countdown).
		Msg("Recalculated travel times from existing departure")
	
	return &TravelTimeData{
		Departure: departureStr, // Keep original departure time
		Arrival:   arrivalTime.Format("2006-01-02 15:04:05"),
		Countdown: countdown,
	}
}

// readExistingTravelData reads existing travel records from sheet to preserve departure times
func (wp *WarProcessor) readExistingTravelData(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.TravelRecord, error) {
	// Read all data from the sheet (starting from row 2 to skip headers)
	range_ := fmt.Sprintf("'%s'!A2:G", sheetName)
	values, err := wp.sheetsClient.ReadSheet(ctx, spreadsheetID, range_)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing travel data: %w", err)
	}

	existingData := make(map[string]app.TravelRecord)
	
	for _, row := range values {
		if len(row) < 7 { // Need all 7 columns: Name, Level, Location, State, Departure, Arrival, Countdown
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

		// Parse other string fields
		if location, ok := row[2].(string); ok {
			record.Location = location
		}
		if state, ok := row[3].(string); ok {
			record.State = state
		}
		if departure, ok := row[4].(string); ok {
			record.Departure = departure
		}
		if arrival, ok := row[5].(string); ok {
			record.Arrival = arrival
		}
		if countdown, ok := row[6].(string); ok {
			record.Countdown = countdown
		}

		existingData[name] = record
	}

	log.Debug().
		Int("existing_records", len(existingData)).
		Str("sheet_name", sheetName).
		Msg("Read existing travel data")

	return existingData, nil
}

// processTravelStatus fetches and processes travel status for a faction
func (wp *WarProcessor) processTravelStatus(ctx context.Context, war *app.War, factionID int, spreadsheetID string) error {
	// Determine if this is our faction or enemy faction for logging
	isOurFaction := factionID == wp.ourFactionID
	factionType := "enemy faction"
	if isOurFaction {
		factionType = "our faction"
	}
	
	log.Debug().
		Int("faction_id", factionID).
		Str("faction_type", factionType).
		Msg("Processing travel status")

	// Ensure travel status sheet exists
	travelSheetName, err := wp.sheetsClient.EnsureTravelStatusSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		return fmt.Errorf("failed to ensure travel status sheet: %w", err)
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

	// Fetch faction basic data
	factionData, err := wp.tornClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		return fmt.Errorf("failed to fetch faction basic data: %w", err)
	}
	
	// Get faction name from war data
	factionName := wp.getFactionName(war, factionID)
	
	// Load previous member states from persistent storage
	previousStateSheetName, err := wp.sheetsClient.EnsurePreviousStateSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		log.Error().
			Err(err).
			Int("faction_id", factionID).
			Msg("Failed to ensure previous state sheet - skipping state change detection")
	} else {
		// Load previous states
		previousMembers, err := wp.sheetsClient.LoadPreviousMemberStates(ctx, spreadsheetID, previousStateSheetName)
		if err != nil {
			log.Warn().
				Err(err).
				Int("faction_id", factionID).
				Str("sheet_name", previousStateSheetName).
				Msg("Failed to load previous member states - skipping state change detection")
		} else if len(previousMembers) > 0 {
			log.Debug().
				Int("faction_id", factionID).
				Int("previous_members", len(previousMembers)).
				Int("current_members", len(factionData.Members)).
				Msg("Processing state changes")
				
			if err := wp.processStateChanges(ctx, factionID, factionName, previousMembers, factionData.Members, spreadsheetID); err != nil {
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

	// Convert faction members to travel records with travel time tracking
	currentTime := time.Now()
	var travelRecords []app.TravelRecord
	
	for userIDStr, member := range factionData.Members {
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
		location := wp.parseLocation(member.Status.Description)
		
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

		// Check existing record for state transition logic
		existingRecord, hasExisting := existingTravelData[member.Name]
		previousState := ""
		if hasExisting {
			previousState = existingRecord.State
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
					Str("new_state", state).
					Msg("State transition from Traveling - clearing departure/arrival times")
			}
			// Departure and Arrival remain empty for Hospital state
			
		} else if strings.EqualFold(state, "Traveling") {
			if userID > 0 {
				if strings.EqualFold(previousState, "Traveling") && hasExisting && existingRecord.Departure != "" {
					// Still traveling - preserve departure/arrival, only update countdown
					record.Departure = existingRecord.Departure
					record.Arrival = existingRecord.Arrival
					
					// Recalculate only the countdown
					if travelData := wp.calculateTravelTimesFromDeparture(ctx, userID, location, existingRecord.Departure, member.Status.TravelType, currentTime); travelData != nil {
						record.Countdown = travelData.Countdown
					}
					
					log.Debug().
						Str("player", member.Name).
						Str("departure", record.Departure).
						Str("arrival", record.Arrival).
						Str("countdown", record.Countdown).
						Msg("Still traveling - preserved departure/arrival, updated countdown only")
						
				} else if hasExisting && !strings.EqualFold(previousState, "Traveling") {
					// State transition TO Traveling - set new departure/arrival times
					travelDestination := wp.getTravelDestinationForCalculation(member.Status.Description, location)
					if travelData := wp.calculateTravelTimes(ctx, userID, travelDestination, member.Status.TravelType, currentTime); travelData != nil {
						record.Departure = travelData.Departure
						record.Arrival = travelData.Arrival
						record.Countdown = travelData.Countdown
						
						log.Info().
							Str("player", member.Name).
							Str("previous_state", previousState).
							Str("new_state", state).
							Str("destination", travelDestination).
							Str("departure", record.Departure).
							Msg("State transition to Traveling - set new departure/arrival times")
					}
				} else {
					// No existing data (fresh start) or already traveling with no departure time
					// Leave departure/arrival blank since we didn't observe the transition
					log.Debug().
						Str("player", member.Name).
						Str("previous_state", previousState).
						Str("new_state", state).
						Bool("has_existing", hasExisting).
						Msg("Player traveling but no observed state transition - leaving times blank")
				}
			}
		} else {
			// Not traveling and not in hospital - clear travel times if transitioning from Traveling
			if strings.EqualFold(previousState, "Traveling") {
				log.Debug().
					Str("player", member.Name).
					Str("previous_state", previousState).
					Str("new_state", state).
					Msg("State transition from Traveling - clearing departure/arrival times")
			}
			// Departure, Arrival, and Countdown remain empty for non-traveling states
		}

		travelRecords = append(travelRecords, record)
	}

	// Sort records by level (descending), then alphabetically by name
	sort.Slice(travelRecords, func(i, j int) bool {
		if travelRecords[i].Level == travelRecords[j].Level {
			return travelRecords[i].Name < travelRecords[j].Name
		}
		return travelRecords[i].Level > travelRecords[j].Level
	})

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

// normalizeHospitalDescription removes countdown from hospital descriptions for comparison
func (wp *WarProcessor) normalizeHospitalDescription(description string) string {
	// Match "In hospital for X hrs Y mins" and replace with "In hospital"
	hospitalRegex := regexp.MustCompile(`(?i)^(in hospital).*`)
	if hospitalRegex.MatchString(description) {
		return "In hospital"
	}
	return description
}

// hasStatusChanged compares two member states to detect changes, ignoring hospital countdown changes
func (wp *WarProcessor) hasStatusChanged(oldMember, newMember app.FactionMember) bool {
	// Check LastAction.Status changes
	if oldMember.LastAction.Status != newMember.LastAction.Status {
		return true
	}

	// Check Status fields, normalizing hospital descriptions
	oldDesc := wp.normalizeHospitalDescription(oldMember.Status.Description)
	newDesc := wp.normalizeHospitalDescription(newMember.Status.Description)
	
	if oldDesc != newDesc ||
		oldMember.Status.State != newMember.Status.State ||
		oldMember.Status.Color != newMember.Status.Color ||
		oldMember.Status.Details != newMember.Status.Details ||
		oldMember.Status.TravelType != newMember.Status.TravelType ||
		oldMember.Status.PlaneImageType != newMember.Status.PlaneImageType {
		return true
	}

	// Check Until field (but be careful with nil pointers)
	if (oldMember.Status.Until == nil) != (newMember.Status.Until == nil) {
		return true
	}
	if oldMember.Status.Until != nil && newMember.Status.Until != nil {
		if *oldMember.Status.Until != *newMember.Status.Until {
			return true
		}
	}

	return false
}

// processStateChanges detects and records state changes for faction members
func (wp *WarProcessor) processStateChanges(ctx context.Context, factionID int, factionName string, oldMembers, newMembers map[string]app.FactionMember, spreadsheetID string) error {
	// Ensure state change records sheet exists
	sheetName, err := wp.sheetsClient.EnsureStateChangeRecordsSheet(ctx, spreadsheetID, factionID)
	if err != nil {
		return fmt.Errorf("failed to ensure state change records sheet: %w", err)
	}

	currentTime := time.Now()
	stateChanges := 0

	// Check each current member against previous state
	for userIDStr, newMember := range newMembers {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Warn().
				Str("user_id_str", userIDStr).
				Str("member_name", newMember.Name).
				Msg("Failed to parse user ID for state change tracking")
			continue
		}

		// Check if we have previous state for this member
		if oldMember, exists := oldMembers[userIDStr]; exists {
			// Compare states
			if wp.hasStatusChanged(oldMember, newMember) {
				// Create state change record
				record := app.StateChangeRecord{
					Timestamp:            currentTime,
					MemberName:           newMember.Name,
					MemberID:             userID,
					FactionName:          factionName,
					FactionID:            factionID,
					LastActionStatus:     newMember.LastAction.Status,
					StatusDescription:    newMember.Status.Description,
					StatusState:          newMember.Status.State,
					StatusColor:          newMember.Status.Color,
					StatusDetails:        newMember.Status.Details,
					StatusUntil:          newMember.Status.Until,
					StatusTravelType:     newMember.Status.TravelType,
					StatusPlaneImageType: newMember.Status.PlaneImageType,
				}

				// Add record to sheet
				if err := wp.sheetsClient.AddStateChangeRecord(ctx, spreadsheetID, sheetName, record); err != nil {
					log.Error().
						Err(err).
						Str("member_name", newMember.Name).
						Int("member_id", userID).
						Msg("Failed to add state change record")
				} else {
					stateChanges++
					log.Debug().
						Str("member_name", newMember.Name).
						Int("member_id", userID).
						Str("old_state", oldMember.Status.State).
						Str("new_state", newMember.Status.State).
						Str("old_last_action", oldMember.LastAction.Status).
						Str("new_last_action", newMember.LastAction.Status).
						Msg("Recorded state change")
				}
			}
		}
		// Note: We don't record first-time appearances as state changes
	}

	if stateChanges > 0 {
		log.Info().
			Int("faction_id", factionID).
			Str("faction_name", factionName).
			Int("state_changes", stateChanges).
			Str("sheet_name", sheetName).
			Msg("Processed member state changes")
	}

	return nil
}