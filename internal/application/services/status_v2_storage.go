package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
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

// readCurrentStateRecords returns the latest state record per member for a faction via BigQuery.
func (s *StatusV2Service) readCurrentStateRecords(ctx context.Context, _ string, factionID string) ([]app.StateRecord, error) {
	if s.bigqueryClient == nil {
		return nil, nil
	}
	return s.bigqueryClient.QueryLatestStatePerFaction(ctx, factionID)
}

// getString safely gets a string from a spreadsheet row using type-safe Cell wrapper
func getString(row []interface{}, index int) string {
	if index >= len(row) {
		return ""
	}
	return sheets.NewCell(row[index]).String()
}
