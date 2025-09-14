package sheets

import (
	"context"
	"fmt"
	"sort"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// War-related API functions that use the infrastructure layer
// These functions delegate to the specialized managers for actual business logic

// EnsureWarSheets creates summary and records sheets for a war if they don't exist
func (c *Client) EnsureWarSheets(ctx context.Context, spreadsheetID string, war *app.War) (*app.SheetConfig, error) {
	manager := NewWarSheetsManager(c)
	return manager.EnsureWarSheets(ctx, spreadsheetID, war)
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

// Travel and State Management Functions - delegate to specialized managers

// convertTravelRecordsToRows converts TravelRecord structs to sheet row format
func (c *Client) convertTravelRecordsToRows(records []app.TravelRecord) [][]interface{} {
	manager := NewTravelStatusManager(c)
	return manager.ConvertTravelRecordsToRows(records)
}

// convertMembersToStateRows converts member map to sheet row format
func (c *Client) convertMembersToStateRows(members map[string]app.FactionMember) [][]interface{} {
	manager := NewStateChangeManager(c)
	return manager.ConvertMembersToStateRows(members)
}

// EnsureTravelStatusSheet creates travel status sheet for a faction if it doesn't exist
func (c *Client) EnsureTravelStatusSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewTravelStatusManager(c)
	return manager.EnsureTravelStatusSheet(ctx, spreadsheetID, factionID)
}

// UpdateTravelStatus updates the travel status sheet with current member data
func (c *Client) UpdateTravelStatus(ctx context.Context, spreadsheetID, sheetName string, records []app.TravelRecord) error {
	manager := NewTravelStatusManager(c)
	return manager.UpdateTravelStatus(ctx, spreadsheetID, sheetName, records)
}

// EnsureStateChangeRecordsSheet creates state change records sheet for a faction if it doesn't exist
func (c *Client) EnsureStateChangeRecordsSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewStateChangeManager(c)
	return manager.EnsureStateChangeRecordsSheet(ctx, spreadsheetID, factionID)
}

// AddStateChangeRecord adds a new state change record to the sheet
func (c *Client) AddStateChangeRecord(ctx context.Context, spreadsheetID, sheetName string, record app.StateChangeRecord) error {
	manager := NewStateChangeManager(c)
	return manager.AddStateChangeRecord(ctx, spreadsheetID, sheetName, record)
}

// EnsurePreviousStateSheet creates previous state sheet for a faction if it doesn't exist
func (c *Client) EnsurePreviousStateSheet(ctx context.Context, spreadsheetID string, factionID int) (string, error) {
	manager := NewStateChangeManager(c)
	return manager.EnsurePreviousStateSheet(ctx, spreadsheetID, factionID)
}

// StorePreviousMemberStates stores current member states to the previous state sheet
func (c *Client) StorePreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string, members map[string]app.FactionMember) error {
	manager := NewStateChangeManager(c)
	return manager.StorePreviousMemberStates(ctx, spreadsheetID, sheetName, members)
}

// LoadPreviousMemberStates loads previous member states from the sheet
func (c *Client) LoadPreviousMemberStates(ctx context.Context, spreadsheetID, sheetName string) (map[string]app.FactionMember, error) {
	manager := NewStateChangeManager(c)
	return manager.LoadPreviousMemberStates(ctx, spreadsheetID, sheetName)
}
