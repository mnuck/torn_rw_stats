package sheets

import (
	"context"
	"fmt"

	"torn_rw_stats/internal/app"

	"log/slog"
)

// WarSheetsManager handles business logic for war sheet management
// Separated from infrastructure concerns for better testability
type WarSheetsManager struct {
	api SheetsAPI
}

// NewWarSheetsManager creates a new war sheets manager with the given API client
func NewWarSheetsManager(api SheetsAPI) *WarSheetsManager {
	return &WarSheetsManager{
		api: api,
	}
}

// EnsureWarSheets creates summary and records sheets for a war if they don't exist
func (m *WarSheetsManager) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error) {
	summaryTabName := m.GenerateSummaryTabName(war.ID)
	recordsTabName := m.GenerateRecordsTabName(war.ID)

	slog.Debug("Ensuring war sheets exist",
		"war_id", war.ID,
		"summary_tab", summaryTabName,
		"records_tab", recordsTabName)

	// Check if summary sheet exists
	summaryExists, err := m.api.SheetExists(ctx, spreadsheetID, summaryTabName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if summary sheet exists: %w", err)
	}

	if !summaryExists {
		slog.Info("Creating summary sheet",
			"sheet_name", summaryTabName)

		if err := m.api.CreateSheet(ctx, spreadsheetID, summaryTabName); err != nil {
			return nil, fmt.Errorf("failed to create summary sheet: %w", err)
		}

		// Initialize summary sheet with headers
		if err := m.InitializeSummarySheet(ctx, spreadsheetID, summaryTabName); err != nil {
			return nil, fmt.Errorf("failed to initialize summary sheet: %w", err)
		}
	}

	// Check if records sheet exists
	recordsExists, err := m.api.SheetExists(ctx, spreadsheetID, recordsTabName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if records sheet exists: %w", err)
	}

	if !recordsExists {
		slog.Info("Creating records sheet",
			"sheet_name", recordsTabName)

		if err := m.api.CreateSheet(ctx, spreadsheetID, recordsTabName); err != nil {
			return nil, fmt.Errorf("failed to create records sheet: %w", err)
		}

		// Initialize records sheet with headers
		if err := m.InitializeRecordsSheet(ctx, spreadsheetID, recordsTabName); err != nil {
			return nil, fmt.Errorf("failed to initialize records sheet: %w", err)
		}
	}

	return &app.SheetConfig{
		WarID:          war.ID,
		SummaryTabName: summaryTabName,
		RecordsTabName: recordsTabName,
		SpreadsheetID:  spreadsheetID,
	}, nil
}

// GenerateSummaryTabName creates a standardized summary tab name for a war
func (m *WarSheetsManager) GenerateSummaryTabName(warID int) string {
	return fmt.Sprintf("Summary - %d", warID)
}

// GenerateRecordsTabName creates a standardized records tab name for a war
func (m *WarSheetsManager) GenerateRecordsTabName(warID int) string {
	return fmt.Sprintf("Records - %d", warID)
}

// InitializeSummarySheet sets up headers and initial content for a summary sheet
func (m *WarSheetsManager) InitializeSummarySheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateSummarySheetHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write summary headers: %w", err)
	}

	slog.Debug("Initialized summary sheet with headers",
		"sheet_name", sheetName,
		"header_rows", len(headers))

	return nil
}

// GenerateSummarySheetHeaders creates the standard headers for war summary sheets
func (m *WarSheetsManager) GenerateSummarySheetHeaders() [][]interface{} {
	return [][]interface{}{
		{"War Summary"},
		{},
		{"War ID", ""},
		{"Status", ""},
		{"Start Time", ""},
		{"End Time", ""},
		{},
		{"Our Faction", ""},
		{"Enemy Faction", ""},
		{},
		{"Current Scores"},
		{"Our Score", ""},
		{"Enemy Score", ""},
		{},
		{"Attack Statistics"},
		{"Total Attacks", ""},
		{"Attacks Won", ""},
		{"Attacks Lost", ""},
		{"Win Rate", ""},
		{},
		{"Respect Statistics"},
		{"Respect Gained", ""},
		{"Respect Lost", ""},
		{"Net Respect", ""},
	}
}

