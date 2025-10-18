package sheets

import (
	"context"
	"fmt"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// StateChangeManager handles business logic for state change tracking
// Separated from infrastructure concerns for better testability
type StateChangeManager struct {
	api SheetsAPI
}

// NewStateChangeManager creates a new state change manager with the given API client
func NewStateChangeManager(api SheetsAPI) *StateChangeManager {
	return &StateChangeManager{
		api: api,
	}
}

// EnsureStateChangeRecordsSheet creates a state change records sheet for a faction if it doesn't exist
func (m *StateChangeManager) EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := m.GenerateStateChangeSheetName(factionID)

	// Check if sheet exists
	exists, err := m.api.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if state change sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Int("faction_id", factionID).
			Msg("Creating state change records sheet")

		if err := m.api.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create state change sheet: %w", err)
		}

		// Initialize with headers
		if err := m.InitializeStateChangeRecordsSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize state change sheet: %w", err)
		}
	}

	return sheetName, nil
}

// GenerateStateChangeSheetName creates a standardized state change sheet name for a faction
func (m *StateChangeManager) GenerateStateChangeSheetName(factionID int) string {
	return fmt.Sprintf("State Changes - %d", factionID)
}

// InitializeStateChangeRecordsSheet sets up headers for a state change records sheet
func (m *StateChangeManager) InitializeStateChangeRecordsSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateStateChangeHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write state change headers: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Initialized state change records sheet with headers")

	return nil
}

// GenerateStateChangeHeaders creates the standard headers for state change sheets
func (m *StateChangeManager) GenerateStateChangeHeaders() [][]interface{} {
	return [][]interface{}{
		{
			"Timestamp",
			"Date",
			"Time",
			"Player ID",
			"Player Name",
			"Change Type",
			"Old Status",
			"New Status",
			"Description",
		},
	}
}

// AddStateChangeRecord adds a single state change record to the sheet
func (m *StateChangeManager) AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error {
	// Convert record to spreadsheet format
	row := m.ConvertStateChangeToRow(record)

	// Append the record
	rangeSpec := fmt.Sprintf("%s!A:I", sheetName)
	if err := m.api.AppendRows(ctx, spreadsheetID, rangeSpec, [][]interface{}{row}); err != nil {
		return fmt.Errorf("failed to add state change record: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int("member_id", record.MemberID).
		Str("old_state", record.PreviousState).
		Str("new_state", record.CurrentState).
		Msg("Added state change record")

	return nil
}

// ConvertStateChangeToRow converts a state change record into spreadsheet row format
func (m *StateChangeManager) ConvertStateChangeToRow(record app.StateChangeRecord) []interface{} {
	// Format timestamp
	timestamp := record.Timestamp.UTC()
	dateStr := timestamp.Format("2006-01-02")
	timeStr := timestamp.Format("15:04:05")

	return []interface{}{
		record.Timestamp.Unix(),  // Timestamp (for sorting)
		dateStr,                  // Date
		timeStr,                  // Time
		record.MemberID,          // Player ID
		record.MemberName,        // Player Name
		"State Change",           // Change Type
		record.PreviousState,     // Old Status
		record.CurrentState,      // New Status
		record.StatusDescription, // Description
	}
}
