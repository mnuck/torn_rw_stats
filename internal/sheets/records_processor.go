package sheets

import (
	"context"
	"fmt"
	"sort"
	"time"

	"torn_rw_stats/internal/application/ports"
	"torn_rw_stats/internal/domain"

	"log/slog"
)

// AttackRecordsProcessor handles business logic for attack records management
// Separated from infrastructure concerns for better testability
type AttackRecordsProcessor struct {
	api SheetsAPI
}

// NewAttackRecordsProcessor creates a new attack records processor with the given API client
func NewAttackRecordsProcessor(api SheetsAPI) *AttackRecordsProcessor {
	return &AttackRecordsProcessor{
		api: api,
	}
}

// ReadExistingRecords reads existing attack records from a sheet to determine what's already there
func (p *AttackRecordsProcessor) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*domain.RecordsInfo, error) {
	slog.Debug("Reading existing attack records",
		"sheet_name", sheetName)

	// Read all data from the sheet (starting from row 2 to skip headers)
	rangeSpec := fmt.Sprintf("'%s'!A2:AF", sheetName)
	values, err := p.api.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing records: %w", err)
	}

	info := &domain.RecordsInfo{
		AttackCodes:      make(map[string]bool),
		LatestTimestamp:  0,
		RecordCount:      len(values),
		LastRowProcessed: 1, // Header is row 1
	}

	validRows := 0
	for _, row := range values {
		if len(row) < 3 { // Need at least Code and Started timestamp
			continue
		}

		// Parse Attack Code (column B) - always a string
		codeStr := ports.NewCell(row[1]).String()
		if codeStr != "" {
			info.AttackCodes[codeStr] = true
			validRows++
		}

		// Parse Started timestamp (column C) to find latest
		startedStr := ports.NewCell(row[2]).String()
		if startedTime, err := time.Parse("2006-01-02 15:04:05", startedStr); err == nil {
			timestamp := startedTime.Unix()
			if timestamp > info.LatestTimestamp {
				info.LatestTimestamp = timestamp
			}
		}
	}

	// Update record count to reflect valid rows only
	info.RecordCount = validRows
	info.LastRowProcessed = len(values) + 1 // +1 for header row

	slog.Debug("Analyzed existing records",
		"total_rows_read", len(values),
		"valid_records", info.RecordCount,
		"unique_attack_codes", len(info.AttackCodes),
		"latest_timestamp", info.LatestTimestamp,
		"latest_time", time.Unix(info.LatestTimestamp, 0).Format("2006-01-02 15:04:05"))

	// Validation: warn if no attack codes were parsed from non-empty sheet
	if len(values) > 0 && len(info.AttackCodes) == 0 {
		slog.Warn("No attack codes parsed from existing sheet - possible column mismatch",
			"rows_in_sheet", len(values))
	}

	return info, nil
}

// UpdateAttackRecords updates the attack records sheet with new records
func (p *AttackRecordsProcessor) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *domain.SheetConfig, records []domain.AttackRecord) error {
	if len(records) == 0 {
		return nil
	}

	slog.Info("=== ENTERING UpdateAttackRecords ===",
		"war_id", config.WarID,
		"sheet_name", config.RecordsTabName,
		"records_count", len(records))

	// Read existing records to determine update strategy
	existing, err := p.ReadExistingRecords(ctx, spreadsheetID, config.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Filter out duplicate attacks and sort chronologically
	slog.Debug("Starting deduplication",
		"input_records", len(records),
		"existing_attack_codes", len(existing.AttackCodes),
		"existing_record_count", existing.RecordCount)

	newRecords := p.FilterAndSortRecords(records, existing)

	if len(newRecords) == 0 {
		slog.Info("=== EXITING UpdateAttackRecords - No new records after deduplication ===")
		return nil
	}

	slog.Info("Processed records for update",
		"original_records", len(records),
		"new_records", len(newRecords),
		"existing_records", existing.RecordCount)

	// Convert to spreadsheet format
	rows := p.ConvertRecordsToRows(newRecords)

	// Calculate required sheet dimensions (matching wars_api.go approach)
	startRow := existing.RecordCount + 2 // +2 for header row and 1-based indexing
	endRow := startRow + len(rows) - 1
	requiredRows := endRow
	requiredCols := 32 // AF column = 32

	// Ensure sheet has sufficient capacity
	if err := p.api.EnsureSheetCapacity(ctx, spreadsheetID, config.RecordsTabName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Append new rows to the sheet
	rangeSpec := fmt.Sprintf("'%s'!A%d:AF%d", config.RecordsTabName, startRow, endRow)

	// Log first few rows being written to detect duplicates at write time
	sampleRows := make([]string, 0, 3)
	for i, row := range rows {
		if i < 3 && len(row) >= 2 {
			attackID := row[0]
			if code, ok := row[1].(string); ok {
				sampleRows = append(sampleRows, fmt.Sprintf("ID:%v Code:%s", attackID, code))
			}
		}
	}

	slog.Info("=== WRITING TO SHEET ===",
		"range", rangeSpec,
		"rows_to_write", len(rows),
		"sample_rows", sampleRows)

	// Use UpdateRange instead of AppendRows for exact range specification
	err = p.api.UpdateRange(ctx, spreadsheetID, rangeSpec, rows)
	if err != nil {
		return fmt.Errorf("failed to append attack records: %w", err)
	}

	slog.Info("=== EXITING UpdateAttackRecords - Successfully appended records ===",
		"war_id", config.WarID,
		"records_appended", len(newRecords),
		"range", rangeSpec)

	return nil
}

// FilterAndSortRecords filters out existing records and sorts by timestamp
func (p *AttackRecordsProcessor) FilterAndSortRecords(records []domain.AttackRecord, existing *domain.RecordsInfo) []domain.AttackRecord {
	var newRecords []domain.AttackRecord

	// Filter out duplicates using attack codes AND records older than existing timestamp
	duplicates := 0
	for _, record := range records {
		// Skip if duplicate attack code
		if existing.AttackCodes[record.Code] {
			duplicates++
			slog.Debug("Filtered duplicate attack",
				"attack_code", record.Code,
				"attack_id", record.AttackID)
			continue
		}

		// Skip if record is older than or equal to existing timestamp (already processed)
		if record.Started.Unix() <= existing.LatestTimestamp {
			duplicates++
			slog.Debug("Filtered old attack (timestamp)",
				"attack_id", record.AttackID,
				"record_timestamp", record.Started.Unix(),
				"existing_timestamp", existing.LatestTimestamp)
			continue
		}

		// Record is new and recent enough
		newRecords = append(newRecords, record)
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

	slog.Info("Completed deduplication filtering",
		"input_records", len(records),
		"duplicates_filtered", duplicates,
		"new_records", len(newRecords),
		"sample_attack_codes", sampleCodes)

	// Sort chronologically (oldest first)
	sort.Slice(newRecords, func(i, j int) bool {
		return newRecords[i].Started.Before(newRecords[j].Started)
	})

	slog.Debug("Filtered and sorted records chronologically",
		"filtered_records", len(newRecords))

	return newRecords
}

// ConvertRecordsToRows converts attack records into spreadsheet row format
func (p *AttackRecordsProcessor) ConvertRecordsToRows(records []domain.AttackRecord) [][]interface{} {
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
			record.Started.UTC().Format("2006-01-02 15:04:05"),
			record.Ended.UTC().Format("2006-01-02 15:04:05"),
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

// ParseValue functions for reading data from sheets
// These are pure functions that can be easily unit tested
// Note: Using functions from wars.go to avoid duplication
