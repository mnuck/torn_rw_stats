package services

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing"
	"torn_rw_stats/internal/sheets"

	"github.com/rs/zerolog/log"
)

// StateTrackingService handles the complete state tracking workflow
type StateTrackingService struct {
	tornClient   processing.TornClientInterface
	sheetsClient processing.SheetsClientInterface
	converter    *processing.StateRecordConverter
	comparator   *processing.StateRecordComparator
}

// NewStateTrackingService creates a new state tracking service
func NewStateTrackingService(tornClient processing.TornClientInterface, sheetsClient processing.SheetsClientInterface) *StateTrackingService {
	return &StateTrackingService{
		tornClient:   tornClient,
		sheetsClient: sheetsClient,
		converter:    processing.NewStateRecordConverter(),
		comparator:   processing.NewStateRecordComparator(),
	}
}

// ProcessStateChanges executes the complete state tracking workflow
func (s *StateTrackingService) ProcessStateChanges(ctx context.Context, spreadsheetID string, factionIDs []int) error {
	currentTime := time.Now().UTC()

	log.Info().
		Int("faction_count", len(factionIDs)).
		Msg("Starting state change processing")

	// Step 1: Get current StateRecords for all factions
	currentStateRecords, err := s.getCurrentStateRecords(ctx, factionIDs, currentTime)
	if err != nil {
		return fmt.Errorf("failed to get current state records: %w", err)
	}

	log.Debug().
		Int("current_records", len(currentStateRecords)).
		Msg("Retrieved current state records")

	// Step 2: Ensure Changed States sheet exists
	if err := s.ensureChangedStatesSheet(ctx, spreadsheetID); err != nil {
		return fmt.Errorf("failed to ensure Changed States sheet: %w", err)
	}

	// Step 3: Read existing records from Changed States sheet
	allPreviousStates, err := s.readChangedStatesSheet(ctx, spreadsheetID)
	if err != nil {
		return fmt.Errorf("failed to read Changed States sheet: %w", err)
	}

	log.Debug().
		Int("previous_records", len(allPreviousStates)).
		Msg("Read previous state records from sheet")

	// Step 4: Create previous state collection for comparison
	previousStateRecords := s.comparator.CreatePreviousStateCollection(currentStateRecords, allPreviousStates)

	log.Debug().
		Int("previous_for_comparison", len(previousStateRecords)).
		Msg("Created previous states collection for comparison")

	// Step 5: Compare states and find changes
	updatedStateRecords := s.comparator.FindChangedStates(currentStateRecords, s.mapToSlice(previousStateRecords))

	log.Info().
		Int("changed_states", len(updatedStateRecords)).
		Msg("Found state changes")

	// Step 6: Add updated records to sheet (if any)
	if len(updatedStateRecords) > 0 {
		if err := s.addStateRecords(ctx, spreadsheetID, updatedStateRecords); err != nil {
			return fmt.Errorf("failed to add state records to sheet: %w", err)
		}

		log.Info().
			Int("records_added", len(updatedStateRecords)).
			Msg("Successfully added state changes to Changed States sheet")
	} else {
		log.Info().Msg("No state changes detected - no records added")
	}

	return nil
}

// getCurrentStateRecords retrieves current state for all specified factions
func (s *StateTrackingService) getCurrentStateRecords(ctx context.Context, factionIDs []int, currentTime time.Time) ([]app.StateRecord, error) {
	var allRecords []app.StateRecord

	for _, factionID := range factionIDs {
		// Get faction data
		factionData, err := s.tornClient.GetFactionBasic(ctx, factionID)
		if err != nil {
			log.Error().
				Err(err).
				Int("faction_id", factionID).
				Msg("Failed to get faction data - skipping")
			continue
		}

		// Convert faction data to state records
		records := s.converter.ConvertFromFactionBasic(factionData, currentTime)

		allRecords = append(allRecords, records...)

		log.Debug().
			Int("faction_id", factionID).
			Int("member_count", len(records)).
			Msg("Retrieved state records for faction")
	}

	return allRecords, nil
}

// mapToSlice converts a map of StateRecords to a slice
func (s *StateTrackingService) mapToSlice(recordMap map[string]app.StateRecord) []app.StateRecord {
	var slice []app.StateRecord
	for _, record := range recordMap {
		slice = append(slice, record)
	}
	return slice
}

