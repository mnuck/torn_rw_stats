package services

import (
	"context"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/status"
	"torn_rw_stats/internal/domain/travel"
	"torn_rw_stats/internal/processing"

	"github.com/rs/zerolog/log"
)

// StatusV2Service handles conversion of StateRecords to StatusV2Records,
// tracking departure times for traveling players and calculating arrival predictions.
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
	// Use domain functions for pure calculations
	existing := status.GetExistingRecord(stateRecord.FactionID, stateRecord.MemberID, stateRecord.MemberName, existingData)
	level := status.ResolveLevel(stateRecord.MemberID, factionMembers, existing)
	location := s.calculateLocation(stateRecord)

	travelInfo := s.calculateTravelInfo(ctx, stateRecord, existing, departureMap, currentTime, location)

	return s.buildStatusV2Record(stateRecord, level, location, travelInfo)
}

// buildStatusV2Record constructs the final StatusV2Record
func (s *StatusV2Service) buildStatusV2Record(stateRecord app.StateRecord, level int, location string, travelInfo TravelInfo) app.StatusV2Record {
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
