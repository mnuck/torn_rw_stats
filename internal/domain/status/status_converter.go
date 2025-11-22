package status

import (
	"torn_rw_stats/internal/app"
)

// StatusRow represents a single row in the status sheet
type StatusRow struct {
	MemberID  int
	Name      string
	Level     int
	Status    string
	Location  string
	Countdown string
	Departure string
	Arrival   string
}

// ConversionInput contains all data needed for conversion
type ConversionInput struct {
	StateRecords []app.StateRecord
	ExistingData map[int]StatusRow // Keyed by member ID
	WarID        int
}

// ConvertToStatusV2 converts state records to sheet rows (pure function)
func ConvertToStatusV2(input ConversionInput) [][]interface{} {
	result := make([][]interface{}, 0, len(input.StateRecords))

	for _, record := range input.StateRecords {
		row := convertSingleRecord(record, input.ExistingData)
		result = append(result, row)
	}

	return result
}

// convertSingleRecord converts one state record (pure function)
func convertSingleRecord(
	record app.StateRecord,
	existingData map[int]StatusRow,
) []interface{} {
	// Parse member ID
	memberID := parseInt(record.MemberID)

	// Check for existing data
	existing, hasExisting := existingData[memberID]

	row := make([]interface{}, 0, 10)
	row = append(row, record.MemberID)
	row = append(row, record.MemberName)
	row = append(row, 0) // Level placeholder - filled from faction data in application layer

	// Use existing data if available for unchanged fields
	if hasExisting && record.StatusState == existing.Status {
		row = append(row, existing.Status)
		row = append(row, existing.Location)
		row = append(row, existing.Countdown)
		row = append(row, existing.Departure)
		row = append(row, existing.Arrival)
	} else {
		row = append(row, record.StatusState)
		row = append(row, record.StatusDescription)
		row = append(row, "") // Countdown - calculated in application layer
		row = append(row, "") // Departure - preserved from existing or calculated
		row = append(row, "") // Arrival - preserved from existing or calculated
	}

	return row
}

// ParseExistingStatusData converts raw sheet data to structured format (pure function)
func ParseExistingStatusData(rawData [][]interface{}) map[int]StatusRow {
	result := make(map[int]StatusRow)

	for _, row := range rawData {
		if len(row) < 3 {
			continue
		}

		memberID := parseInt(toString(row[0]))
		if memberID == 0 {
			continue
		}

		statusRow := StatusRow{
			MemberID: memberID,
			Name:     toString(row[1]),
			Level:    toInt(row[2]),
		}

		if len(row) > 3 {
			statusRow.Status = toString(row[3])
		}
		if len(row) > 4 {
			statusRow.Location = toString(row[4])
		}
		if len(row) > 5 {
			statusRow.Countdown = toString(row[5])
		}
		if len(row) > 6 {
			statusRow.Departure = toString(row[6])
		}
		if len(row) > 7 {
			statusRow.Arrival = toString(row[7])
		}

		result[memberID] = statusRow
	}

	return result
}

// toString converts interface{} to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// toInt converts interface{} to int
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

// parseInt converts string to int
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}