// ensureChangedStatesSheet creates the Changed States sheet if it doesn't exist
func (s *StateTrackingService) ensureChangedStatesSheet(ctx context.Context, spreadsheetID string) error {
	sheetName := "Changed States"

	exists, err := s.sheetsClient.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return fmt.Errorf("failed to check if Changed States sheet exists: %w", err)
	}

	if !exists {
		if err := s.sheetsClient.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return fmt.Errorf("failed to create Changed States sheet: %w", err)
		}

		// Initialize with headers
		headers := [][]interface{}{
			{
				"Timestamp", "Member ID", "Member Name",
				"Faction ID", "Faction Name", "Last Action Status", "Status Description",
				"Status State", "Status Until", "Status Travel Type",
			},
		}

		rangeSpec := fmt.Sprintf("%s!A1", sheetName)
		if err := s.sheetsClient.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
			return fmt.Errorf("failed to write Changed States headers: %w", err)
		}

		log.Info().Str("sheet_name", sheetName).Msg("Created and initialized Changed States sheet")
	}

	return nil
}

// readChangedStatesSheet reads all records from the Changed States sheet
func (s *StateTrackingService) readChangedStatesSheet(ctx context.Context, spreadsheetID string) ([]app.StateRecord, error) {
	sheetName := "Changed States"
	rangeSpec := fmt.Sprintf("%s!A2:J", sheetName) // Skip header row, only 10 columns now

	values, err := s.sheetsClient.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read Changed States sheet: %w", err)
	}

	var records []app.StateRecord
	for _, row := range values {
		if len(row) < 8 { // Minimum required: Timestamp through Status State
			log.Debug().Int("row_length", len(row)).Interface("row", row).Msg("Skipping row with insufficient columns")
			continue
		}

		record, err := s.convertRowToStateRecord(row)
		if err != nil {
			log.Warn().Err(err).Interface("row", row).Int("row_length", len(row)).Msg("Failed to convert row to StateRecord, skipping")
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

// addStateRecords adds multiple state records to the Changed States sheet
func (s *StateTrackingService) addStateRecords(ctx context.Context, spreadsheetID string, records []app.StateRecord) error {
	if len(records) == 0 {
		return nil
	}

	sheetName := "Changed States"
	var rows [][]interface{}

	for _, record := range records {
		row := s.convertStateRecordToRow(record)
		rows = append(rows, row)
	}

	rangeSpec := fmt.Sprintf("%s!A:J", sheetName)
	return s.sheetsClient.AppendRows(ctx, spreadsheetID, rangeSpec, rows)
}

// convertStateRecordToRow converts a StateRecord into spreadsheet row format
func (s *StateTrackingService) convertStateRecordToRow(record app.StateRecord) []interface{} {
	// Format timestamp as human-readable string
	timestampStr := record.Timestamp.UTC().Format("2006-01-02 15:04:05")

	// Handle StatusUntil - only include if it's a meaningful time (not zero time)
	var statusUntilStr string
	if !record.StatusUntil.IsZero() {
		statusUntilStr = record.StatusUntil.UTC().Format("2006-01-02 15:04:05")
	}

	return []interface{}{
		timestampStr, record.MemberID, record.MemberName,
		record.FactionID, record.FactionName, record.LastActionStatus, record.StatusDescription,
		record.StatusState, statusUntilStr, record.StatusTravelType,
	}
}

// convertRowToStateRecord converts a spreadsheet row into a StateRecord using type-safe Cell
func (s *StateTrackingService) convertRowToStateRecord(row []interface{}) (app.StateRecord, error) {
	var record app.StateRecord

	// Parse timestamp (now a string) using type-safe Cell
	timestampStr := sheets.NewCell(row[0]).String()
	if timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
		record.Timestamp = timestamp.UTC()
	}

	// Parse string fields with new column indices using type-safe Cell
	record.MemberID = sheets.NewCell(row[1]).String()
	record.MemberName = sheets.NewCell(row[2]).String()
	record.FactionID = sheets.NewCell(row[3]).String()
	record.FactionName = sheets.NewCell(row[4]).String()
	record.LastActionStatus = sheets.NewCell(row[5]).String()
	record.StatusDescription = sheets.NewCell(row[6]).String()
	record.StatusState = sheets.NewCell(row[7]).String()

	if len(row) > 9 {
		record.StatusTravelType = sheets.NewCell(row[9]).String()
	}

	// Parse StatusUntil - only if not empty
	if len(row) > 8 {
		statusUntilStr := sheets.NewCell(row[8]).String()
		if statusUntilStr != "" {
			if statusUntil, err := time.Parse("2006-01-02 15:04:05", statusUntilStr); err == nil {
				record.StatusUntil = statusUntil.UTC()
			}
		}
	}

	return record, nil
}
