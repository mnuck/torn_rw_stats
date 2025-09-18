package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/processing"

	"github.com/rs/zerolog/log"
)

// StatusV2Service handles conversion of StateRecords to StatusV2Records
// and tracks departure times for traveling players
type StatusV2Service struct {
	sheetsClient      processing.SheetsClientInterface
	locationService   *travel.LocationService
	travelTimeService *travel.TravelTimeService
}

// NewStatusV2Service creates a new Status v2 service
func NewStatusV2Service(sheetsClient processing.SheetsClientInterface) *StatusV2Service {
	return &StatusV2Service{
		sheetsClient:      sheetsClient,
		locationService:   travel.NewLocationService(),
		travelTimeService: travel.NewTravelTimeService(),
	}
}

// ConvertStateRecordsToStatusV2 converts StateRecords to StatusV2Records
// incorporating departure time tracking and countdown calculations
func (s *StatusV2Service) ConvertStateRecordsToStatusV2(ctx context.Context, spreadsheetID string, stateRecords []app.StateRecord, factionMembers map[string]app.FactionMember, factionID int) ([]app.StatusV2Record, error) {
	log.Info().
		Int("faction_id", factionID).
		Int("input_state_records", len(stateRecords)).
		Int("faction_members", len(factionMembers)).
		Msg("Starting StateRecord to StatusV2 conversion")

	var records []app.StatusV2Record

	// Get existing departure data to preserve manual adjustments
	existingData, err := s.getExistingStatusV2Data(ctx, spreadsheetID, factionID)
	if err != nil {
		log.Warn().Err(err).Int("faction_id", factionID).Msg("Failed to get existing Status v2 data, will use defaults")
		existingData = make(map[string]app.StatusV2Record)
	}

	log.Debug().
		Int("faction_id", factionID).
		Int("existing_status_v2_records", len(existingData)).
		Msg("Retrieved existing Status v2 data")

	// Get travel state changes for departure time tracking
	departureMap, err := s.buildDepartureMap(ctx, spreadsheetID, stateRecords)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to build departure map, will use current timestamp for traveling players")
		departureMap = make(map[string]time.Time)
	}

	currentTime := time.Now().UTC()

	for i, stateRecord := range stateRecords {
		log.Debug().
			Int("faction_id", factionID).
			Int("record_index", i).
			Str("member_id", stateRecord.MemberID).
			Str("member_name", stateRecord.MemberName).
			Str("status_state", stateRecord.StatusState).
			Msg("Converting individual state record")

		// Skip members who are no longer in the faction
		if _, exists := factionMembers[stateRecord.MemberID]; !exists {
			log.Debug().
				Int("faction_id", factionID).
				Str("member_id", stateRecord.MemberID).
				Str("member_name", stateRecord.MemberName).
				Msg("Skipping member who is no longer in faction")
			continue
		}

		record := s.convertSingleStateRecord(ctx, stateRecord, factionMembers, existingData, departureMap, currentTime)
		records = append(records, record)
	}

	log.Info().
		Int("faction_id", factionID).
		Int("output_status_v2_records", len(records)).
		Msg("Completed StateRecord to StatusV2 conversion")

	return records, nil
}

// convertSingleStateRecord converts a single StateRecord to StatusV2Record
func (s *StatusV2Service) convertSingleStateRecord(ctx context.Context, stateRecord app.StateRecord, factionMembers map[string]app.FactionMember, existingData map[string]app.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time) app.StatusV2Record {
	existing := s.getExistingRecord(stateRecord, existingData)
	level := s.resolveLevel(stateRecord, factionMembers, existing)
	location := s.calculateLocation(stateRecord)
	countdown := s.calculateCountdown(stateRecord.StatusUntil, currentTime)

	travelInfo := s.calculateTravelInfo(ctx, stateRecord, existing, departureMap, currentTime, location)

	return s.buildStatusV2Record(stateRecord, level, location, countdown, travelInfo)
}

// getExistingRecord finds existing data for a member using both ID and name keys
func (s *StatusV2Service) getExistingRecord(stateRecord app.StateRecord, existingData map[string]app.StatusV2Record) *app.StatusV2Record {
	memberKey := fmt.Sprintf("%s_%s", stateRecord.FactionID, stateRecord.MemberID)
	nameKey := fmt.Sprintf("%s_%s", stateRecord.FactionID, stateRecord.MemberName)

	if existing, hasExisting := existingData[memberKey]; hasExisting {
		return &existing
	}
	if existing, hasExisting := existingData[nameKey]; hasExisting {
		return &existing
	}
	return nil
}

