package sheets

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
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

// RecordsInfo contains information about existing records in a sheet
type RecordsInfo struct {
	LastTimestamp    int64
	RecordCount      int
	LastRowProcessed int
}

// ReadExistingRecords reads existing attack records from a sheet to determine what's already there
func (p *AttackRecordsProcessor) ReadExistingRecords(ctx context.Context, spreadsheetID, sheetName string) (*RecordsInfo, error) {
	// Read the timestamp column (column A) starting from row 2 (skip header)
	rangeSpec := fmt.Sprintf("%s!A2:A", sheetName)

	values, err := p.api.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing records: %w", err)
	}

	info := &RecordsInfo{
		LastTimestamp:    0,
		RecordCount:      0,
		LastRowProcessed: 1, // Header is row 1
	}

	// Find the last (highest) timestamp
	for i, row := range values {
		if len(row) == 0 || row[0] == nil {
			continue
		}

		timestampStr := parseStringValue(row[0])
		if timestampStr == "" {
			continue
		}

		// Parse the timestamp
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			log.Debug().
				Str("timestamp_str", timestampStr).
				Int("row", i+2). // +2 because we start from row 2
				Err(err).
				Msg("Failed to parse timestamp in existing records")
			continue
		}

		if timestamp > info.LastTimestamp {
			info.LastTimestamp = timestamp
		}

		info.RecordCount++
		info.LastRowProcessed = i + 2 // +2 because we start from row 2
	}

	log.Debug().
		Str("sheet_name", sheetName).
		Int64("last_timestamp", info.LastTimestamp).
		Int("record_count", info.RecordCount).
		Int("last_row", info.LastRowProcessed).
		Msg("Read existing records info")

	return info, nil
}

// UpdateAttackRecords updates the attack records sheet with new records
func (p *AttackRecordsProcessor) UpdateAttackRecords(ctx context.Context, spreadsheetID string, config *app.SheetConfig, records []app.AttackRecord) error {
	if len(records) == 0 {
		log.Debug().
			Str("sheet_name", config.RecordsTabName).
			Msg("No attack records to update")
		return nil
	}

	// Read existing records to determine what's new
	existing, err := p.ReadExistingRecords(ctx, spreadsheetID, config.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Filter and sort new records
	newRecords := p.FilterAndSortRecords(records, existing)

	if len(newRecords) == 0 {
		log.Debug().
			Str("sheet_name", config.RecordsTabName).
			Int("total_records", len(records)).
			Msg("No new records to add after filtering")
		return nil
	}

	// Convert to spreadsheet format
	rows := p.ConvertRecordsToRows(newRecords)

	// Determine the range to append to
	startRow := existing.LastRowProcessed + 1
	rangeSpec := fmt.Sprintf("%s!A%d", config.RecordsTabName, startRow)

	// Ensure sheet has enough capacity
	requiredRows := startRow + len(rows)
	requiredCols := 13 // Based on our header structure
	if err := p.api.EnsureSheetCapacity(ctx, spreadsheetID, config.RecordsTabName, requiredRows, requiredCols); err != nil {
		return fmt.Errorf("failed to ensure sheet capacity: %w", err)
	}

	// Append the new records
	if err := p.api.AppendRows(ctx, spreadsheetID, rangeSpec, rows); err != nil {
		return fmt.Errorf("failed to append attack records: %w", err)
	}

	log.Info().
		Int("war_id", config.WarID).
		Str("sheet_name", config.RecordsTabName).
		Int("new_records_added", len(newRecords)).
		Int("total_input_records", len(records)).
		Msg("Updated attack records sheet")

	return nil
}

// FilterAndSortRecords filters out existing records and sorts by timestamp
func (p *AttackRecordsProcessor) FilterAndSortRecords(records []app.AttackRecord, existing *RecordsInfo) []app.AttackRecord {
	var newRecords []app.AttackRecord

	// Filter out records that already exist (timestamp <= last existing timestamp)
	for _, record := range records {
		recordTimestamp := record.Started.Unix()
		if recordTimestamp > existing.LastTimestamp {
			newRecords = append(newRecords, record)
		}
	}

	// Sort by timestamp (oldest first) for consistent ordering
	sort.Slice(newRecords, func(i, j int) bool {
		return newRecords[i].Started.Unix() < newRecords[j].Started.Unix()
	})

	log.Debug().
		Int("input_records", len(records)).
		Int("filtered_records", len(newRecords)).
		Int64("last_existing_timestamp", existing.LastTimestamp).
		Msg("Filtered and sorted attack records")

	return newRecords
}

// ConvertRecordsToRows converts attack records into spreadsheet row format
func (p *AttackRecordsProcessor) ConvertRecordsToRows(records []app.AttackRecord) [][]interface{} {
	rows := make([][]interface{}, len(records))

	for i, record := range records {
		// Format timestamp as readable date/time
		timestamp := record.Started.UTC()
		dateStr := timestamp.Format("2006-01-02")
		timeStr := timestamp.Format("15:04:05")

		// Calculate respect per chain if chain > 0 (placeholder logic)
		respectPerChain := ""
		// Note: AttackRecord doesn't have Chain/Respect fields in current structure
		// This would need to be calculated from actual war/attack data

		rows[i] = []interface{}{
			record.Started.Unix(),      // Timestamp (for sorting/filtering)
			dateStr,                    // Date
			timeStr,                    // Time
			record.Direction,           // Direction (Attack/Defense)
			record.AttackerName,        // Attacker
			record.AttackerFactionName, // Attacker Faction
			record.DefenderName,        // Defender
			record.DefenderFactionName, // Defender Faction
			record.Code,                // Result/Code
			0,                          // Respect (placeholder - would need calculation)
			0,                          // Chain (placeholder - would need calculation)
			respectPerChain,            // Respect/Chain
			record.AttackID,            // Attack ID
		}
	}

	return rows
}

// ParseValue functions for reading data from sheets
// These are pure functions that can be easily unit tested
// Note: Using functions from wars.go to avoid duplication
