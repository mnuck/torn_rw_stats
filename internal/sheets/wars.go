package sheets

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// EnsureWarSheets creates summary and records sheets for a war if they don't exist
func (c *Client) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error) {
	summaryTabName := fmt.Sprintf("Summary - %d", war.ID)
	recordsTabName := fmt.Sprintf("Records - %d", war.ID)

	log.Debug().
		Int("war_id", war.ID).
		Str("summary_tab", summaryTabName).
		Str("records_tab", recordsTabName).
		Msg("Ensuring war sheets exist")

	// Check if summary sheet exists
	summaryExists, err := c.SheetExists(ctx, spreadsheetID, summaryTabName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if summary sheet exists: %w", err)
	}

	if !summaryExists {
		log.Info().
			Str("sheet_name", summaryTabName).
			Msg("Creating summary sheet")

		if err := c.CreateSheet(ctx, spreadsheetID, summaryTabName); err != nil {
			return nil, fmt.Errorf("failed to create summary sheet: %w", err)
		}

		// Initialize summary sheet with headers
		if err := c.initializeSummarySheet(ctx, spreadsheetID, summaryTabName); err != nil {
			return nil, fmt.Errorf("failed to initialize summary sheet: %w", err)
		}
	}

	// Check if records sheet exists
	recordsExists, err := c.SheetExists(ctx, spreadsheetID, recordsTabName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if records sheet exists: %w", err)
	}

	if !recordsExists {
		log.Info().
			Str("sheet_name", recordsTabName).
			Msg("Creating records sheet")

		if err := c.CreateSheet(ctx, spreadsheetID, recordsTabName); err != nil {
			return nil, fmt.Errorf("failed to create records sheet: %w", err)
		}

		// Initialize records sheet with headers
		if err := c.initializeRecordsSheet(ctx, spreadsheetID, recordsTabName); err != nil {
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

// initializeSummarySheet sets up headers and initial content for a summary sheet
func (c *Client) initializeSummarySheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := [][]interface{}{
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
		{},
		{"Last Updated", ""},
	}

	range_ := fmt.Sprintf("'%s'!A1:B%d", sheetName, len(headers))
	return c.UpdateRange(ctx, spreadsheetID, range_, headers)
}

// initializeRecordsSheet sets up headers for a records sheet
func (c *Client) initializeRecordsSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := [][]interface{}{
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

	range_ := fmt.Sprintf("'%s'!A1:AF1", sheetName)
	return c.UpdateRange(ctx, spreadsheetID, range_, headers)
}

// UpdateWarSummary updates the summary sheet with current war statistics
func (c *Client) UpdateWarSummary(ctx context.Context, spreadsheetID string, config *app.SheetConfig, summary *app.WarSummary) error {
	log.Debug().
		Int("war_id", summary.WarID).
		Str("sheet_name", config.SummaryTabName).
		Msg("Updating war summary sheet")

	endTimeStr := "Active"
	if summary.EndTime != nil {
		endTimeStr = summary.EndTime.Format("2006-01-02 15:04:05")
	}

	winRate := float64(0)
	if summary.TotalAttacks > 0 {
		winRate = float64(summary.AttacksWon) / float64(summary.TotalAttacks) * 100
	}

	netRespect := summary.RespectGained - summary.RespectLost

	// Update the summary data (column B)
	values := [][]interface{}{
		{summary.WarID},
		{summary.Status},
		{summary.StartTime.Format("2006-01-02 15:04:05")},
		{endTimeStr},
		{},
		{summary.OurFaction.Name},
		{summary.EnemyFaction.Name},
		{},
		{},
		{summary.OurFaction.Score},
		{summary.EnemyFaction.Score},
		{},
		{},
		{summary.TotalAttacks},
		{summary.AttacksWon},
		{summary.AttacksLost},
		{fmt.Sprintf("%.1f%%", winRate)},
		{},
		{},
		{fmt.Sprintf("%.2f", summary.RespectGained)},
		{fmt.Sprintf("%.2f", summary.RespectLost)},
		{fmt.Sprintf("%.2f", netRespect)},
		{},
		{summary.LastUpdated.Format("2006-01-02 15:04:05")},
	}

	range_ := fmt.Sprintf("'%s'!B3:B%d", config.SummaryTabName, len(values)+2)
	return c.UpdateRange(ctx, spreadsheetID, range_, values)
}

// ExistingRecordsInfo contains information about existing attack records in the sheet
type ExistingRecordsInfo struct {
	AttackCodes     map[string]bool
	LatestTimestamp int64
	RecordCount     int
}

// ReadExistingRecords analyzes existing attack records in the sheet
func (c *Client) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*ExistingRecordsInfo, error) {
	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Reading existing attack records")

	// Read all data from the sheet (starting from row 2 to skip headers)
	range_ := fmt.Sprintf("'%s'!A2:AF", sheetName)
	values, err := c.ReadSheet(ctx, spreadsheetID, range_)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing records: %w", err)
	}

	info := &ExistingRecordsInfo{
		AttackCodes:     make(map[string]bool),
		LatestTimestamp: 0,
		RecordCount:     len(values),
	}

	validRows := 0
	for _, row := range values {
		if len(row) < 3 { // Need at least Code and Started timestamp
			continue
		}

		// Parse Attack Code (column B) - always a string
		if codeStr, ok := row[1].(string); ok && codeStr != "" {
			info.AttackCodes[codeStr] = true
			validRows++
		}

		// Parse Started timestamp (column C) to find latest
		if startedStr, ok := row[2].(string); ok {
			if startedTime, err := time.Parse("2006-01-02 15:04:05", startedStr); err == nil {
				timestamp := startedTime.Unix()
				if timestamp > info.LatestTimestamp {
					info.LatestTimestamp = timestamp
				}
			}
		}
	}

	// Update record count to reflect valid rows only
	info.RecordCount = validRows

	log.Debug().
		Int("total_rows_read", len(values)).
		Int("valid_records", info.RecordCount).
		Int("unique_attack_codes", len(info.AttackCodes)).
		Int64("latest_timestamp", info.LatestTimestamp).
		Str("latest_time", time.Unix(info.LatestTimestamp, 0).Format("2006-01-02 15:04:05")).
		Msg("Analyzed existing records")

	// Validation: warn if no attack codes were parsed from non-empty sheet
	if len(values) > 0 && len(info.AttackCodes) == 0 {
		log.Warn().
			Int("rows_in_sheet", len(values)).
			Msg("No attack codes parsed from existing sheet - possible column mismatch")
	}

	return info, nil
}