// resolveLevel determines the member's level from faction data or existing records
func (s *StatusV2Service) resolveLevel(stateRecord app.StateRecord, factionMembers map[string]app.FactionMember, existing *app.StatusV2Record) int {
	if member, exists := factionMembers[stateRecord.MemberID]; exists {
		return member.Level
	}
	if existing != nil {
		return existing.Level
	}
	return 0
}

// TravelInfo holds travel-related data for a member
type TravelInfo struct {
	Departure       string
	Arrival         string
	BusinessArrival string
	Countdown       string
}

// calculateTravelInfo handles all travel-related calculations and preserves manual adjustments
func (s *StatusV2Service) calculateTravelInfo(ctx context.Context, stateRecord app.StateRecord, existing *app.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time, location string) TravelInfo {
	if stateRecord.StatusState != "Traveling" {
		return TravelInfo{} // Clear travel data for non-traveling members
	}

	memberKey := fmt.Sprintf("%s_%s", stateRecord.FactionID, stateRecord.MemberID)
	departure := s.calculateDeparture(memberKey, existing, departureMap, currentTime)

	// Calculate arrival times using TravelTimeService
	arrival, businessArrival, countdown := s.calculateArrivalTimes(ctx, stateRecord, existing, departure, location, currentTime)

	// Preserve manual adjustments
	return s.applyManualAdjustments(existing, TravelInfo{
		Departure:       departure,
		Arrival:         arrival,
		BusinessArrival: businessArrival,
		Countdown:       countdown,
	})
}

// calculateDeparture determines the departure time for a traveling member
func (s *StatusV2Service) calculateDeparture(memberKey string, existing *app.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time) string {
	if departureTime, hasDeparture := departureMap[memberKey]; hasDeparture {
		return departureTime.Format("2006-01-02 15:04:05")
	}
	if existing != nil && existing.Departure != "" {
		return existing.Departure
	}
	return currentTime.Format("2006-01-02 15:04:05")
}

// calculateArrivalTimes uses TravelTimeService to calculate arrival times and countdown
func (s *StatusV2Service) calculateArrivalTimes(ctx context.Context, stateRecord app.StateRecord, existing *app.StatusV2Record, departure, location string, currentTime time.Time) (string, string, string) {
	if departure == "" {
		return "", "", ""
	}

	memberID := 0
	if id, err := strconv.Atoi(stateRecord.MemberID); err == nil {
		memberID = id
	}

	existingArrival := ""
	if existing != nil {
		existingArrival = existing.Arrival
	}

	travelData := s.travelTimeService.CalculateTravelTimesFromDeparture(
		ctx,
		memberID,
		location,
		departure,
		existingArrival,
		stateRecord.StatusTravelType,
		currentTime,
		s.locationService,
		stateRecord.StatusDescription,
	)

	if travelData != nil {
		return travelData.Arrival, travelData.BusinessArrival, travelData.Countdown
	}
	return "", "", ""
}

// applyManualAdjustments preserves any manual adjustments from existing data
func (s *StatusV2Service) applyManualAdjustments(existing *app.StatusV2Record, calculated TravelInfo) TravelInfo {
	if existing == nil {
		return calculated
	}

	result := calculated
	if existing.Departure != "" {
		result.Departure = existing.Departure
	}
	if existing.Arrival != "" {
		result.Arrival = existing.Arrival
	}
	if existing.BusinessArrival != "" {
		result.BusinessArrival = existing.BusinessArrival
	}
	return result
}

// buildStatusV2Record constructs the final StatusV2Record
func (s *StatusV2Service) buildStatusV2Record(stateRecord app.StateRecord, level int, location, countdown string, travelInfo TravelInfo) app.StatusV2Record {
	return app.StatusV2Record{
		Name:            stateRecord.MemberName,
		MemberID:        stateRecord.MemberID,
		Level:           level,
		State:           stateRecord.LastActionStatus,
		Status:          stateRecord.StatusState,
		Location:        location,
		Countdown:       travelInfo.Countdown,
		Departure:       travelInfo.Departure,
		Arrival:         travelInfo.Arrival,
		BusinessArrival: travelInfo.BusinessArrival,
		Until:           stateRecord.StatusUntil,
	}
}

