package sheets

import (
	"context"
	"fmt"
	"sort"
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
		WarID:           war.ID,
		SummaryTabName:  summaryTabName,
		RecordsTabName:  recordsTabName,
		SpreadsheetID:   spreadsheetID,
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

	log.Debug().
		Int("war_id", config.WarID).
		Str("sheet_name", config.RecordsTabName).
		Int("records_count", len(records)).
		Msg("Updating attack records sheet")

	// Read existing records to determine update strategy
	existingInfo, err := c.ReadExistingRecords(ctx, spreadsheetID, config.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Filter out duplicate attacks and sort chronologically
	newRecords := c.filterAndSortRecords(records, existingInfo)
	
	if len(newRecords) == 0 {
		log.Debug().Msg("No new records to add after deduplication")
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
	
	return c.UpdateRange(ctx, spreadsheetID, range_, rows)
}

// filterAndSortRecords removes duplicates and sorts records chronologically
func (c *Client) filterAndSortRecords(records []app.AttackRecord, existing *ExistingRecordsInfo) []app.AttackRecord {
	var newRecords []app.AttackRecord

	// Filter out duplicates using attack codes (guaranteed unique strings)
	for _, record := range records {
		if !existing.AttackCodes[record.Code] {
			newRecords = append(newRecords, record)
		}
	}

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