// UpdateAttackRecords updates the records sheet with new attack data using append strategy
func (c *Client) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error {
	if len(records) == 0 {
		return nil
	}

	log.Info().
		Int("war_id", config.WarID).
		Str("sheet_name", config.RecordsTabName).
		Int("records_count", len(records)).
		Msg("=== ENTERING UpdateAttackRecords ===")

	// Read existing records to determine update strategy
	existingInfo, err := c.ReadExistingRecords(ctx, spreadsheetID, config.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Filter out duplicate attacks and sort chronologically
	log.Debug().
		Int("input_records", len(records)).
		Int("existing_attack_codes", len(existingInfo.AttackCodes)).
		Int("existing_record_count", existingInfo.RecordCount).
		Msg("Starting deduplication")

	newRecords := c.filterAndSortRecords(records, existingInfo)

	if len(newRecords) == 0 {
		log.Info().Msg("=== EXITING UpdateAttackRecords - No new records after deduplication ===")
		return nil
	}

	log.Info().
		Int("original_records", len(records)).
		Int("new_records", len(newRecords)).
		Int("existing_records", existingInfo.RecordCount).
		Msg("Processed records for update")

	// Convert new records to sheet rows
	rows := c.convertRecordsToRows(newRecords)

	// Calculate required sheet dimensions
	startRow := existingInfo.RecordCount + 2 // +2 for header row and 1-based indexing
	endRow := startRow + len(rows) - 1
	requiredRows := endRow
	requiredCols := 32 // AF column = 32

	// Ensure sheet has sufficient capacity
	if err := c.EnsureSheetCapacity(ctx, spreadsheetID, config.RecordsTabName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Append new rows to the sheet
	range_ := fmt.Sprintf("'%s'!A%d:AF%d", config.RecordsTabName, startRow, endRow)

	// Log first few rows being written to detect duplicates at write time
	sampleRows := make([]string, 0, 3)
	for i, row := range rows {
		if i < 3 && len(row) >= 2 {
			if attackID, ok := row[0].(interface{}); ok {
				if code, ok := row[1].(string); ok {
					sampleRows = append(sampleRows, fmt.Sprintf("ID:%v Code:%s", attackID, code))
				}
			}
		}
	}

	log.Info().
		Str("range", range_).
		Int("rows_to_write", len(rows)).
		Strs("sample_rows", sampleRows).
		Msg("=== WRITING TO SHEET ===")

	err = c.UpdateRange(ctx, spreadsheetID, range_, rows)
	if err != nil {
		return fmt.Errorf("failed to append attack records: %w", err)
	}

	log.Info().
		Int("war_id", config.WarID).
		Int("records_appended", len(newRecords)).
		Str("range", range_).
		Msg("=== EXITING UpdateAttackRecords - Successfully appended records ===")

	return nil
}

// filterAndSortRecords removes duplicates and sorts records chronologically
func (c *Client) filterAndSortRecords(records []app.AttackRecord, existing *ExistingRecordsInfo) []app.AttackRecord {
	var newRecords []app.AttackRecord

	// Filter out duplicates using attack codes (guaranteed unique strings)
	duplicates := 0
	for _, record := range records {
		if !existing.AttackCodes[record.Code] {
			newRecords = append(newRecords, record)
		} else {
			duplicates++
			log.Debug().
				Str("attack_code", record.Code).
				Int64("attack_id", record.AttackID).
				Msg("Filtered duplicate attack")
		}
	}

	// Log some example attack codes for debugging
	sampleCodes := make([]string, 0, 3)
	for i, record := range newRecords {
		if i < 3 {
			sampleCodes = append(sampleCodes, record.Code)
		} else {
			break
		}
	}

	log.Info().
		Int("input_records", len(records)).
		Int("duplicates_filtered", duplicates).
		Int("new_records", len(newRecords)).
		Strs("sample_attack_codes", sampleCodes).
		Msg("Completed deduplication filtering")

	// Sort chronologically (oldest first)
	sort.Slice(newRecords, func(i, j int) bool {
		return newRecords[i].Started.Before(newRecords[j].Started)
	})

	log.Debug().
		Int("filtered_records", len(newRecords)).
		Msg("Filtered and sorted records chronologically")

	return newRecords
}

// convertRecordsToRows converts AttackRecord structs to sheet row format
func (c *Client) convertRecordsToRows(records []app.AttackRecord) [][]interface{} {
	var rows [][]interface{}

	for _, record := range records {
		// Helper function to safely convert nullable int pointers
		factionID := func(id *int) interface{} {
			if id == nil {
				return ""
			}
			return *id
		}

		row := []interface{}{
			record.AttackID,
			record.Code,
			record.Started.Format("2006-01-02 15:04:05"),
			record.Ended.Format("2006-01-02 15:04:05"),
			record.Direction,
			record.AttackerID,
			record.AttackerName,
			record.AttackerLevel,
			factionID(record.AttackerFactionID),
			record.AttackerFactionName,
			record.DefenderID,
			record.DefenderName,
			record.DefenderLevel,
			factionID(record.DefenderFactionID),
			record.DefenderFactionName,
			record.Result,
			fmt.Sprintf("%.2f", record.RespectGain),
			fmt.Sprintf("%.2f", record.RespectLoss),
			record.Chain,
			record.IsInterrupted,
			record.IsStealthed,
			record.IsRaid,
			record.IsRankedWar,
			record.ModifierFairFight,
			record.ModifierWar,
			record.ModifierRetaliation,
			record.ModifierGroup,
			record.ModifierOverseas,
			record.ModifierChain,
			record.ModifierWarlord,
			record.FinishingHitName,
			record.FinishingHitValue,
		}
		rows = append(rows, row)
	}

	return rows
}

// EnsureTravelStatusSheet creates travel status sheet for a faction if it doesn't exist
func (c *Client) EnsureTravelStatusSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	travelTabName := fmt.Sprintf("Travel Status - %d", factionID)

	log.Debug().
		Int("faction_id", factionID).
		Str("travel_tab", travelTabName).
		Msg("Ensuring travel status sheet exists")

	// Check if travel status sheet exists
	travelExists, err := c.SheetExists(ctx, spreadsheetID, travelTabName)
	if err != nil {
		return "", fmt.Errorf("failed to check if travel status sheet exists: %w", err)
	}

	if !travelExists {
		log.Info().
			Str("sheet_name", travelTabName).
			Msg("Creating travel status sheet")

		if err := c.CreateSheet(ctx, spreadsheetID, travelTabName); err != nil {
			return "", fmt.Errorf("failed to create travel status sheet: %w", err)
		}

		// Initialize travel status sheet with headers
		if err := c.initializeTravelStatusSheet(ctx, spreadsheetID, travelTabName); err != nil {
			return "", fmt.Errorf("failed to initialize travel status sheet: %w", err)
		}
	}

	return travelTabName, nil
}

// initializeTravelStatusSheet sets up headers for a travel status sheet
func (c *Client) initializeTravelStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := [][]interface{}{
		{
			"Name",
			"Level",
			"Location",
			"State",
			"Departure",
			"Arrival",
			"Countdown",
		},
	}

	range_ := fmt.Sprintf("'%s'!A1:G1", sheetName)
	return c.UpdateRange(ctx, spreadsheetID, range_, headers)
}