// calculateLocation determines the location based on member state using LocationService
func (s *StatusV2Service) calculateLocation(stateRecord app.StateRecord) string {
	// Use the LocationService to parse location from status description
	// This handles all patterns: hospitals, travel, locations, etc.
	return s.locationService.ParseLocation(stateRecord.StatusDescription)
}

// calculateCountdown calculates countdown string from StatusUntil timestamp
func (s *StatusV2Service) calculateCountdown(statusUntil time.Time, currentTime time.Time) string {
	if statusUntil.IsZero() {
		return ""
	}

	duration := statusUntil.Sub(currentTime)
	if duration <= 0 {
		return "0:00:00"
	}

	// Format as H:MM:SS
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// buildDepartureMap builds a map of member departure times from state changes
func (s *StatusV2Service) buildDepartureMap(ctx context.Context, spreadsheetID string, currentStateRecords []app.StateRecord) (map[string]time.Time, error) {
	departureMap := make(map[string]time.Time)

	// Read all state records from Changed States sheet
	allStateRecords, err := s.ReadAllStateRecords(ctx, spreadsheetID)
	if err != nil {
		return departureMap, fmt.Errorf("failed to read state records: %w", err)
	}

	// For each current traveling member, find their most recent transition to traveling
	for _, currentRecord := range currentStateRecords {
		if currentRecord.StatusState != "Traveling" {
			continue
		}

		memberKey := fmt.Sprintf("%s_%s", currentRecord.FactionID, currentRecord.MemberID)
		// Parse the current destination location instead of using raw status description
		currentParsedLocation := s.locationService.ParseLocation(currentRecord.StatusDescription)
		departureTime := s.findMostRecentTravelingTransition(allStateRecords, currentRecord.MemberID, currentParsedLocation)

		if !departureTime.IsZero() {
			departureMap[memberKey] = departureTime
		}
	}

	return departureMap, nil
}

// findMostRecentTravelingTransition finds when a member most recently started traveling to their current destination
func (s *StatusV2Service) findMostRecentTravelingTransition(allRecords []app.StateRecord, memberID, currentDestination string) time.Time {
	memberRecords := s.getMemberRecordsChronologically(allRecords, memberID)
	return s.findLastDepartureToDestination(memberRecords, currentDestination)
}

// getMemberRecordsChronologically filters and sorts records for a specific member
func (s *StatusV2Service) getMemberRecordsChronologically(allRecords []app.StateRecord, memberID string) []app.StateRecord {
	var records []app.StateRecord
	for _, record := range allRecords {
		if record.MemberID == memberID {
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	return records
}

// findLastDepartureToDestination finds the most recent departure time to a specific destination
func (s *StatusV2Service) findLastDepartureToDestination(records []app.StateRecord, destination string) time.Time {
	var lastDeparture time.Time

	for i := 0; i < len(records); i++ {
		current := records[i]
		if current.StatusState != "Traveling" {
			continue
		}

		currentDestination := s.locationService.ParseLocation(current.StatusDescription)
		if currentDestination != destination {
			continue
		}

		// Check if this is a new journey (different from previous travel)
		if s.isNewJourneyToDestination(records, i, destination) {
			lastDeparture = current.Timestamp
		}
	}

	return lastDeparture
}

// isNewJourneyToDestination checks if this record represents a new journey to the destination
func (s *StatusV2Service) isNewJourneyToDestination(records []app.StateRecord, currentIndex int, destination string) bool {
	if currentIndex == 0 {
		return true // First record is always a new journey
	}

	current := records[currentIndex]
	previous := records[currentIndex-1]

	// New journey if previous status was not traveling
	if previous.StatusState != "Traveling" {
		return true
	}

	// New journey if previous destination was different
	previousDestination := s.locationService.ParseLocation(previous.StatusDescription)
	currentDestination := s.locationService.ParseLocation(current.StatusDescription)

	return previousDestination != currentDestination
}

// getExistingStatusV2Data reads existing Status v2 data to preserve manual adjustments
func (s *StatusV2Service) getExistingStatusV2Data(ctx context.Context, spreadsheetID string, factionID int) (map[string]app.StatusV2Record, error) {
	sheetName := fmt.Sprintf("Status v2 - %d", factionID)
	rangeSpec := fmt.Sprintf("%s!A2:J", sheetName)

	values, err := s.sheetsClient.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing Status v2 data: %w", err)
	}

	data := make(map[string]app.StatusV2Record)
	factionIDStr := strconv.Itoa(factionID)

	for _, row := range values {
		if len(row) < 8 {
			continue
		}

		// Extract member name and create key
		name, ok := row[0].(string)
		if !ok {
			continue
		}

		// We'll use name as key since MemberID isn't in the sheet
		memberKey := fmt.Sprintf("%s_%s", factionIDStr, name)

		level := 0
		if levelStr, ok := row[1].(string); ok {
			if l, err := strconv.Atoi(levelStr); err == nil {
				level = l
			}
		}

		// Parse Until timestamp from column 9 (column J)
		var until time.Time
		if len(row) > 9 {
			if untilStr := getString(row, 9); untilStr != "" {
				if parsedUntil, err := time.Parse("2006-01-02 15:04:05", untilStr); err == nil {
					until = parsedUntil.UTC()
				}
			}
		}

		record := app.StatusV2Record{
			Name:            name,
			MemberID:        "", // MemberID not stored in spreadsheet, populated from StateRecord
			Level:           level,
			State:           getString(row, 2),
			Status:          getString(row, 3),
			Location:        getString(row, 4),
			Countdown:       getString(row, 5),
			Departure:       getString(row, 6),
			Arrival:         getString(row, 7),
			BusinessArrival: getString(row, 8), // Column I
			Until:           until,
		}

		data[memberKey] = record
	}

	return data, nil
}

// ReadAllStateRecords reads all state records from the Changed States sheet
func (s *StatusV2Service) ReadAllStateRecords(ctx context.Context, spreadsheetID string) ([]app.StateRecord, error) {
	sheetName := "Changed States"
	rangeSpec := fmt.Sprintf("%s!A2:L", sheetName)

	log.Info().
		Str("sheet_name", sheetName).
		Str("range_spec", rangeSpec).
		Msg("Reading state records from Changed States sheet")

	values, err := s.sheetsClient.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read Changed States sheet: %w", err)
	}

	log.Info().
		Int("raw_rows_read", len(values)).
		Msg("Successfully read raw data from Changed States sheet")

	var records []app.StateRecord
	validRows := 0
	for i, row := range values {
		if len(row) < 8 {
			log.Debug().
				Int("row_index", i).
				Int("row_length", len(row)).
				Interface("row_sample", row).
				Msg("Skipping row with insufficient columns - showing data")
			continue
		}

		record, err := s.parseStateRecordFromRow(row)
		if err != nil {
			log.Warn().Err(err).Interface("row", row).Msg("Failed to parse state record from row")
			continue
		}

		records = append(records, record)
		validRows++
	}

	log.Info().
		Int("total_rows_processed", len(values)).
		Int("valid_state_records", validRows).
		Int("final_records_count", len(records)).
		Msg("Completed reading Changed States data")

	return records, nil
}

// parseStateRecordFromRow parses a spreadsheet row into a StateRecord
func (s *StatusV2Service) parseStateRecordFromRow(row []interface{}) (app.StateRecord, error) {
	var record app.StateRecord

	// Actual Changed States format: [Timestamp, Member ID, Member Name, Faction ID, Faction Name, Last Action Status, Status Description, Status State, Status Until, Status Travel Type]
	// NOTE: The sheet does NOT have Date and Time columns, so indices are shifted left by 2 from the header definition

	// Parse timestamp from column 0 - this is already formatted as "2025-09-15 1:08:57"
	if timestampStr, ok := row[0].(string); ok {
		if timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
			record.Timestamp = timestamp.UTC()
		}
	}

	record.MemberID = getString(row, 1)
	record.MemberName = getString(row, 2)
	record.FactionID = getString(row, 3)
	record.FactionName = getString(row, 4)
	record.LastActionStatus = getString(row, 5)
	record.StatusDescription = getString(row, 6)
	record.StatusState = getString(row, 7)

	// Parse StatusUntil from column 8 (optional - only present for some status types)
	if len(row) > 8 {
		if statusUntilStr := getString(row, 8); statusUntilStr != "" {
			if statusUntil, err := time.Parse("2006-01-02 15:04:05", statusUntilStr); err == nil {
				record.StatusUntil = statusUntil.UTC()
			}
		}
	}

	// Parse StatusTravelType from column 9 (optional - only present for traveling status)
	if len(row) > 9 {
		record.StatusTravelType = getString(row, 9)
	}

	return record, nil
}

