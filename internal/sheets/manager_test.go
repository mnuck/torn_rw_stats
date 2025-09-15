package sheets

import (
	"context"
	"strings"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// MockSheetsAPI implements SheetsAPI for testing
type MockSheetsAPI struct {
	sheets          map[string]bool            // Track which sheets exist
	data            map[string][][]interface{} // Store sheet data
	shouldError     bool
	lastReadRange   string
	lastUpdateRange string
	lastUpdateData  [][]interface{}
}

func NewMockSheetsAPI() *MockSheetsAPI {
	return &MockSheetsAPI{
		sheets: make(map[string]bool),
		data:   make(map[string][][]interface{}),
	}
}

func (m *MockSheetsAPI) ReadSheet(ctx context.Context, spreadsheetID, range_ string) ([][]interface{}, error) {
	if m.shouldError {
		return nil, &mockError{msg: "mock read error"}
	}
	m.lastReadRange = range_

	// Extract sheet name from range (before the '!')
	sheetName := range_
	if exclamationIndex := strings.Index(range_, "!"); exclamationIndex != -1 {
		sheetName = range_[:exclamationIndex]
	}

	// Remove quotes if present
	sheetName = strings.Trim(sheetName, "'\"")

	if data, exists := m.data[sheetName]; exists {
		return data, nil
	}
	return [][]interface{}{}, nil
}

func (m *MockSheetsAPI) UpdateRange(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	if m.shouldError {
		return &mockError{msg: "mock update error"}
	}
	m.lastUpdateRange = range_
	m.lastUpdateData = values

	// Extract sheet name and store data
	sheetName := range_
	if exclamationIndex := strings.Index(range_, "!"); exclamationIndex != -1 {
		sheetName = range_[:exclamationIndex]
	}
	sheetName = strings.Trim(sheetName, "'\"")
	m.data[sheetName] = values
	return nil
}

func (m *MockSheetsAPI) ClearRange(ctx context.Context, spreadsheetID, range_ string) error {
	if m.shouldError {
		return &mockError{msg: "mock clear error"}
	}
	// Simple implementation - just remove data for testing
	sheetName := range_
	if exclamationIndex := strings.Index(range_, "!"); exclamationIndex != -1 {
		sheetName = range_[:exclamationIndex]
	}
	sheetName = strings.Trim(sheetName, "'\"")
	delete(m.data, sheetName)
	return nil
}

func (m *MockSheetsAPI) AppendRows(ctx context.Context, spreadsheetID, range_ string, rows [][]interface{}) error {
	if m.shouldError {
		return &mockError{msg: "mock append error"}
	}

	// Extract sheet name and append data
	sheetName := range_
	if exclamationIndex := strings.Index(range_, "!"); exclamationIndex != -1 {
		sheetName = range_[:exclamationIndex]
	}
	sheetName = strings.Trim(sheetName, "'\"")

	if existing, exists := m.data[sheetName]; exists {
		m.data[sheetName] = append(existing, rows...)
	} else {
		m.data[sheetName] = rows
	}
	return nil
}

func (m *MockSheetsAPI) CreateSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	if m.shouldError {
		return &mockError{msg: "mock create error"}
	}
	m.sheets[sheetName] = true
	return nil
}

func (m *MockSheetsAPI) SheetExists(ctx context.Context, spreadsheetID, sheetName string) (bool, error) {
	if m.shouldError {
		return false, &mockError{msg: "mock exists error"}
	}
	return m.sheets[sheetName], nil
}

func (m *MockSheetsAPI) EnsureSheetCapacity(ctx context.Context, spreadsheetID, sheetName string, requiredRows, requiredCols int) error {
	if m.shouldError {
		return &mockError{msg: "mock capacity error"}
	}
	// For testing, just mark that the sheet exists
	m.sheets[sheetName] = true
	return nil
}

func (m *MockSheetsAPI) FormatStatusSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	if m.shouldError {
		return &mockError{msg: "mock format error"}
	}
	// For testing, this is a no-op
	return nil
}

func (m *MockSheetsAPI) SetError(shouldError bool) {
	m.shouldError = shouldError
}

func (m *MockSheetsAPI) GetSheetData(sheetName string) [][]interface{} {
	return m.data[sheetName]
}