// UpdateTravelStatus updates the travel status sheet with current member data
func (c *Client) UpdateTravelStatus(ctx context.Context, spreadsheetID, sheetName string, records []app.TravelRecord) error {
	if len(records) == 0 {
		log.Debug().Str("sheet_name", sheetName).Msg("No travel records to update")
		return nil
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int("records_count", len(records)).
		Msg("Updating travel status sheet")

	// Clear existing data (except headers)
	if len(records) > 0 {
		clearRange := fmt.Sprintf("'%s'!A2:G", sheetName)
		if err := c.ClearRange(ctx, spreadsheetID, clearRange); err != nil {
			return fmt.Errorf("failed to clear existing travel data: %w", err)
		}
	}

	// Convert records to rows
	rows := c.convertTravelRecordsToRows(records)

	// Calculate required sheet dimensions
	requiredRows := len(rows) + 1 // +1 for header
	requiredCols := 7             // A-G columns

	// Ensure sheet has sufficient capacity
	if err := c.EnsureSheetCapacity(ctx, spreadsheetID, sheetName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Update with new data starting from row 2
	if len(rows) > 0 {
		endRow := len(rows) + 1
		range_ := fmt.Sprintf("'%s'!A2:G%d", sheetName, endRow)
		return c.UpdateRange(ctx, spreadsheetID, range_, rows)
	}

	return nil
}

// convertTravelRecordsToRows converts TravelRecord structs to sheet row format
func (c *Client) convertTravelRecordsToRows(records []app.TravelRecord) [][]interface{} {
	var rows [][]interface{}

	for _, record := range records {
		row := []interface{}{
			record.Name,
			record.Level,
			record.Location,
			record.State,
			record.Departure,
			record.Arrival,
			record.Countdown,
		}
		rows = append(rows, row)
	}

	return rows
}

// EnsureStateChangeRecordsSheet creates state change records sheet for a faction if it doesn't exist
func (c *Client) EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := fmt.Sprintf("State Change Records - %d", factionID)

	log.Debug().
		Int("faction_id", factionID).
		Str("sheet_name", sheetName).
		Msg("Ensuring state change records sheet exists")

	// Check if sheet exists
	exists, err := c.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if state change records sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Msg("Creating state change records sheet")

		if err := c.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create state change records sheet: %w", err)
		}

		// Initialize sheet with headers
		if err := c.initializeStateChangeRecordsSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize state change records sheet: %w", err)
		}
	}

	return sheetName, nil
}

