package status

import (
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