func (m *MockSheetsAPI) SetSheetData(sheetName string, data [][]interface{}) {
	m.data[sheetName] = data
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Test WarSheetsManager

func TestWarSheetsManagerEnsureWarSheets(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewWarSheetsManager(mockAPI)

	war := &app.War{
		ID: 123,
	}

	config, err := manager.EnsureWarSheets(context.Background(), "test_spreadsheet", war)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.WarID != 123 {
		t.Errorf("Expected WarID 123, got %d", config.WarID)
	}

	expectedSummaryTab := "Summary - 123"
	expectedRecordsTab := "Records - 123"

	if config.SummaryTabName != expectedSummaryTab {
		t.Errorf("Expected summary tab '%s', got '%s'", expectedSummaryTab, config.SummaryTabName)
	}

	if config.RecordsTabName != expectedRecordsTab {
		t.Errorf("Expected records tab '%s', got '%s'", expectedRecordsTab, config.RecordsTabName)
	}

	// Verify sheets were created
	if !mockAPI.sheets[expectedSummaryTab] {
		t.Error("Expected summary sheet to be created")
	}

	if !mockAPI.sheets[expectedRecordsTab] {
		t.Error("Expected records sheet to be created")
	}
}

func TestWarSheetsManagerGenerateTabNames(t *testing.T) {
	manager := NewWarSheetsManager(NewMockSheetsAPI())

	testCases := []struct {
		warID           int
		expectedSummary string
		expectedRecords string
	}{
		{123, "Summary - 123", "Records - 123"},
		{456, "Summary - 456", "Records - 456"},
		{0, "Summary - 0", "Records - 0"},
	}

	for _, tc := range testCases {
		summaryTab := manager.GenerateSummaryTabName(tc.warID)
		recordsTab := manager.GenerateRecordsTabName(tc.warID)

		if summaryTab != tc.expectedSummary {
			t.Errorf("Expected summary tab '%s', got '%s'", tc.expectedSummary, summaryTab)
		}

		if recordsTab != tc.expectedRecords {
			t.Errorf("Expected records tab '%s', got '%s'", tc.expectedRecords, recordsTab)
		}
	}
}

func TestWarSheetsManagerGenerateHeaders(t *testing.T) {
	manager := NewWarSheetsManager(NewMockSheetsAPI())

	summaryHeaders := manager.GenerateSummarySheetHeaders()
	if len(summaryHeaders) == 0 {
		t.Error("Expected summary headers to be generated")
	}

	// Check that first row is title
	if len(summaryHeaders[0]) == 0 || summaryHeaders[0][0] != "War Summary" {
		t.Error("Expected first header row to contain 'War Summary'")
	}

	recordsHeaders := manager.GenerateRecordsSheetHeaders()
	if len(recordsHeaders) == 0 || len(recordsHeaders[0]) == 0 {
		t.Error("Expected records headers to be generated")
	}

	// Check for key columns
	headerRow := recordsHeaders[0]
	expectedCols := []string{"Timestamp", "Date", "Time", "Direction", "Attacker", "Defender"}
	for _, expectedCol := range expectedCols {
		found := false
		for _, actualCol := range headerRow {
			if actualCol == expectedCol {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find column '%s' in records headers", expectedCol)
		}
	}
}

func TestWarSheetsManagerWithAPIError(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	mockAPI.SetError(true)
	manager := NewWarSheetsManager(mockAPI)

	war := &app.War{ID: 123}

	_, err := manager.EnsureWarSheets(context.Background(), "test_spreadsheet", war)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}
}

// Test AttackRecordsProcessor

func TestAttackRecordsProcessorReadExistingRecords(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	// Set up mock data with attack records (ID, Code, Started timestamp)
	mockAPI.SetSheetData("test_sheet", [][]interface{}{
		{100001, "attack_code_1", "2024-01-01 10:16:40", "2024-01-01 10:17:40"},
		{100002, "attack_code_2", "2024-01-01 10:33:20", "2024-01-01 10:34:20"},
		{100003, "attack_code_3", "2024-01-01 10:25:00", "2024-01-01 10:26:00"},
	})

	info, err := processor.ReadExistingRecords(context.Background(), "test_spreadsheet", "test_sheet")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Expected: 2024-01-01 10:33:20 Unix timestamp
	expectedTimestamp := int64(1704105200) // 2024-01-01 10:33:20 UTC
	if info.LatestTimestamp != expectedTimestamp {
		t.Errorf("Expected last timestamp %d, got %d", expectedTimestamp, info.LatestTimestamp)
	}

	if info.RecordCount != 3 {
		t.Errorf("Expected record count 3, got %d", info.RecordCount)
	}

	// Check attack codes
	if len(info.AttackCodes) != 3 {
		t.Errorf("Expected 3 attack codes, got %d", len(info.AttackCodes))
	}
	if !info.AttackCodes["attack_code_1"] || !info.AttackCodes["attack_code_2"] || !info.AttackCodes["attack_code_3"] {
		t.Error("Expected all attack codes to be present in map")
	}
}

func TestAttackRecordsProcessorFilterAndSortRecords(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	records := []app.AttackRecord{
		{AttackID: 1, Code: "new_code_1", Started: time.Unix(1000, 0)},
		{AttackID: 2, Code: "existing_code", Started: time.Unix(500, 0)}, // Should be filtered out (duplicate)
		{AttackID: 3, Code: "new_code_2", Started: time.Unix(1500, 0)},
		{AttackID: 4, Code: "existing_code", Started: time.Unix(750, 0)}, // Should be filtered out (duplicate)
	}

	existing := &RecordsInfo{
		AttackCodes: map[string]bool{
			"existing_code": true, // This code already exists
		},
		LatestTimestamp: 0,
		RecordCount:     1,
	}

	filtered := processor.FilterAndSortRecords(records, existing)

	// Should have 2 records (new_code_1 and new_code_2), duplicates filtered out
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered records, got %d", len(filtered))
	}

	// Should be sorted by timestamp (oldest first: 1000, then 1500)
	if len(filtered) >= 2 {
		if filtered[0].Started.Unix() != 1000 {
			t.Errorf("Expected first record timestamp 1000, got %d", filtered[0].Started.Unix())
		}
		if filtered[1].Started.Unix() != 1500 {
			t.Errorf("Expected second record timestamp 1500, got %d", filtered[1].Started.Unix())
		}
		if filtered[0].Code != "new_code_1" {
			t.Errorf("Expected first record code 'new_code_1', got %s", filtered[0].Code)
		}
		if filtered[1].Code != "new_code_2" {
			t.Errorf("Expected second record code 'new_code_2', got %s", filtered[1].Code)
		}
	}
}