// initializeStateChangeRecordsSheet sets up headers for a state change records sheet
func (c *Client) initializeStateChangeRecordsSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := [][]interface{}{
		{
			"Timestamp",
			"Member Name",
			"Member ID",
			"Faction Name",
			"Faction ID",
			"Last Action Status",
			"Status Description",
			"Status State",
			"Status Color",
			"Status Details",
			"Status Until",
			"Status Travel Type",
			"Status Plane Image Type",
		},
	}

	range_ := fmt.Sprintf("'%s'!A1:M1", sheetName)
	return c.UpdateRange(ctx, spreadsheetID, range_, headers)
}

// AddStateChangeRecord adds a new state change record to the sheet
func (c *Client) AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error {
	log.Debug().
		Str("sheet_name", sheetName).
		Str("member_name", record.MemberName).
		Int("member_id", record.MemberID).
		Msg("Adding state change record")

	// Convert record to row format
	row := []interface{}{
		record.Timestamp.Format("2006-01-02 15:04:05"),
		record.MemberName,
		record.MemberID,
		record.FactionName,
		record.FactionID,
		record.LastActionStatus,
		record.StatusDescription,
		record.StatusState,
		record.StatusColor,
		record.StatusDetails,
		record.StatusUntil,
		record.StatusTravelType,
		record.StatusPlaneImageType,
	}

	// Append to the sheet
	range_ := fmt.Sprintf("'%s'!A:M", sheetName)
	if err := c.AppendRows(ctx, spreadsheetID, range_, [][]interface{}{row}); err != nil {
		return fmt.Errorf("failed to append state change record: %w", err)
	}

	return nil
}

