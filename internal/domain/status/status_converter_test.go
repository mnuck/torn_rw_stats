package status

import (
	"fmt"
	"testing"

	"torn_rw_stats/internal/app"
)

func TestConvertToStatusV2(t *testing.T) {
	tests := []struct {
		name         string
		input        ConversionInput
		expectedRows int
	}{
		{
			name: "converts basic state records",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay"},
					{MemberID: "2", MemberName: "Player2", StatusState: "Hospital"},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 2,
		},
		{
			name: "merges with existing data",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay"},
				},
				ExistingData: map[int]StatusRow{
					1: {MemberID: 1, Name: "Player1", Level: 50, Status: "Okay"},
				},
				WarID: 12345,
			},
			expectedRows: 1,
		},
		{
			name: "handles empty state records",
			input: ConversionInput{
				StateRecords: []app.StateRecord{},
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStatusV2(tt.input)

			if len(result) != tt.expectedRows {
				t.Errorf("expected %d rows, got %d", tt.expectedRows, len(result))
			}

			// Verify each row has expected structure
			for i, row := range result {
				if len(row) < 8 {
					t.Errorf("row %d: expected at least 8 columns, got %d", i, len(row))
				}
			}
		})
	}
}

func TestParseExistingStatusData(t *testing.T) {
	tests := []struct {
		name         string
		rawData      [][]interface{}
		expectedSize int
		checkData    bool
		expectedData map[int]StatusRow
	}{
		{
			name: "parses valid data",
			rawData: [][]interface{}{
				{"1", "Player1", 50, "Okay"},
				{"2", "Player2", 60, "Hospital"},
			},
			expectedSize: 2,
			checkData:    true,
			expectedData: map[int]StatusRow{
				1: {MemberID: 1, Name: "Player1", Level: 50, Status: "Okay"},
				2: {MemberID: 2, Name: "Player2", Level: 60, Status: "Hospital"},
			},
		},
		{
			name: "handles malformed rows",
			rawData: [][]interface{}{
				{"1", "Player1", 50, "Okay"},
				{"invalid"}, // Malformed - too short
				{"2", "Player2", 60, "Hospital"},
			},
			expectedSize: 2, // Should skip malformed row
		},
		{
			name: "handles rows with invalid member IDs",
			rawData: [][]interface{}{
				{"1", "Player1", 50},
				{"", "Player2", 60},    // Empty ID
				{"abc", "Player3", 70}, // Non-numeric ID
				{"3", "Player4", 80},
			},
			expectedSize: 2, // Should skip invalid IDs
		},
		{
			name: "parses extended data",
			rawData: [][]interface{}{
				{"1", "Player1", 50, "Traveling", "Japan", "1:30:00", "2025-01-01", "2025-01-02"},
			},
			expectedSize: 1,
			checkData:    true,
			expectedData: map[int]StatusRow{
				1: {
					MemberID:  1,
					Name:      "Player1",
					Level:     50,
					Status:    "Traveling",
					Location:  "Japan",
					Countdown: "1:30:00",
					Departure: "2025-01-01",
					Arrival:   "2025-01-02",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseExistingStatusData(tt.rawData)

			if len(result) != tt.expectedSize {
				t.Errorf("expected %d entries, got %d", tt.expectedSize, len(result))
			}

			if tt.checkData {
				for memberID, expected := range tt.expectedData {
					actual, exists := result[memberID]
					if !exists {
						t.Errorf("expected member %d not found in result", memberID)
						continue
					}

					if actual.Name != expected.Name {
						t.Errorf("member %d: expected name %s, got %s", memberID, expected.Name, actual.Name)
					}
					if actual.Level != expected.Level {
						t.Errorf("member %d: expected level %d, got %d", memberID, expected.Level, actual.Level)
					}
					if actual.Status != expected.Status {
						t.Errorf("member %d: expected status %s, got %s", memberID, expected.Status, actual.Status)
					}
					if actual.Location != expected.Location {
						t.Errorf("member %d: expected location %s, got %s", memberID, expected.Location, actual.Location)
					}
				}
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{name: "string value", input: "test", expected: "test"},
		{name: "nil value", input: nil, expected: ""},
		{name: "int value", input: 123, expected: ""},
		{name: "empty string", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{name: "int value", input: 123, expected: 123},
		{name: "int64 value", input: int64(456), expected: 456},
		{name: "float64 value", input: float64(789.5), expected: 789},
		{name: "nil value", input: nil, expected: 0},
		{name: "string value", input: "123", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "simple number", input: "123", expected: 123},
		{name: "zero", input: "0", expected: 0},
		{name: "large number", input: "999999", expected: 999999},
		{name: "empty string", input: "", expected: 0},
		{name: "non-numeric", input: "abc", expected: 0},
		{name: "mixed alphanumeric", input: "12abc34", expected: 1234},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestConvertToStatusV2EdgeCases tests edge cases and boundary conditions
func TestConvertToStatusV2EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		input        ConversionInput
		expectedRows int
		description  string
	}{
		{
			name: "handles nil existing data map",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay"},
				},
				ExistingData: nil,
				WarID:        12345,
			},
			expectedRows: 1,
			description:  "should handle nil existing data without panicking",
		},
		{
			name: "handles special characters in names",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player's Name [TAG]", StatusState: "Okay"},
					{MemberID: "2", MemberName: "Name-With-Dash", StatusState: "Hospital"},
					{MemberID: "3", MemberName: "Name.With.Dots", StatusState: "Traveling"},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 3,
			description:  "should handle special characters in player names",
		},
		{
			name: "handles non-numeric member IDs",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "abc", MemberName: "Invalid1", StatusState: "Okay"},
					{MemberID: "", MemberName: "Invalid2", StatusState: "Hospital"},
					{MemberID: "123", MemberName: "Valid", StatusState: "Okay"},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 3,
			description:  "should process all records even with invalid IDs",
		},
		{
			name: "handles very long status descriptions",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{
						MemberID:          "1",
						MemberName:        "Player1",
						StatusState:       "Traveling",
						StatusDescription: "Traveling to " + string(make([]byte, 1000)),
					},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 1,
			description:  "should handle very long status descriptions",
		},
		{
			name: "merges correctly when existing data has same status",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Hospital"},
				},
				ExistingData: map[int]StatusRow{
					1: {
						MemberID:  1,
						Name:      "Player1",
						Status:    "Hospital",
						Location:  "Torn City",
						Countdown: "5:00:00",
						Departure: "2025-01-01",
						Arrival:   "2025-01-02",
					},
				},
				WarID: 12345,
			},
			expectedRows: 1,
			description:  "should preserve existing data when status unchanged",
		},
		{
			name: "overwrites when existing data has different status",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay", StatusDescription: "Okay"},
				},
				ExistingData: map[int]StatusRow{
					1: {
						MemberID:  1,
						Name:      "Player1",
						Status:    "Hospital",
						Location:  "Torn City",
						Countdown: "5:00:00",
					},
				},
				WarID: 12345,
			},
			expectedRows: 1,
			description:  "should overwrite existing data when status changes",
		},
		{
			name: "handles large batch of records",
			input: ConversionInput{
				StateRecords: func() []app.StateRecord {
					records := make([]app.StateRecord, 1000)
					for i := 0; i < 1000; i++ {
						records[i] = app.StateRecord{
							MemberID:          fmt.Sprintf("%d", i+1),
							MemberName:        fmt.Sprintf("Player%d", i+1),
							StatusState:       "Okay",
							StatusDescription: "Okay",
						}
					}
					return records
				}(),
				ExistingData: make(map[int]StatusRow),
				WarID:        12345,
			},
			expectedRows: 1000,
			description:  "should handle large batches efficiently",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStatusV2(tt.input)

			if len(result) != tt.expectedRows {
				t.Errorf("%s: expected %d rows, got %d", tt.description, tt.expectedRows, len(result))
			}

			// Verify each row has minimum required structure
			for i, row := range result {
				if len(row) < 8 {
					t.Errorf("row %d: expected at least 8 columns, got %d", i, len(row))
				}
			}
		})
	}
}

