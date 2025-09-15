package sheets

import (
	"context"
	"fmt"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// TravelStatusManager handles business logic for travel status tracking
// Separated from infrastructure concerns for better testability
type TravelStatusManager struct {
	api SheetsAPI
}

// NewTravelStatusManager creates a new travel status manager with the given API client
func NewTravelStatusManager(api SheetsAPI) *TravelStatusManager {
	return &TravelStatusManager{
		api: api,
	}
}

// EnsureTravelStatusSheet creates a status sheet for a faction if it doesn't exist
func (m *TravelStatusManager) EnsureTravelStatusSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := m.GenerateTravelSheetName(factionID)

	// Check if sheet exists
	exists, err := m.api.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if travel sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Int("faction_id", factionID).
			Msg("Creating status sheet")

		if err := m.api.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create travel sheet: %w", err)
		}

		// Initialize with headers
		if err := m.InitializeTravelStatusSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize status sheet: %w", err)
		}

		// Note: Formatting will be applied after data is added to prevent inheritance issues
	}

	return sheetName, nil
}

// GenerateTravelSheetName creates a standardized status sheet name for a faction
func (m *TravelStatusManager) GenerateTravelSheetName(factionID int) string {
	return fmt.Sprintf("Status - %d", factionID)
}

// InitializeTravelStatusSheet sets up headers for a travel status sheet
func (m *TravelStatusManager) InitializeTravelStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateTravelStatusHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write travel headers: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Initialized status sheet with headers")

	return nil
}

// GenerateTravelStatusHeaders creates the standard headers for travel status sheets
func (m *TravelStatusManager) GenerateTravelStatusHeaders() [][]interface{} {
	return [][]interface{}{
		{
			"Player Name",
			"Level",
			"Status",
			"Location",
			"Countdown",
			"Departure",
			"Arrival",
		},
	}
}

// UpdateTravelStatus updates the travel status sheet with current member data
func (m *TravelStatusManager) UpdateTravelStatus(ctx context.Context, spreadsheetID, sheetName string, records []app.TravelRecord) error {
	if len(records) == 0 {
		log.Debug().
			Str("sheet_name", sheetName).
			Msg("No travel records to update")
		return nil
	}

	// Convert records to spreadsheet format
	rows := m.ConvertTravelRecordsToRows(records)

	// Clear existing content (except headers) and write new data
	rangeSpec := fmt.Sprintf("%s!A2:G", sheetName)
	if err := m.api.ClearRange(ctx, spreadsheetID, rangeSpec); err != nil {
		return fmt.Errorf("failed to clear travel data: %w", err)
	}

	// Ensure sheet has enough capacity
	requiredRows := len(rows) + 1 // +1 for header
	requiredCols := 7
	if err := m.api.EnsureSheetCapacity(ctx, spreadsheetID, sheetName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Write the data starting from row 2
	dataRangeSpec := fmt.Sprintf("%s!A2", sheetName)
	if err := m.api.AppendRows(ctx, spreadsheetID, dataRangeSpec, rows); err != nil {
		return fmt.Errorf("failed to update travel records: %w", err)
	}

	// Apply formatting after data is added to prevent inheritance issues
	if err := m.api.FormatStatusSheet(ctx, spreadsheetID, sheetName); err != nil {
		// Log error but don't fail - formatting is nice-to-have, not critical
		log.Warn().
			Err(err).
			Str("sheet_name", sheetName).
			Msg("Failed to apply formatting to status sheet")
	}

	log.Info().
		Str("sheet_name", sheetName).
		Int("records_updated", len(records)).
		Msg("Updated status sheet")

	return nil
}

// ConvertTravelRecordsToRows converts travel records into spreadsheet row format
func (m *TravelStatusManager) ConvertTravelRecordsToRows(records []app.TravelRecord) [][]interface{} {
	rows := make([][]interface{}, len(records))

	for i, record := range records {
		rows[i] = []interface{}{
			record.Name,      // Player Name
			record.Level,     // Level
			record.State,     // Status
			record.Location,  // Location
			record.Countdown, // Countdown (travel time left or hospital time)
			record.Departure, // Departure time
			record.Arrival,   // Arrival time
		}
	}

	return rows
}