// EnsurePreviousStateSheet creates previous state sheet for a faction if it doesn't exist
func (c *Client) EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	sheetName := fmt.Sprintf("Previous State - %d", factionID)

	log.Debug().
		Int("faction_id", factionID).
		Str("sheet_name", sheetName).
		Msg("Ensuring previous state sheet exists")

	// Check if sheet exists
	exists, err := c.SheetExists(ctx, spreadsheetID, sheetName)
	if err != nil {
		return "", fmt.Errorf("failed to check if previous state sheet exists: %w", err)
	}

	if !exists {
		log.Info().
			Str("sheet_name", sheetName).
			Msg("Creating previous state sheet")

		if err := c.CreateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to create previous state sheet: %w", err)
		}

		// Initialize sheet with headers
		if err := c.initializePreviousStateSheet(ctx, spreadsheetID, sheetName); err != nil {
			return "", fmt.Errorf("failed to initialize previous state sheet: %w", err)
		}
	}

	return sheetName, nil
}

// initializePreviousStateSheet sets up headers for a previous state sheet
func (c *Client) initializePreviousStateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	headers := [][]interface{}{
		{
			"User ID",
			"Member Name",
			"Level",
			"Days In Faction",
			"Position",
			"Last Action Status",
			"Last Action Timestamp",
			"Last Action Relative",
			"Status Description",
			"Status State",
			"Status Color",
			"Status Details",
			"Status Until",
			"Status Travel Type",
			"Status Plane Image Type",
		},
	}

	range_ := fmt.Sprintf("'%s'!A1:O1", sheetName)
	return c.UpdateRange(ctx, spreadsheetID, range_, headers)
}