// getString safely gets a string from a spreadsheet row
func getString(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	if str, ok := row[index].(string); ok {
		return str
	}
	return ""
}

// ConvertToJSON converts StatusV2Records to the JSON export format
func (s *StatusV2Service) ConvertToJSON(records []app.StatusV2Record, factionName string, currentTime time.Time, updateInterval time.Duration) app.StatusV2JSON {
	locations := s.groupRecordsByLocation(records)

	return app.StatusV2JSON{
		Faction:   factionName,
		Updated:   currentTime.Format(time.RFC3339),
		Interval:  int(updateInterval.Seconds()),
		Locations: locations,
	}
}

// groupRecordsByLocation organizes records by location and filters empty locations
func (s *StatusV2Service) groupRecordsByLocation(records []app.StatusV2Record) map[string]app.LocationData {
	locations := make(map[string]app.LocationData)

	for _, record := range records {
		if record.Location == "" {
			continue
		}

		member := s.createJSONMember(record)
		s.addMemberToLocation(locations, record.Location, member, s.isTraveling(record))
	}

	// Filter out empty locations
	return s.filterEmptyLocations(locations)
}

// createJSONMember creates a JSONMember from a StatusV2Record
func (s *StatusV2Service) createJSONMember(record app.StatusV2Record) app.JSONMember {
	member := app.JSONMember{
		Name:     record.Name,
		MemberID: record.MemberID,
		Level:    record.Level,
		State:    record.State,
	}

	if !record.Until.IsZero() {
		member.Until = record.Until.Format(time.RFC3339)
	}

	if s.isTraveling(record) {
		s.addTravelingFields(&member, record)
	} else {
		s.addLocatedFields(&member, record)
	}

	return member
}

