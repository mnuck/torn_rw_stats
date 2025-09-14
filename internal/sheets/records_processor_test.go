package sheets

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

func TestAttackRecordsProcessorReadExistingRecordsDetailed(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	testCases := []struct {
		name          string
		data          [][]interface{}
		expectedLast  int64
		expectedCount int
		expectError   bool
	}{
		{
			name: "valid timestamps",
			data: [][]interface{}{
				{"1000", "data1", "data2"},
				{"2000", "data3", "data4"},
				{"1500", "data5", "data6"},
			},
			expectedLast:  2000,
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "mixed valid and invalid timestamps",
			data: [][]interface{}{
				{"1000", "data1"},
				{"invalid", "data2"},
				{"2500", "data3"},
				{"", "data4"},
			},
			expectedLast:  2500,
			expectedCount: 2, // Only valid timestamps are counted
			expectError:   false,
		},
		{
			name:          "empty data",
			data:          [][]interface{}{},
			expectedLast:  0,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "rows with insufficient columns",
			data: [][]interface{}{
				{},
				{"1000"},
				{"2000", "data"},
			},
			expectedLast:  2000,
			expectedCount: 2, // Only rows with valid timestamps are counted
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI.SetSheetData("test_sheet", tc.data)

			info, err := processor.ReadExistingRecords(context.Background(), "test_spreadsheet", "test_sheet")

			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if info.LastTimestamp != tc.expectedLast {
				t.Errorf("Expected last timestamp %d, got %d", tc.expectedLast, info.LastTimestamp)
			}

			if info.RecordCount != tc.expectedCount {
				t.Errorf("Expected record count %d, got %d", tc.expectedCount, info.RecordCount)
			}
		})
	}
}

func TestAttackRecordsProcessorFilterAndSortRecordsComprehensive(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	testCases := []struct {
		name              string
		records           []app.AttackRecord
		existingTimestamp int64
		expectedCount     int
		expectedFirst     int64
		expectedLast      int64
	}{
		{
			name: "basic filtering and sorting",
			records: []app.AttackRecord{
				{AttackID: 1, Started: time.Unix(1000, 0)},
				{AttackID: 2, Started: time.Unix(500, 0)}, // Should be filtered out
				{AttackID: 3, Started: time.Unix(1500, 0)},
				{AttackID: 4, Started: time.Unix(750, 0)}, // Should be filtered out
				{AttackID: 5, Started: time.Unix(1200, 0)},
			},
			existingTimestamp: 800,
			expectedCount:     3,
			expectedFirst:     1000, // Oldest remaining
			expectedLast:      1500, // Newest remaining
		},
		{
			name: "all records filtered out",
			records: []app.AttackRecord{
				{AttackID: 1, Started: time.Unix(100, 0)},
				{AttackID: 2, Started: time.Unix(200, 0)},
			},
			existingTimestamp: 500,
			expectedCount:     0,
		},
		{
			name: "no filtering needed",
			records: []app.AttackRecord{
				{AttackID: 1, Started: time.Unix(1000, 0)},
				{AttackID: 2, Started: time.Unix(1500, 0)},
			},
			existingTimestamp: 500,
			expectedCount:     2,
			expectedFirst:     1000,
			expectedLast:      1500,
		},
		{
			name:              "empty records",
			records:           []app.AttackRecord{},
			existingTimestamp: 500,
			expectedCount:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			existing := &RecordsInfo{
				LastTimestamp: tc.existingTimestamp,
			}

			filtered := processor.FilterAndSortRecords(tc.records, existing)

			if len(filtered) != tc.expectedCount {
				t.Errorf("Expected %d filtered records, got %d", tc.expectedCount, len(filtered))
			}

			if tc.expectedCount > 0 {
				if filtered[0].Started.Unix() != tc.expectedFirst {
					t.Errorf("Expected first record timestamp %d, got %d", tc.expectedFirst, filtered[0].Started.Unix())
				}

				if len(filtered) > 1 && filtered[len(filtered)-1].Started.Unix() != tc.expectedLast {
					t.Errorf("Expected last record timestamp %d, got %d", tc.expectedLast, filtered[len(filtered)-1].Started.Unix())
				}
			}
		})
	}
}

func TestAttackRecordsProcessorConvertRecordsToRowsComprehensive(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	testTime := time.Unix(1640995200, 0) // 2022-01-01 00:00:00 UTC

	records := []app.AttackRecord{
		{
			AttackID:            123456,
			Started:             testTime,
			Direction:           "Outgoing",
			AttackerName:        "TestAttacker",
			AttackerFactionName: "AttackerFaction",
			DefenderName:        "TestDefender",
			DefenderFactionName: "DefenderFaction",
			Code:                "Win",
			Result:              "Victory",
			RespectGain:         25.5,
		},
		{
			AttackID:            789012,
			Started:             testTime.Add(1 * time.Hour),
			Direction:           "Incoming",
			AttackerName:        "EnemyAttacker",
			AttackerFactionName: "EnemyFaction",
			DefenderName:        "OurDefender",
			DefenderFactionName: "OurFaction",
			Code:                "Loss",
			Result:              "Defeat",
			RespectGain:         0.0,
		},
	}

	rows := processor.ConvertRecordsToRows(records)

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Test first record
	row1 := rows[0]
	if len(row1) < 13 {
		t.Fatalf("Expected at least 13 columns in first row, got %d", len(row1))
	}

	expectedValues1 := []interface{}{
		int64(1640995200), // Timestamp
		"2022-01-01",      // Date
		"00:00:00",        // Time
		"Outgoing",        // Direction
		"TestAttacker",    // Attacker
		"AttackerFaction", // Attacker Faction
		"TestDefender",    // Defender
		"DefenderFaction", // Defender Faction
		"Win",             // Code
		0,                 // Respect (placeholder)
		0,                 // Chain (placeholder)
		"",                // Respect/Chain (placeholder)
		int64(123456),     // Attack ID
	}

	for i, expected := range expectedValues1 {
		if i < len(row1) && row1[i] != expected {
			t.Errorf("Row 1, column %d: expected %v (%T), got %v (%T)", i, expected, expected, row1[i], row1[i])
		}
	}

	// Test second record
	row2 := rows[1]
	if len(row2) < 13 {
		t.Fatalf("Expected at least 13 columns in second row, got %d", len(row2))
	}

	// Check key fields for second record
	if row2[0] != int64(1640998800) { // 1 hour later
		t.Errorf("Expected timestamp 1640998800, got %v", row2[0])
	}

	if row2[3] != "Incoming" {
		t.Errorf("Expected direction 'Incoming', got %v", row2[3])
	}

	if row2[4] != "EnemyAttacker" {
		t.Errorf("Expected attacker 'EnemyAttacker', got %v", row2[4])
	}

	if row2[8] != "Loss" {
		t.Errorf("Expected code 'Loss', got %v", row2[8])
	}

	if row2[10] != 0 {
		t.Errorf("Expected respect (placeholder) 0, got %v", row2[10])
	}
}

