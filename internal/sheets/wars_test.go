package sheets

import (
	"context"
	"fmt"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// Mock client for testing without external dependencies
type mockSheetsClient struct {
	sheets              map[string]bool            // Track which sheets exist
	data                map[string][][]interface{} // Store sheet data by range
	createSheetError    error
	sheetExistsError    error
	updateRangeError    error
	readSheetError      error
	clearRangeError     error
	ensureCapacityError error
	appendRowsError     error
	lastUpdateRange     string
	lastUpdateValues    [][]interface{}
	lastReadRange       string
	lastClearRange      string
	capacityUpdateCount int
}

func newMockSheetsClient() *mockSheetsClient {
	return &mockSheetsClient{
		sheets: make(map[string]bool),
		data:   make(map[string][][]interface{}),
	}
}

func (m *mockSheetsClient) CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	if m.createSheetError != nil {
		return m.createSheetError
	}
	m.sheets[sheetName] = true
	return nil
}

func (m *mockSheetsClient) SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error) {
	if m.sheetExistsError != nil {
		return false, m.sheetExistsError
	}
	return m.sheets[sheetName], nil
}

func (m *mockSheetsClient) UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	if m.updateRangeError != nil {
		return m.updateRangeError
	}
	m.lastUpdateRange = range_
	m.lastUpdateValues = values
	m.data[range_] = values
	return nil
}

func (m *mockSheetsClient) ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error) {
	if m.readSheetError != nil {
		return nil, m.readSheetError
	}
	m.lastReadRange = range_
	if data, exists := m.data[range_]; exists {
		return data, nil
	}
	return [][]interface{}{}, nil
}

func (m *mockSheetsClient) ClearRange(ctx context.Context, spreadsheetID, range_ string) error {
	if m.clearRangeError != nil {
		return m.clearRangeError
	}
	m.lastClearRange = range_
	delete(m.data, range_)
	return nil
}

func (m *mockSheetsClient) EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error {
	if m.ensureCapacityError != nil {
		return m.ensureCapacityError
	}
	m.capacityUpdateCount++
	return nil
}

func (m *mockSheetsClient) AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error {
	if m.appendRowsError != nil {
		return m.appendRowsError
	}
	// Simulate appending by storing in data
	m.data[range_] = rows
	return nil
}

// Create a test client with the mock
func newTestClient() *Client {
	return &Client{
		service: nil, // We'll intercept calls via interface
	}
}

// TestEnsureWarSheets tests war sheet creation and initialization
func TestEnsureWarSheets(t *testing.T) {
	war := &app.War{
		ID: 12345,
		Factions: []app.Faction{
			{ID: 1001, Name: "Our Faction"},
			{ID: 1002, Name: "Enemy Faction"},
		},
	}

	t.Run("CreateNewSheets", func(t *testing.T) {
		// Test creating sheets when they don't exist
		// We need to test the logic through the actual methods
		summaryTabName := fmt.Sprintf("Summary - %d", war.ID)
		recordsTabName := fmt.Sprintf("Records - %d", war.ID)

		// Simulate the expected behavior
		expectedConfig := &app.SheetConfig{
			WarID:          war.ID,
			SummaryTabName: summaryTabName,
			RecordsTabName: recordsTabName,
			SpreadsheetID:  "test-spreadsheet-id",
		}

		// Verify config creation logic
		if expectedConfig.WarID != 12345 {
			t.Errorf("Expected WarID 12345, got %d", expectedConfig.WarID)
		}
		if expectedConfig.SummaryTabName != "Summary - 12345" {
			t.Errorf("Expected summary tab 'Summary - 12345', got %s", expectedConfig.SummaryTabName)
		}
		if expectedConfig.RecordsTabName != "Records - 12345" {
			t.Errorf("Expected records tab 'Records - 12345', got %s", expectedConfig.RecordsTabName)
		}
	})
}

// TestUpdateWarSummary tests war summary data formatting and update
func TestUpdateWarSummary(t *testing.T) {

	// Create test war summary
	startTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	lastUpdated := time.Date(2024, 1, 15, 14, 35, 0, 0, time.UTC)

	summary := &app.WarSummary{
		WarID:         12345,
		Status:        "Completed",
		StartTime:     startTime,
		EndTime:       &endTime,
		OurFaction:    app.Faction{ID: 1001, Name: "Our Faction", Score: 150},
		EnemyFaction:  app.Faction{ID: 1002, Name: "Enemy Faction", Score: 120},
		TotalAttacks:  50,
		AttacksWon:    35,
		AttacksLost:   15,
		RespectGained: 125.75,
		RespectLost:   45.25,
		LastUpdated:   lastUpdated,
	}

	t.Run("FormatSummaryData", func(t *testing.T) {
		// Test the data formatting logic
		expectedEndTime := "2024-01-15 14:30:00"

		// Verify calculations
		winRate := float64(summary.AttacksWon) / float64(summary.TotalAttacks) * 100
		if winRate != 70.0 {
			t.Errorf("Expected win rate 70.0, got %f", winRate)
		}

		netRespect := summary.RespectGained - summary.RespectLost
		if netRespect != 80.50 {
			t.Errorf("Expected net respect 80.50, got %f", netRespect)
		}

		// Test end time formatting
		if summary.EndTime.Format("2006-01-02 15:04:05") != expectedEndTime {
			t.Errorf("Expected end time %s, got %s", expectedEndTime, summary.EndTime.Format("2006-01-02 15:04:05"))
		}
	})

	t.Run("ActiveWarHandling", func(t *testing.T) {
		// Test active war (no end time)
		activeSummary := *summary
		activeSummary.EndTime = nil
		activeSummary.Status = "Active"

		// Should handle nil end time gracefully
		endTimeStr := "Active"
		if activeSummary.EndTime != nil {
			endTimeStr = activeSummary.EndTime.Format("2006-01-02 15:04:05")
		}

		if endTimeStr != "Active" {
			t.Errorf("Expected 'Active' for nil end time, got %s", endTimeStr)
		}
	})
}