// InitializeRecordsSheet sets up headers and initial content for a records sheet
func (m *WarSheetsManager) InitializeRecordsSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := m.GenerateRecordsSheetHeaders()

	rangeSpec := fmt.Sprintf("%s!A1", sheetName)
	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, headers); err != nil {
		return fmt.Errorf("failed to write records headers: %w", err)
	}

	slog.Debug("Initialized records sheet with headers",
		"sheet_name", sheetName,
		"header_columns", len(headers[0]))

	return nil
}

// GenerateRecordsSheetHeaders creates the standard headers for attack records sheets
func (m *WarSheetsManager) GenerateRecordsSheetHeaders() [][]interface{} {
	return [][]interface{}{
		{
			"Attack ID",
			"Code",
			"Started",
			"Ended",
			"Direction",
			"Attacker ID",
			"Attacker Name",
			"Attacker Level",
			"Attacker Faction ID",
			"Attacker Faction Name",
			"Defender ID",
			"Defender Name",
			"Defender Level",
			"Defender Faction ID",
			"Defender Faction Name",
			"Result",
			"Respect Gain",
			"Respect Loss",
			"Chain",
			"Is Interrupted",
			"Is Stealthed",
			"Is Raid",
			"Is Ranked War",
			"Modifier Fair Fight",
			"Modifier War",
			"Modifier Retaliation",
			"Modifier Group",
			"Modifier Overseas",
			"Modifier Chain",
			"Modifier Warlord",
			"Finishing Hit Name",
			"Finishing Hit Value",
		},
	}
}

// UpdateWarSummary updates the summary sheet with current war statistics
func (m *WarSheetsManager) UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error {
	// Generate summary data rows
	summaryData := m.ConvertSummaryToRows(summary)

	// Update the summary data (starting from row 3, column B to avoid overwriting labels)
	rangeSpec := fmt.Sprintf("%s!B3:B%d", config.SummaryTabName, 2+len(summaryData))

	// Convert to the format expected by UpdateRange
	values := make([][]interface{}, len(summaryData))
	for i, row := range summaryData {
		values[i] = []interface{}{row}
	}

	if err := m.api.UpdateRange(ctx, spreadsheetID, rangeSpec, values); err != nil {
		return fmt.Errorf("failed to update war summary: %w", err)
	}

	slog.Debug("Updated war summary sheet",
		"war_id", config.WarID,
		"sheet_name", config.SummaryTabName,
		"data_rows", len(summaryData))

	return nil
}

// ConvertSummaryToRows converts a WarSummary into spreadsheet row format
func (m *WarSheetsManager) ConvertSummaryToRows(summary *app.WarSummary) []interface{} {
	endTimeStr := "Ongoing"
	if summary.EndTime != nil {
		endTimeStr = summary.EndTime.UTC().Format("2006-01-02 15:04:05")
	}

	winRate := 0.0
	if summary.TotalAttacks > 0 {
		winRate = float64(summary.AttacksWon) / float64(summary.TotalAttacks) * 100
	}

	return []interface{}{
		summary.WarID,  // War ID
		summary.Status, // Status
		summary.StartTime.UTC().Format("2006-01-02 15:04:05"), // Start Time
		endTimeStr,                     // End Time
		"",                             // Empty row
		summary.OurFaction.Name,        // Our Faction Name
		summary.EnemyFaction.Name,      // Enemy Faction Name
		"",                             // Empty row
		"",                             // Current Scores header
		summary.OurFaction.Score,       // Our Score
		summary.EnemyFaction.Score,     // Enemy Score
		"",                             // Empty row
		"",                             // Attack Statistics header
		summary.TotalAttacks,           // Total Attacks
		summary.AttacksWon,             // Attacks Won
		summary.AttacksLost,            // Attacks Lost
		fmt.Sprintf("%.1f%%", winRate), // Win Rate
		"",                             // Empty row
		"",                             // Respect Statistics header
		summary.RespectGained,          // Respect Gained
		summary.RespectLost,            // Respect Lost
		summary.RespectGained - summary.RespectLost, // Net Respect
	}
}
