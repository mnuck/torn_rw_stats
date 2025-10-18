package sheets

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// ChangedStatesSheetManager handles operations on the Changed States sheet, which
// tracks member state changes over time (status, location, travel, etc.).
type ChangedStatesSheetManager struct {
	api SheetsAPI
}

// NewChangedStatesSheetManager creates a new Changed States sheet manager
func NewChangedStatesSheetManager(api SheetsAPI) *ChangedStatesSheetManager {
	return &ChangedStatesSheetManager{
		api: api,
	}
}

// EnsureChangedStatesSheet creates the Changed States sheet if it doesn't exist
func (m *ChangedStatesSheetManager) EnsureChangedStatesSheet(ctx context.Context, spreadsheetID string) error {
	sheetName := "Changed States"

	// Check if sheet exists
	exists, err := m.api.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return fmt.Errorf("failed to check if Changed States sheet exists: %w", err)
	}

	if !exists {
		// Create the sheet
		if err := m.api.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return fmt.Errorf("failed to create Changed States sheet: %w", err)
		}

		// Initialize with headers
		if err := m.InitializeChangedStatesSheet(ctx, spreadsheetID, sheetName); err != nil {
			return fmt.Errorf("failed to initialize Changed States sheet: %w", err)
		}

		log.Info().
			Str("sheet_name", sheetName).
			Msg("Created and initialized Changed States sheet")
	}

	return nil
}

// InitializeChangedStatesSheet sets up headers for the Changed States sheet
func (m *ChangedStatesSheetManager) InitializeChangedStatesSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateChangedStatesHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write Changed States headers: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Initialized Changed States sheet with headers")

	return nil
}

// GenerateChangedStatesHeaders creates the standard headers for the Changed States sheet
func (m *ChangedStatesSheetManager) GenerateChangedStatesHeaders() [][]interface{} {
	return [][]interface{}{
		{
			"Timestamp",
			"Date",
			"Time",
			"Member ID",
			"Member Name",
			"Faction ID",
			"Faction Name",
			"Last Action Status",
			"Status Description",
			"Status State",
			"Status Until",
			"Status Travel Type",
		},
	}
}

// ReadChangedStatesSheet reads all records from the Changed States sheet
func (m *ChangedStatesSheetManager) ReadChangedStatesSheet(ctx context.Context, spreadsheetID string) ([]app.StateRecord, error) {
	sheetName := "Changed States"
	rangeSpec := fmt.Sprintf("%s!A2:L", sheetName) // Skip header row

	values, err := m.api.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read Changed States sheet: %w", err)
	}

	var records []app.StateRecord
	for _, row := range values {
		if len(row) < 12 {
			continue // Skip incomplete rows
		}

		record, err := m.ConvertRowToStateRecord(row)
		if err != nil {
			log.Warn().
				Err(err).
				Interface("row", row).
				Msg("Failed to convert row to StateRecord, skipping")
			continue
		}

		records = append(records, record)
	}

	log.Debug().
		Int("records_count", len(records)).
		Msg("Read records from Changed States sheet")

	return records, nil
}

// AddStateRecords adds multiple state records to the Changed States sheet in a single batch
func (m *ChangedStatesSheetManager) AddStateRecords(ctx context.Context, spreadsheetID string, records []app.StateRecord) error {
	if len(records) == 0 {
		return nil
	}

	sheetName := "Changed States"

	// Convert records to rows
	var rows [][]interface{}
	for _, record := range records {
		row := m.ConvertStateRecordToRow(record)
		rows = append(rows, row)
	}

	// Append to sheet
	rangeSpec := fmt.Sprintf("%s!A:L", sheetName)
	if err := m.api.AppendRows(ctx, spreadsheetID, rangeSpec, rows); err != nil {
		return fmt.Errorf("failed to append state records to Changed States sheet: %w", err)
	}

	log.Info().
		Int("records_added", len(records)).
		Str("sheet_name", sheetName).
		Msg("Added state records to Changed States sheet")

	return nil
}

// ConvertStateRecordToRow converts a StateRecord into spreadsheet row format
func (m *ChangedStatesSheetManager) ConvertStateRecordToRow(record app.StateRecord) []interface{} {
	// Format timestamp
	timestamp := record.Timestamp.UTC()
	dateStr := timestamp.Format("2006-01-02")
	timeStr := timestamp.Format("15:04:05")

	// Format StatusUntil
	var statusUntilStr string
	if !record.StatusUntil.IsZero() {
		statusUntilStr = record.StatusUntil.UTC().Format("2006-01-02 15:04:05")
	}

	return []interface{}{
		record.Timestamp.Unix(),  // Timestamp (for sorting)
		dateStr,                  // Date
		timeStr,                  // Time
		record.MemberID,          // Member ID
		record.MemberName,        // Member Name
		record.FactionID,         // Faction ID
		record.FactionName,       // Faction Name
		record.LastActionStatus,  // Last Action Status
		record.StatusDescription, // Status Description
		record.StatusState,       // Status State
		statusUntilStr,           // Status Until
		record.StatusTravelType,  // Status Travel Type
	}
}

// ConvertRowToStateRecord converts a spreadsheet row into a StateRecord
func (m *ChangedStatesSheetManager) ConvertRowToStateRecord(row []interface{}) (app.StateRecord, error) {
	var record app.StateRecord

	// Parse timestamp
	timestampStr := NewCell(row[0]).String()
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		record.Timestamp = time.Unix(timestamp, 0).UTC()
	}

	// Parse string fields using type-safe Cell accessors
	record.MemberID = NewCell(row[3]).String()
	record.MemberName = NewCell(row[4]).String()
	record.FactionID = NewCell(row[5]).String()
	record.FactionName = NewCell(row[6]).String()
	record.LastActionStatus = NewCell(row[7]).String()
	record.StatusDescription = NewCell(row[8]).String()
	record.StatusState = NewCell(row[9]).String()
	record.StatusTravelType = NewCell(row[11]).String()

	// Parse StatusUntil
	statusUntilStr := NewCell(row[10]).String()
	if statusUntilStr != "" {
		if statusUntil, err := time.Parse("2006-01-02 15:04:05", statusUntilStr); err == nil {
			record.StatusUntil = statusUntil.UTC()
		}
	}

	return record, nil
}