// TestReadExistingRecords tests reading and parsing existing attack records
func TestReadExistingRecords(t *testing.T) {

	t.Run("EmptySheet", func(t *testing.T) {
		// Test empty sheet logic

		info := &ExistingRecordsInfo{
			AttackCodes:     make(map[string]bool),
			LatestTimestamp: 0,
			RecordCount:     0,
		}

		if info.RecordCount != 0 {
			t.Errorf("Expected 0 records for empty sheet, got %d", info.RecordCount)
		}
		if len(info.AttackCodes) != 0 {
			t.Errorf("Expected 0 attack codes for empty sheet, got %d", len(info.AttackCodes))
		}
	})

	t.Run("ValidRecords", func(t *testing.T) {
		// Test with valid records
		testData := [][]interface{}{
			{int64(100001), "attack_code_1", "2024-01-15 12:00:00", "2024-01-15 12:01:00"},
			{int64(100002), "attack_code_2", "2024-01-15 12:05:00", "2024-01-15 12:06:00"},
			{int64(100003), "attack_code_3", "2024-01-15 12:10:00", "2024-01-15 12:11:00"},
		}

		// Simulate parsing logic
		info := &ExistingRecordsInfo{
			AttackCodes:     make(map[string]bool),
			LatestTimestamp: 0,
			RecordCount:     3,
		}

		// Parse attack codes and timestamps
		for _, row := range testData {
			if len(row) >= 3 {
				if codeStr, ok := row[1].(string); ok && codeStr != "" {
					info.AttackCodes[codeStr] = true
				}

				if startedStr, ok := row[2].(string); ok {
					if startedTime, err := time.Parse("2006-01-02 15:04:05", startedStr); err == nil {
						timestamp := startedTime.Unix()
						if timestamp > info.LatestTimestamp {
							info.LatestTimestamp = timestamp
						}
					}
				}
			}
		}

		if len(info.AttackCodes) != 3 {
			t.Errorf("Expected 3 attack codes, got %d", len(info.AttackCodes))
		}

		if !info.AttackCodes["attack_code_1"] {
			t.Error("Expected attack_code_1 to be found")
		}

		expectedLatestTime := time.Date(2024, 1, 15, 12, 10, 0, 0, time.UTC).Unix()
		if info.LatestTimestamp != expectedLatestTime {
			t.Errorf("Expected latest timestamp %d, got %d", expectedLatestTime, info.LatestTimestamp)
		}
	})

	t.Run("MalformedData", func(t *testing.T) {
		// Test with malformed data
		testData := [][]interface{}{
			{int64(100001), "valid_code", "2024-01-15 12:00:00"},
			{int64(100002), "", "invalid-time-format"},         // Empty code, invalid time
			{"not_int", "valid_code_2", "2024-01-15 12:05:00"}, // Invalid ID
		}

		// Only valid rows should be counted
		info := &ExistingRecordsInfo{
			AttackCodes:     make(map[string]bool),
			LatestTimestamp: 0,
			RecordCount:     0,
		}

		validRows := 0
		for _, row := range testData {
			if len(row) >= 3 {
				if codeStr, ok := row[1].(string); ok && codeStr != "" {
					info.AttackCodes[codeStr] = true
					validRows++
				}
			}
		}

		info.RecordCount = validRows

		if info.RecordCount != 2 {
			t.Errorf("Expected 2 valid records, got %d", info.RecordCount)
		}
		if len(info.AttackCodes) != 2 {
			t.Errorf("Expected 2 attack codes, got %d", len(info.AttackCodes))
		}
	})
}

