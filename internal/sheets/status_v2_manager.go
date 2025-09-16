package sheets

import (
	"context"
	"fmt"
	"sort"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// StatusV2Manager handles Status v2 sheets for faction monitoring
type StatusV2Manager struct {
	api SheetsAPI
}

// NewStatusV2Manager creates a new Status v2 manager
func NewStatusV2Manager(api SheetsAPI) *StatusV2Manager {
	return &StatusV2Manager{
		api: api,
	}
}

// EnsureStatusV2Sheet creates a Status v2 sheet for a faction if it doesn't exist
func (m *StatusV2Manager) EnsureStatusV2Sheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := m.GenerateStatusV2SheetName(factionID)

	// Check if sheet exists
	exists, err := m.api.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if Status v2 sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Int("faction_id", factionID).
			Msg("Creating Status v2 sheet")

		if err := m.api.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create Status v2 sheet: %w", err)
		}

		// Initialize with headers
		if err := m.InitializeStatusV2Sheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize Status v2 sheet: %w", err)
		}
	}

	return sheetName, nil
}

// GenerateStatusV2SheetName creates a standardized Status v2 sheet name for a faction
func (m *StatusV2Manager) GenerateStatusV2SheetName(factionID int) string {
	return fmt.Sprintf("Status v2 - %d", factionID)
}

// InitializeStatusV2Sheet sets up headers for a Status v2 sheet
func (m *StatusV2Manager) InitializeStatusV2Sheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateStatusV2Headers()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write Status v2 headers: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Initialized Status v2 sheet with headers")

	return nil
}

// GenerateStatusV2Headers creates the headers for Status v2 sheets
// Same as Status but with "State" column added between Level and Status
func (m *StatusV2Manager) GenerateStatusV2Headers() [][]interface{} {
	return [][]interface{}{
		{
			"Player Name",
			"Level",
			"State",  // NEW: LastActionStatus from StateRecord
			"Status", // Status Description
			"Location",
			"Countdown",
			"Departure",
			"Arrival",
		},
	}
}

// UpdateStatusV2 updates the Status v2 sheet with current state record data
func (m *StatusV2Manager) UpdateStatusV2(ctx context.Context, spreadsheetID, sheetName string, records []app.StatusV2Record) error {
	if len(records) == 0 {
		log.Debug().
			Str("sheet_name", sheetName).
			Msg("No Status v2 records to update")
		return nil
	}

	// Sort records by Level (descending - highest level first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Level > records[j].Level
	})

	// Convert records to spreadsheet format
	rows := m.ConvertStatusV2RecordsToRows(records)

	// Clear existing content (except headers) and write new data
	rangeSpec := fmt.Sprintf("%s!A2:H", sheetName)
	if err := m.api.ClearRange(ctx, spreadsheetID, rangeSpec); err != nil {
		return fmt.Errorf("failed to clear Status v2 data: %w", err)
	}

	// Ensure sheet has enough capacity
	requiredRows := len(rows) + 1 // +1 for header
	requiredCols := 8
	if err := m.api.EnsureSheetCapacity(ctx, spreadsheetID, sheetName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Write the data starting from row 2
	dataRangeSpec := fmt.Sprintf("%s!A2", sheetName)
	if err := m.api.AppendRows(ctx, spreadsheetID, dataRangeSpec, rows); err != nil {
		return fmt.Errorf("failed to update Status v2 records: %w", err)
	}

	// Apply formatting after data is added
	if err := m.api.FormatStatusSheet(ctx, spreadsheetID, sheetName); err != nil {
		// Log error but don't fail - formatting is nice-to-have
		log.Warn().
			Err(err).
			Str("sheet_name", sheetName).
			Msg("Failed to apply formatting to Status v2 sheet")
	}

	log.Info().
		Str("sheet_name", sheetName).
		Int("records_updated", len(records)).
		Msg("Updated Status v2 sheet")

	return nil
}

// ConvertStatusV2RecordsToRows converts Status v2 records into spreadsheet row format
func (m *StatusV2Manager) ConvertStatusV2RecordsToRows(records []app.StatusV2Record) [][]interface{} {
	rows := make([][]interface{}, len(records))

	for i, record := range records {
		rows[i] = []interface{}{
			record.Name,      // Player Name
			record.Level,     // Level
			record.State,     // State (LastActionStatus)
			record.Status,    // Status (Status Description)
			record.Location,  // Location
			record.Countdown, // Countdown (calculated from StatusUntil)
			record.Departure, // Departure time (manual adjustment preserved)
			record.Arrival,   // Arrival time (manual adjustment preserved)
		}
	}

	return rows
}
