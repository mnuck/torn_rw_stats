package services

import (
	"context"
	"time"

	"log/slog"
	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/status"
	"torn_rw_stats/internal/domain/travel"

	"torn_rw_stats/internal/application/ports"
)

// StatusV2Service handles conversion of StateRecords to StatusV2Records,
// tracking departure times for traveling players and calculating arrival predictions.
type StatusV2Service struct {
	sheetsClient      ports.SheetsClient
	bigqueryClient    ports.BigQueryClient // nil = disabled
	locationService   *travel.LocationService
	travelTimeService *travel.TravelTimeService
}

// NewStatusV2Service creates a new Status v2 service
func NewStatusV2Service(sheetsClient ports.SheetsClient) *StatusV2Service {
	return &StatusV2Service{
		sheetsClient:      sheetsClient,
		locationService:   travel.NewLocationService(),
		travelTimeService: travel.NewTravelTimeService(),
	}
}

// NewStatusV2ServiceWithBigQuery creates a Status v2 service that uses BigQuery for history lookups.
func NewStatusV2ServiceWithBigQuery(sheetsClient ports.SheetsClient, bqClient ports.BigQueryClient) *StatusV2Service {
	return &StatusV2Service{
		sheetsClient:      sheetsClient,
		bigqueryClient:    bqClient,
		locationService:   travel.NewLocationService(),
		travelTimeService: travel.NewTravelTimeService(),
	}
}

// ConvertStateRecordsToStatusV2 converts StateRecords to StatusV2Records
// incorporating departure time tracking and countdown calculations
func (s *StatusV2Service) ConvertStateRecordsToStatusV2(ctx context.Context, spreadsheetID string, stateRecords []domain.StateRecord, factionMembers map[string]domain.FactionMember, factionID int) ([]domain.StatusV2Record, error) {
	slog.Info("Starting StateRecord to StatusV2 conversion",
		"faction_id", factionID,
		"input_state_records", len(stateRecords),
		"faction_members", len(factionMembers))

	var records []domain.StatusV2Record

	// Get existing departure data to preserve manual adjustments
	existingData, err := s.getExistingStatusV2Data(ctx, spreadsheetID, factionID)
	if err != nil {
		slog.Warn("Failed to get existing Status v2 data, will use defaults", "err", err, "faction_id", factionID)
		existingData = make(map[string]domain.StatusV2Record)
	}

	slog.Debug("Retrieved existing Status v2 data",
		"faction_id", factionID,
		"existing_status_v2_records", len(existingData))

	// Get travel state changes for departure time tracking
	departureMap, err := s.buildDepartureMap(ctx, spreadsheetID, stateRecords)
	if err != nil {
		slog.Warn("Failed to build departure map, will use current timestamp for traveling players", "err", err)
		departureMap = make(map[string]time.Time)
	}

	currentTime := time.Now().UTC()

	for i, stateRecord := range stateRecords {
		slog.Debug("Converting individual state record",
			"faction_id", factionID,
			"record_index", i,
			"member_id", stateRecord.MemberID,
			"member_name", stateRecord.MemberName,
			"status_state", stateRecord.StatusState)

		// Skip members who are no longer in the faction
		if _, exists := factionMembers[stateRecord.MemberID]; !exists {
			slog.Debug("Skipping member who is no longer in faction",
				"faction_id", factionID,
				"member_id", stateRecord.MemberID,
				"member_name", stateRecord.MemberName)
			continue
		}

		record := s.convertSingleStateRecord(ctx, stateRecord, factionMembers, existingData, departureMap, currentTime)
		records = append(records, record)
	}

	slog.Info("Completed StateRecord to StatusV2 conversion",
		"faction_id", factionID,
		"output_status_v2_records", len(records))

	return records, nil
}

// convertSingleStateRecord converts a single StateRecord to StatusV2Record
func (s *StatusV2Service) convertSingleStateRecord(ctx context.Context, stateRecord domain.StateRecord, factionMembers map[string]domain.FactionMember, existingData map[string]domain.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time) domain.StatusV2Record {
	// Use domain functions for pure calculations
	existing := status.GetExistingRecord(stateRecord.FactionID, stateRecord.MemberID, stateRecord.MemberName, existingData)
	level := status.ResolveLevel(stateRecord.MemberID, factionMembers, existing)
	location := s.calculateLocation(stateRecord)

	travelInfo := s.calculateTravelInfo(ctx, stateRecord, existing, departureMap, currentTime, location)

	return s.buildStatusV2Record(stateRecord, level, location, travelInfo)
}

// buildStatusV2Record constructs the final StatusV2Record
func (s *StatusV2Service) buildStatusV2Record(stateRecord domain.StateRecord, level int, location string, travelInfo TravelInfo) domain.StatusV2Record {
	return domain.StatusV2Record{
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
func (s *StatusV2Service) calculateLocation(stateRecord domain.StateRecord) string {
	// Use the LocationService to parse location from status description
	// This handles all patterns: hospitals, travel, locations, etc.
	return s.locationService.ParseLocation(stateRecord.StatusDescription)
}