func TestAttackRecordsProcessorConvertRecordsToRows(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	processor := NewAttackRecordsProcessor(mockAPI)

	records := []app.AttackRecord{
		{
			AttackID:            123,
			Started:             time.Unix(1000, 0),
			Direction:           "Outgoing",
			AttackerName:        "TestAttacker",
			AttackerFactionName: "AttackerFaction",
			DefenderName:        "TestDefender",
			DefenderFactionName: "DefenderFaction",
			Code:                "Win",
		},
	}

	rows := processor.ConvertRecordsToRows(records)

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if len(row) != 32 {
		t.Fatalf("Expected 32 columns, got %d", len(row))
	}

	// Check key fields in new format
	if row[0] != int64(123) { // AttackID
		t.Errorf("Expected AttackID 123, got %v", row[0])
	}

	if row[1] != "Win" { // Code
		t.Errorf("Expected code 'Win', got %v", row[1])
	}

	if row[4] != "Outgoing" { // Direction
		t.Errorf("Expected direction 'Outgoing', got %v", row[4])
	}

	if row[6] != "TestAttacker" { // AttackerName
		t.Errorf("Expected attacker 'TestAttacker', got %v", row[6])
	}
}

// Test ParseValue functions

func TestParseStringValue(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{123, "123"},
		{12.34, "12.34"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := parseStringValue(tc.input)
		if result != tc.expected {
			t.Errorf("parseStringValue(%v): expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestParseIntValue(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected int
	}{
		{nil, 0},
		{"123", 123},
		{123, 123},
		{123.0, 123},
		{int64(456), 456},
		{"invalid", 0},
		{"", 0},
	}

	for _, tc := range testCases {
		result := parseIntValue(tc.input)
		if result != tc.expected {
			t.Errorf("parseIntValue(%v): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestParseInt64Value(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected int64
	}{
		{nil, 0},
		{"123", 123},
		{123, 123},
		{123.0, 123},
		{int64(456), 456},
		{"invalid", 0},
	}

	for _, tc := range testCases {
		result := parseInt64Value(tc.input)
		if result != tc.expected {
			t.Errorf("parseInt64Value(%v): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestParseInt64PointerValue(t *testing.T) {
	result := parseInt64PointerValue(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}

	result = parseInt64PointerValue("123")
	if result == nil || *result != 123 {
		t.Errorf("Expected *123, got %v", result)
	}

	result = parseInt64PointerValue(0)
	if result != nil {
		t.Error("Expected nil for 0 input")
	}
}