// isTraveling determines if a member is currently traveling
func (s *StatusV2Service) isTraveling(record app.StatusV2Record) bool {
	return strings.Contains(strings.ToLower(record.Status), "traveling")
}

// addTravelingFields adds travel-specific fields to a JSON member
func (s *StatusV2Service) addTravelingFields(member *app.JSONMember, record app.StatusV2Record) {
	if record.Countdown != "" && record.Countdown != "00:00:00" {
		member.Countdown = strings.TrimPrefix(record.Countdown, "'")
	}
	if record.Arrival != "" {
		member.Arrival = record.Arrival
	}
	if record.BusinessArrival != "" {
		member.BusinessArrival = record.BusinessArrival
	}
}

// addLocatedFields adds location-specific fields to a JSON member
func (s *StatusV2Service) addLocatedFields(member *app.JSONMember, record app.StatusV2Record) {
	if record.Status != "" && record.Status != "Okay" {
		member.Status = record.Status
		if record.Countdown != "" && record.Countdown != "00:00:00" {
			member.Countdown = strings.TrimPrefix(record.Countdown, "'")
		}
	}
}

// addMemberToLocation adds a member to the appropriate location array
func (s *StatusV2Service) addMemberToLocation(locations map[string]app.LocationData, location string, member app.JSONMember, isTraveling bool) {
	if _, exists := locations[location]; !exists {
		locations[location] = app.LocationData{
			Traveling: []app.JSONMember{},
			LocatedIn: []app.JSONMember{},
		}
	}

	locationData := locations[location]
	if isTraveling {
		locationData.Traveling = append(locationData.Traveling, member)
	} else {
		locationData.LocatedIn = append(locationData.LocatedIn, member)
	}
	locations[location] = locationData
}

// filterEmptyLocations removes locations with no members
func (s *StatusV2Service) filterEmptyLocations(locations map[string]app.LocationData) map[string]app.LocationData {
	filteredLocations := make(map[string]app.LocationData)
	for location, data := range locations {
		if len(data.Traveling) > 0 || len(data.LocatedIn) > 0 {
			filteredLocations[location] = data
		}
	}
	return filteredLocations
}