// TestParseExistingStatusDataEdgeCases tests edge cases for parsing
func TestParseExistingStatusDataEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		rawData      [][]interface{}
		expectedSize int
		description  string
	}{
		{
			name:         "handles nil input",
			rawData:      nil,
			expectedSize: 0,
			description:  "should handle nil input without panicking",
		},
		{
			name:         "handles empty slice",
			rawData:      [][]interface{}{},
			expectedSize: 0,
			description:  "should handle empty slice",
		},
		{
			name: "handles rows with only 1 column",
			rawData: [][]interface{}{
				{"1"},
			},
			expectedSize: 0,
			description:  "should skip rows with insufficient columns",
		},
		{
			name: "handles rows with only 2 columns",
			rawData: [][]interface{}{
				{"1", "Player1"},
			},
			expectedSize: 0,
			description:  "should skip rows with only 2 columns (needs at least 3)",
		},
		{
			name: "handles mixed nil values",
			rawData: [][]interface{}{
				{nil, nil, nil, nil},
				{"1", "Player1", 50, "Okay"},
			},
			expectedSize: 1,
			description:  "should handle nil values gracefully",
		},
		{
			name: "handles rows with extra columns",
			rawData: [][]interface{}{
				{"1", "Player1", 50, "Okay", "Torn", "5:00", "2025-01-01", "2025-01-02", "ExtraColumn1", "ExtraColumn2"},
			},
			expectedSize: 1,
			description:  "should handle extra columns without error",
		},
		{
			name: "handles negative member IDs (becomes positive via parseInt)",
			rawData: [][]interface{}{
				{"-1", "InvalidPlayer", 50},
			},
			expectedSize: 1,
			description:  "parseInt extracts digits, so '-1' becomes 1",
		},
		{
			name: "handles very large member IDs",
			rawData: [][]interface{}{
				{"999999999", "Player1", 100, "Okay"},
			},
			expectedSize: 1,
			description:  "should handle very large member IDs",
		},
		{
			name: "handles special characters in all fields",
			rawData: [][]interface{}{
				{"123", "Player's [Name] (Test)", 50, "Status: Hospital", "Location: Torn City", "Time: 5:00:00", "Date: 2025-01-01", "Arrival: 2025-01-02"},
			},
			expectedSize: 1,
			description:  "should handle special characters in all fields",
		},
		{
			name: "deduplicates by member ID (last wins)",
			rawData: [][]interface{}{
				{"1", "FirstEntry", 10, "Hospital"},
				{"1", "SecondEntry", 20, "Okay"},
			},
			expectedSize: 1,
			description:  "should keep last entry for duplicate member IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseExistingStatusData(tt.rawData)

			if len(result) != tt.expectedSize {
				t.Errorf("%s: expected %d entries, got %d", tt.description, tt.expectedSize, len(result))
			}
		})
	}
}

// TestConversionInputValidation tests validation behavior
func TestConversionInputValidation(t *testing.T) {
	// Test that conversion works even with unusual but valid inputs
	tests := []struct {
		name        string
		input       ConversionInput
		shouldPanic bool
		description string
	}{
		{
			name: "handles zero war ID",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay"},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        0,
			},
			shouldPanic: false,
			description: "should handle zero war ID",
		},
		{
			name: "handles negative war ID",
			input: ConversionInput{
				StateRecords: []app.StateRecord{
					{MemberID: "1", MemberName: "Player1", StatusState: "Okay"},
				},
				ExistingData: make(map[int]StatusRow),
				WarID:        -1,
			},
			shouldPanic: false,
			description: "should handle negative war ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("%s: unexpected panic: %v", tt.description, r)
					}
				}
			}()

			result := ConvertToStatusV2(tt.input)

			if tt.shouldPanic {
				t.Errorf("%s: expected panic but none occurred", tt.description)
			}

			if len(result) != len(tt.input.StateRecords) {
				t.Errorf("%s: expected %d rows, got %d", tt.description, len(tt.input.StateRecords), len(result))
			}
		})
	}
}