// StorePreviousMemberStates stores current member states to the previous state sheet
func (c *Client) StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error {
	log.Debug().
		Str("sheet_name", sheetName).
		Int("members_count", len(members)).
		Msg("Storing previous member states")

	// Clear existing data (except headers)
	if len(members) > 0 {
		clearRange := fmt.Sprintf("'%s'!A2:O", sheetName)
		if err := c.ClearRange(ctx, spreadsheetID, clearRange); err != nil {
			return fmt.Errorf("failed to clear existing state data: %w", err)
		}
	}

	// Convert members to rows
	rows := c.convertMembersToStateRows(members)

	if len(rows) > 0 {
		// Calculate required sheet dimensions
		requiredRows := len(rows) + 1 // +1 for header
		requiredCols := 15

		// Ensure sheet has enough capacity
		if err := c.EnsureSheetCapacity(ctx, spreadsheetID, sheetName, requiredRows, requiredCols); err != nil {
			return fmt.Errorf("failed to ensure sheet capacity: %w", err)
		}

		// Update with new data
		range_ := fmt.Sprintf("'%s'!A2:O%d", sheetName, len(rows)+1)
		if err := c.UpdateRange(ctx, spreadsheetID, range_, rows); err != nil {
			return fmt.Errorf("failed to update previous state data: %w", err)
		}
	}

	return nil
}

// LoadPreviousMemberStates loads previous member states from the sheet
func (c *Client) LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error) {
	log.Debug().
		Str("sheet_name", sheetName).
		Msg("Loading previous member states")

	// Read all data from sheet (starting from row 2 to skip headers)
	range_ := fmt.Sprintf("'%s'!A2:O", sheetName)
	rows, err := c.ReadSheet(ctx, spreadsheetID, range_)
	if err != nil {
		return nil, fmt.Errorf("failed to read previous state data: %w", err)
	}

	members := make(map[string]app.FactionMember)

	for i, row := range rows {
		// Pad row to expected length if needed (Google Sheets omits trailing empty cells)
		for len(row) < 15 {
			row = append(row, "")
		}

		if len(row) < 15 {
			log.Warn().
				Int("row", i+2).
				Int("columns", len(row)).
				Msg("Skipping row with insufficient columns after padding")
			continue
		}

		// Parse the row data
		userIDStr, ok := row[0].(string)
		if !ok {
			log.Warn().
				Int("row", i+2).
				Msg("Invalid user ID format, skipping row")
			continue
		}

		member := app.FactionMember{
			Name:          parseStringValue(row[1]),
			Level:         parseIntValue(row[2]),
			DaysInFaction: parseIntValue(row[3]),
			Position:      parseStringValue(row[4]),
			LastAction: app.LastAction{
				Status:    parseStringValue(row[5]),
				Timestamp: parseInt64Value(row[6]),
				Relative:  parseStringValue(row[7]),
			},
			Status: app.MemberStatus{
				Description:    parseStringValue(row[8]),
				State:          parseStringValue(row[9]),
				Color:          parseStringValue(row[10]),
				Details:        parseStringValue(row[11]),
				Until:          parseInt64PointerValue(row[12]),
				TravelType:     parseStringValue(row[13]),
				PlaneImageType: parseStringValue(row[14]),
			},
		}

		members[userIDStr] = member
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int("members_loaded", len(members)).
		Msg("Loaded previous member states")

	return members, nil
}

// convertMembersToStateRows converts member map to sheet row format
func (c *Client) convertMembersToStateRows(members map[string]app.FactionMember) [][]interface{} {
	var rows [][]interface{}

	for userID, member := range members {
		row := []interface{}{
			userID,
			member.Name,
			member.Level,
			member.DaysInFaction,
			member.Position,
			member.LastAction.Status,
			member.LastAction.Timestamp,
			member.LastAction.Relative,
			member.Status.Description,
			member.Status.State,
			member.Status.Color,
			member.Status.Details,
			member.Status.Until,
			member.Status.TravelType,
			member.Status.PlaneImageType,
		}
		rows = append(rows, row)
	}

	return rows
}

// Helper functions for parsing sheet values
func parseStringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func parseIntValue(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

func parseInt64Value(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func parseInt64PointerValue(val interface{}) *int64 {
	if val == nil || val == "" {
		return nil
	}
	i := parseInt64Value(val)
	return &i
}