func TestAttackRecordsProcessorUpdateAttackRecords(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	testTime := time.Unix(1640995200, 0)
	config := &app.SheetConfig{
		WarID:          123,
		RecordsTabName: "Records - 123",
	}

	records := []app.AttackRecord{
		{
			AttackID:     111,
			Started:      testTime,
			Direction:    "Outgoing",
			AttackerName: "Player1",
			DefenderName: "Target1",
			Code:         "Win",
		},
		{
			AttackID:     222,
			Started:      testTime.Add(30 * time.Minute),
			Direction:    "Incoming",
			AttackerName: "Enemy1",
			DefenderName: "Player2",
			Code:         "Loss",
		},
	}

	err := processor.UpdateAttackRecords(context.Background(), "test_spreadsheet", config, records)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that records were appended
	sheetData := mockAPI.GetSheetData("Records - 123")
	if len(sheetData) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(sheetData))
	}

	// Verify first record data
	if len(sheetData) > 0 {
		row := sheetData[0]
		if len(row) > 4 && row[4] != "Player1" {
			t.Errorf("Expected attacker 'Player1', got %v", row[4])
		}
	}
}

func TestAttackRecordsProcessorUpdateAttackRecordsEmpty(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	config := &app.SheetConfig{
		WarID:          123,
		RecordsTabName: "Records - 123",
	}
	records := []app.AttackRecord{}

	err := processor.UpdateAttackRecords(context.Background(), "test_spreadsheet", config, records)
	if err != nil {
		t.Fatalf("Expected no error for empty records, got %v", err)
	}

	// Should not have written any data
	sheetData := mockAPI.GetSheetData("Records - 123")
	if len(sheetData) > 0 {
		t.Error("Expected no data to be written for empty records")
	}
}

func TestAttackRecordsProcessorWithAPIError(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	mockAPI.SetError(true)
	processor := NewAttackRecordsProcessor(mockAPI)

	// Test ReadExistingRecords with error
	_, err := processor.ReadExistingRecords(context.Background(), "test_spreadsheet", "test_sheet")
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test UpdateAttackRecords with error
	config := &app.SheetConfig{WarID: 123, RecordsTabName: "Records - 123"}
	records := []app.AttackRecord{{AttackID: 1, Started: time.Now()}}
	err = processor.UpdateAttackRecords(context.Background(), "test_spreadsheet", config, records)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}
}

func TestAttackRecordsProcessorEdgeCases(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	// Test with records that have zero timestamps
	records := []app.AttackRecord{
		{AttackID: 1, Started: time.Unix(0, 0)},
		{AttackID: 2, Started: time.Unix(1, 0)},
	}

	existing := &RecordsInfo{LastTimestamp: 0}
	filtered := processor.FilterAndSortRecords(records, existing)

	// Should keep records with timestamp > 0
	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered record, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].AttackID != 2 {
		t.Errorf("Expected attack ID 2, got %d", filtered[0].AttackID)
	}
}

func TestAttackRecordsProcessorTimeFormatting(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	// Test various time zones and formats
	testCases := []struct {
		timestamp    int64
		expectedDate string
		expectedTime string
	}{
		{1640995200, "2022-01-01", "00:00:00"}, // UTC midnight
		{1640995260, "2022-01-01", "00:01:00"}, // 1 minute later
		{1641081600, "2022-01-02", "00:00:00"}, // Next day
		{1641038400, "2022-01-01", "12:00:00"}, // Noon UTC
	}

	for _, tc := range testCases {
		record := app.AttackRecord{
			AttackID: 1,
			Started:  time.Unix(tc.timestamp, 0),
		}

		rows := processor.ConvertRecordsToRows([]app.AttackRecord{record})
		if len(rows) != 1 {
			t.Fatalf("Expected 1 row, got %d", len(rows))
		}

		row := rows[0]
		if len(row) < 3 {
			t.Fatalf("Expected at least 3 columns, got %d", len(row))
		}

		if row[1] != tc.expectedDate {
			t.Errorf("Timestamp %d: expected date '%s', got %v", tc.timestamp, tc.expectedDate, row[1])
		}

		if row[2] != tc.expectedTime {
			t.Errorf("Timestamp %d: expected time '%s', got %v", tc.timestamp, tc.expectedTime, row[2])
		}
	}
}
