package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"

	"github.com/rs/zerolog/log"
)

// getExistingStatusV2Data reads existing Status v2 data to preserve manual adjustments
func (s *StatusV2Service) getExistingStatusV2Data(ctx context.Context, spreadsheetID string, factionID int) (map[string]app.StatusV2Record, error) {
	sheetName := fmt.Sprintf("Status v2 - %d", factionID)
	rangeSpec := fmt.Sprintf("%s!A2:J", sheetName)

	values, err := s.sheetsClient.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing Status v2 data: %w", err)
	}

	data := make(map[string]app.StatusV2Record)
	factionIDStr := strconv.Itoa(factionID)

	for _, row := range values {
		if len(row) < 8 {
			continue
		}

		// Extract member name and create key using type-safe Cell
		name := sheets.NewCell(row[0]).String()
		if name == "" {
			continue
		}

		// We'll use name as key since MemberID isn't in the sheet
		memberKey := fmt.Sprintf("%s_%s", factionIDStr, name)

		// Parse level using type-safe Cell
		level := 0
		levelStr := getString(row, 1)
		if l, err := strconv.Atoi(levelStr); err == nil {
			level = l
		}

		// Parse Until timestamp from column 9 (column J)
		var until time.Time
		if len(row) > 9 {
			if untilStr := getString(row, 9); untilStr != "" {
				if parsedUntil, err := time.Parse("2006-01-02 15:04:05", untilStr); err == nil {
					until = parsedUntil.UTC()
				}
			}
		}

		record := app.StatusV2Record{
			Name:            name,
			MemberID:        "", // MemberID not stored in spreadsheet, populated from StateRecord
			Level:           level,
			State:           getString(row, 2),
			Status:          getString(row, 3),
			Location:        getString(row, 4),
			Countdown:       getString(row, 5),
			Departure:       getString(row, 6),
			Arrival:         getString(row, 7),
			BusinessArrival: getString(row, 8), // Column I
			Until:           until,
		}

		data[memberKey] = record
	}

	return data, nil
}

// ReadAllStateRecords reads all state records from the Changed States sheet
func (s *StatusV2Service) ReadAllStateRecords(ctx context.Context, spreadsheetID string) ([]app.StateRecord, error) {
	sheetName := "Changed States"
	rangeSpec := fmt.Sprintf("%s!A2:L", sheetName)

	log.Info().
		Str("sheet_name", sheetName).
		Str("range_spec", rangeSpec).
		Msg("Reading state records from Changed States sheet")

	values, err := s.sheetsClient.ReadSheet(ctx, spreadsheetID, rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to read Changed States sheet: %w", err)
	}

	log.Info().
		Int("raw_rows_read", len(values)).
		Msg("Successfully read raw data from Changed States sheet")

	var records []app.StateRecord
	validRows := 0
	for i, row := range values {
		if len(row) < 8 {
			log.Debug().
				Int("row_index", i).
				Int("row_length", len(row)).
				Interface("row_sample", row).
				Msg("Skipping row with insufficient columns - showing data")
			continue
		}

		record, err := s.parseStateRecordFromRow(row)
		if err != nil {
			log.Warn().Err(err).Interface("row", row).Msg("Failed to parse state record from row")
			continue
		}

		records = append(records, record)
		validRows++
	}

	log.Info().
		Int("total_rows_processed", len(values)).
		Int("valid_state_records", validRows).
		Int("final_records_count", len(records)).
		Msg("Completed reading Changed States data")

	return records, nil
}

// parseStateRecordFromRow parses a spreadsheet row into a StateRecord
func (s *StatusV2Service) parseStateRecordFromRow(row []interface{}) (app.StateRecord, error) {
	var record app.StateRecord

	// Actual Changed States format: [Timestamp, Member ID, Member Name, Faction ID, Faction Name, Last Action Status, Status Description, Status State, Status Until, Status Travel Type]
	// NOTE: The sheet does NOT have Date and Time columns, so indices are shifted left by 2 from the header definition

	// Parse timestamp from column 0 - this is already formatted as "2025-09-15 1:08:57"
	if timestampStr, ok := row[0].(string); ok {
		if timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
			record.Timestamp = timestamp.UTC()
		}
	}

	record.MemberID = getString(row, 1)
	record.MemberName = getString(row, 2)
	record.FactionID = getString(row, 3)
	record.FactionName = getString(row, 4)
	record.LastActionStatus = getString(row, 5)
	record.StatusDescription = getString(row, 6)
	record.StatusState = getString(row, 7)

	// Parse StatusUntil from column 8 (optional - only present for some status types)
	if len(row) > 8 {
		if statusUntilStr := getString(row, 8); statusUntilStr != "" {
			if statusUntil, err := time.Parse("2006-01-02 15:04:05", statusUntilStr); err == nil {
				record.StatusUntil = statusUntil.UTC()
			}
		}
	}

	// Parse StatusTravelType from column 9 (optional - only present for traveling status)
	if len(row) > 9 {
		record.StatusTravelType = getString(row, 9)
	}

	return record, nil
}

// getString safely gets a string from a spreadsheet row using type-safe Cell wrapper
func getString(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	return sheets.NewCell(row[index]).String()
}
