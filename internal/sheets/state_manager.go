package sheets

import (
	"context"
	"fmt"
	"time"

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
		Str("old_state", record.OldState).
		Str("new_state", record.NewState).
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
		record.Timestamp.Unix(), // Timestamp (for sorting)
		dateStr,                 // Date
		timeStr,                 // Time
		record.MemberID,         // Player ID
		record.MemberName,       // Player Name
		"State Change",          // Change Type
		record.OldState,         // Old Status
		record.NewState,         // New Status
		record.StatusDescription, // Description
	}
}

// EnsurePreviousStateSheet creates a previous state tracking sheet for a faction if it doesn't exist
func (m *StateChangeManager) EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := m.GeneratePreviousStateSheetName(factionID)

	// Check if sheet exists
	exists, err := m.api.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if previous state sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Int("faction_id", factionID).
			Msg("Creating previous state sheet")

		if err := m.api.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create previous state sheet: %w", err)
		}

		// Initialize with headers
		if err := m.InitializePreviousStateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize previous state sheet: %w", err)
		}
	}

	return sheetName, nil
}

// GeneratePreviousStateSheetName creates a standardized previous state sheet name for a faction
func (m *StateChangeManager) GeneratePreviousStateSheetName(factionID int) string {
	return fmt.Sprintf("Previous States - %d", factionID)
}

// InitializePreviousStateSheet sets up headers for a previous state sheet
func (m *StateChangeManager) InitializePreviousStateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GeneratePreviousStateHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write previous state headers: %w", err)
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Initialized previous state sheet with headers")

	return nil
}

// GeneratePreviousStateHeaders creates the standard headers for previous state sheets
func (m *StateChangeManager) GeneratePreviousStateHeaders() [][]interface{} {
	return [][]interface{}{
		{
			"Player ID",
			"Player Name",
			"Level",
			"Status",
			"Last Action",
			"Until",
			"Description",
			"Location",
			"Last Updated",
		},
	}
}

// StorePreviousMemberStates stores the current member states for future comparison
func (m *StateChangeManager) StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error {
	if len(members) == 0 {
		log.Debug().
			Str("sheet_name", sheetName).
			Msg("No member states to store")
		return nil
	}

	// Convert to spreadsheet format
	rows := m.ConvertMembersToStateRows(members)

	// Clear existing content (except headers) and write new data
	rangeSpec := fmt.Sprintf("%s!A2:I", sheetName)
	if err := m.api.ClearRange(ctx, spreadsheetID, rangeSpec); err != nil {
		return fmt.Errorf("failed to clear previous state data: %w", err)
	}

	// Ensure sheet has enough capacity
	requiredRows := len(rows) + 1 // +1 for header
	requiredCols := 9
	if err := m.api.EnsureSheetCapacity(ctx, spreadsheetID, sheetName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Write the data starting from row 2
	dataRangeSpec := fmt.Sprintf("%s!A2", sheetName)
	if err := m.api.AppendRows(ctx, spreadsheetID, dataRangeSpec, rows); err != nil {
		return fmt.Errorf("failed to store previous member states: %w", err)
	}

	log.Info().
		Str("sheet_name", sheetName).
		Int("members_stored", len(members)).
		Msg("Stored previous member states")

	return nil
}

// LoadPreviousMemberStates loads the previously stored member states for comparison
func (m *StateChangeManager) LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error) {
	// Read all data (skip header row)
	rangeSpec := fmt.Sprintf("%s!A2:I", sheetName)

	values, err := m.api.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read previous member states: %w", err)
	}

	members := make(map[string]app.FactionMember)

	for _, row := range values {
		if len(row) < 8 { // Need at least 8 columns
			continue
		}

		// Parse member data using the existing parse functions
		playerName := parseStringValue(row[1])
		if playerName == "" {
			continue
		}

		level := parseIntValue(row[2])
		location := parseStringValue(row[7])

		member := app.FactionMember{
			Name:     playerName,
			Level:    level,
			Position: location,
			// Note: FactionMember has different structure than what's stored
			// LastAction and Status are complex types, not simple fields
		}

		// Use player name as key for lookup
		members[playerName] = member
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int("members_loaded", len(members)).
		Msg("Loaded previous member states")

	return members, nil
}

// ConvertMembersToStateRows converts faction members into spreadsheet row format
func (m *StateChangeManager) ConvertMembersToStateRows(members map[string]app.FactionMember) [][]interface{} {
	rows := make([][]interface{}, 0, len(members))

	for _, member := range members {
		// Format timestamps from actual structure
		lastActionTimestamp := member.LastAction.Timestamp
		var untilTimestamp interface{}
		if member.Status.Until != nil {
			untilTimestamp = *member.Status.Until
		}

		currentTime := time.Now().Format("2006-01-02 15:04:05")

		row := []interface{}{
			0,                          // Player ID (not available in FactionMember)
			member.Name,                // Player Name
			member.Level,               // Level
			member.Status.Description,  // Status
			lastActionTimestamp,        // Last Action (timestamp)
			untilTimestamp,             // Until (timestamp)
			member.Status.Details,      // Description
			member.Position,            // Location/Position
			currentTime,                // Last Updated
		}

		rows = append(rows, row)
	}

	return rows
}