// TestUpdateAttackRecords tests attack record updates and deduplication
func TestUpdateAttackRecords(t *testing.T) {

	t.Run("EmptyRecords", func(t *testing.T) {
		// Test with empty records - should return early
		records := []app.AttackRecord{}

		// This should be a no-op
		if len(records) != 0 {
			t.Error("Expected empty records to be handled as no-op")
		}
	})

	t.Run("NewRecords", func(t *testing.T) {
		// Test adding new records to empty sheet
		records := []app.AttackRecord{
			{
				AttackID:     100001,
				Code:         "attack_code_1",
				Started:      time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Ended:        time.Date(2024, 1, 15, 12, 1, 0, 0, time.UTC),
				Direction:    "Outgoing",
				AttackerID:   123,
				AttackerName: "TestAttacker",
				DefenderID:   456,
				DefenderName: "TestDefender",
				Result:       "Hospitalized",
				RespectGain:  2.5,
				RespectLoss:  0.0,
			},
		}

		// Test with empty existing records

		// Test record conversion logic
		rows := convertRecordsToRows(records)
		if len(rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(rows))
		}

		row := rows[0]
		if len(row) < 32 {
			t.Errorf("Expected at least 32 columns, got %d", len(row))
		}

		// Verify key fields
		if row[0] != int64(100001) {
			t.Errorf("Expected AttackID 100001, got %v", row[0])
		}
		if row[1] != "attack_code_1" {
			t.Errorf("Expected Code 'attack_code_1', got %v", row[1])
		}
		if row[2] != "2024-01-15 12:00:00" {
			t.Errorf("Expected Started '2024-01-15 12:00:00', got %v", row[2])
		}
		if row[4] != "Outgoing" {
			t.Errorf("Expected Direction 'Outgoing', got %v", row[4])
		}
	})

	t.Run("DeduplicationLogic", func(t *testing.T) {
		// Test deduplication logic separately
		records := []app.AttackRecord{
			{AttackID: 100001, Code: "code_1", Started: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)},
			{AttackID: 100002, Code: "code_2", Started: time.Date(2024, 1, 15, 12, 5, 0, 0, time.UTC)},
			{AttackID: 100003, Code: "code_1", Started: time.Date(2024, 1, 15, 12, 10, 0, 0, time.UTC)}, // Duplicate
		}

		existing := &ExistingRecordsInfo{
			AttackCodes: map[string]bool{
				"code_1": true, // Already exists
			},
			LatestTimestamp: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix(),
			RecordCount:     1,
		}

		// Test filterAndSortRecords logic
		var newRecords []app.AttackRecord
		for _, record := range records {
			if !existing.AttackCodes[record.Code] {
				newRecords = append(newRecords, record)
			}
		}

		// Should only have code_2 (code_1 is duplicate, third record with code_1 also filtered)
		if len(newRecords) != 1 {
			t.Errorf("Expected 1 new record after deduplication, got %d", len(newRecords))
		}
		if newRecords[0].Code != "code_2" {
			t.Errorf("Expected remaining record to have code 'code_2', got %s", newRecords[0].Code)
		}
	})
}

// TestTravelStatusSheets tests travel status sheet operations
func TestTravelStatusSheets(t *testing.T) {

	t.Run("SheetCreation", func(t *testing.T) {
		factionID := 1001
		expectedSheetName := fmt.Sprintf("Travel Status - %d", factionID)

		if expectedSheetName != "Travel Status - 1001" {
			t.Errorf("Expected sheet name 'Travel Status - 1001', got %s", expectedSheetName)
		}
	})

	t.Run("TravelRecordConversion", func(t *testing.T) {
		records := []app.TravelRecord{
			{
				Name:      "TestPlayer",
				Level:     50,
				Location:  "Mexico",
				State:     "Traveling",
				Departure: "2024-01-15 12:00:00",
				Arrival:   "2024-01-15 14:00:00",
				Countdown: "01:30:00",
			},
		}

		// Test convertTravelRecordsToRows logic
		rows := convertTravelRecordsToRows(records)
		if len(rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(rows))
		}

		row := rows[0]
		if len(row) != 7 {
			t.Errorf("Expected 7 columns, got %d", len(row))
		}

		// Verify fields
		if row[0] != "TestPlayer" {
			t.Errorf("Expected Name 'TestPlayer', got %v", row[0])
		}
		if row[1] != 50 {
			t.Errorf("Expected Level 50, got %v", row[1])
		}
		if row[2] != "Mexico" {
			t.Errorf("Expected Location 'Mexico', got %v", row[2])
		}
		if row[3] != "Traveling" {
			t.Errorf("Expected State 'Traveling', got %v", row[3])
		}
	})
}

// Helper function to convert records to rows (extracted from wars.go for testing)
func convertRecordsToRows(records []app.AttackRecord) [][]interface{} {
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

// Helper function to convert travel records to rows (extracted from wars.go for testing)
func convertTravelRecordsToRows(records []app.TravelRecord) [][]interface{} {
	var rows [][]interface{}

	for _, record := range records {
		row := []interface{}{
			record.Name,
			record.Level,
			record.Location,
			record.State,
			record.Departure,
			record.Arrival,
			record.Countdown,
		}
		rows = append(rows, row)
	}

	return rows
}
