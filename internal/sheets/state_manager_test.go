package sheets

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

func TestStateChangeManagerEnsureStateChangeRecordsSheet(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	factionID := 12345
	expectedSheetName := "State Changes - 12345"

	sheetName, err := manager.EnsureStateChangeRecordsSheet(context.Background(), "test_spreadsheet", factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sheetName != expectedSheetName {
		t.Errorf("Expected sheet name '%s', got '%s'", expectedSheetName, sheetName)
	}

	// Verify sheet was created
	if !mockAPI.sheets[expectedSheetName] {
		t.Error("Expected state change sheet to be created")
	}

	// Verify headers were written
	headerData := mockAPI.GetSheetData(expectedSheetName)
	if len(headerData) == 0 {
		t.Error("Expected headers to be written")
	}
}

func TestStateChangeManagerEnsurePreviousStateSheet(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	factionID := 12345
	expectedSheetName := "Previous States - 12345"

	sheetName, err := manager.EnsurePreviousStateSheet(context.Background(), "test_spreadsheet", factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sheetName != expectedSheetName {
		t.Errorf("Expected sheet name '%s', got '%s'", expectedSheetName, sheetName)
	}

	// Verify sheet was created
	if !mockAPI.sheets[expectedSheetName] {
		t.Error("Expected previous state sheet to be created")
	}
}

func TestStateChangeManagerGenerateSheetNames(t *testing.T) {
	manager := NewStateChangeManager(NewMockSheetsAPI())

	testCases := []struct {
		factionID             int
		expectedStateChanges  string
		expectedPreviousState string
	}{
		{123, "State Changes - 123", "Previous States - 123"},
		{456, "State Changes - 456", "Previous States - 456"},
		{0, "State Changes - 0", "Previous States - 0"},
	}

	for _, tc := range testCases {
		stateChangesName := manager.GenerateStateChangeSheetName(tc.factionID)
		previousStateName := manager.GeneratePreviousStateSheetName(tc.factionID)

		if stateChangesName != tc.expectedStateChanges {
			t.Errorf("Expected state changes sheet name '%s', got '%s'", tc.expectedStateChanges, stateChangesName)
		}

		if previousStateName != tc.expectedPreviousState {
			t.Errorf("Expected previous state sheet name '%s', got '%s'", tc.expectedPreviousState, previousStateName)
		}
	}
}

func TestStateChangeManagerGenerateHeaders(t *testing.T) {
	manager := NewStateChangeManager(NewMockSheetsAPI())

	// Test state change headers
	stateChangeHeaders := manager.GenerateStateChangeHeaders()
	if len(stateChangeHeaders) != 1 {
		t.Errorf("Expected 1 header row for state changes, got %d", len(stateChangeHeaders))
	}

	expectedStateChangeColumns := []string{
		"Timestamp", "Date", "Time", "Player ID", "Player Name",
		"Change Type", "Old Status", "New Status", "Description",
	}

	headerRow := stateChangeHeaders[0]
	if len(headerRow) != len(expectedStateChangeColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedStateChangeColumns), len(headerRow))
	}

	for i, expected := range expectedStateChangeColumns {
		if i < len(headerRow) && headerRow[i] != expected {
			t.Errorf("Expected column %d to be '%s', got '%v'", i, expected, headerRow[i])
		}
	}

	// Test previous state headers
	previousStateHeaders := manager.GeneratePreviousStateHeaders()
	if len(previousStateHeaders) != 1 {
		t.Errorf("Expected 1 header row for previous states, got %d", len(previousStateHeaders))
	}

	expectedPreviousStateColumns := []string{
		"Player ID", "Player Name", "Level", "Status", "Last Action",
		"Until", "Description", "Location", "Last Updated",
	}

	prevHeaderRow := previousStateHeaders[0]
	if len(prevHeaderRow) != len(expectedPreviousStateColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedPreviousStateColumns), len(prevHeaderRow))
	}
}

func TestStateChangeManagerAddStateChangeRecord(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	sheetName := "State Changes - 123"
	timestamp := time.Unix(1640995200, 0) // 2022-01-01 00:00:00 UTC

	record := app.StateChangeRecord{
		Timestamp:         timestamp,
		MemberID:          12345,
		MemberName:        "TestPlayer",
		OldState:          "Okay",
		NewState:          "Hospital",
		StatusDescription: "Hospitalized for 30 minutes",
	}

	err := manager.AddStateChangeRecord(context.Background(), "test_spreadsheet", sheetName, record)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify data was written
	sheetData := mockAPI.GetSheetData(sheetName)
	if len(sheetData) != 1 {
		t.Errorf("Expected 1 row, got %d", len(sheetData))
	}

	if len(sheetData) > 0 {
		row := sheetData[0]
		if len(row) < 9 {
			t.Errorf("Expected at least 9 columns, got %d", len(row))
		}

		// Check key fields
		if row[0] != int64(1640995200) { // Timestamp
			t.Errorf("Expected timestamp 1640995200, got %v", row[0])
		}

		if row[1] != "2022-01-01" { // Date
			t.Errorf("Expected date '2022-01-01', got %v", row[1])
		}

		if row[2] != "00:00:00" { // Time
			t.Errorf("Expected time '00:00:00', got %v", row[2])
		}

		if row[3] != 12345 { // Player ID
			t.Errorf("Expected player ID 12345, got %v", row[3])
		}

		if row[4] != "TestPlayer" { // Player Name
			t.Errorf("Expected player name 'TestPlayer', got %v", row[4])
		}

		if row[5] != "State Change" { // Change Type
			t.Errorf("Expected change type 'State Change', got %v", row[5])
		}

		if row[6] != "Okay" { // Old Status
			t.Errorf("Expected old status 'Okay', got %v", row[6])
		}

		if row[7] != "Hospital" { // New Status
			t.Errorf("Expected new status 'Hospital', got %v", row[7])
		}

		if row[8] != "Hospitalized for 30 minutes" { // Description
			t.Errorf("Expected description 'Hospitalized for 30 minutes', got %v", row[8])
		}
	}
}

func TestStateChangeManagerConvertStateChangeToRow(t *testing.T) {
	manager := NewStateChangeManager(NewMockSheetsAPI())

	timestamp := time.Unix(1640995200, 0) // 2022-01-01 00:00:00 UTC
	record := app.StateChangeRecord{
		Timestamp:         timestamp,
		MemberID:          67890,
		MemberName:        "AnotherPlayer",
		OldState:          "Hospital",
		NewState:          "Okay",
		StatusDescription: "Recovered from hospital",
	}

	row := manager.ConvertStateChangeToRow(record)

	expectedValues := []interface{}{
		int64(1640995200),         // Timestamp
		"2022-01-01",              // Date
		"00:00:00",                // Time
		67890,                     // Player ID
		"AnotherPlayer",           // Player Name
		"State Change",            // Change Type
		"Hospital",                // Old Status
		"Okay",                    // New Status
		"Recovered from hospital", // Description
	}

	if len(row) != len(expectedValues) {
		t.Errorf("Expected %d columns, got %d", len(expectedValues), len(row))
	}

	for i, expected := range expectedValues {
		if i < len(row) && row[i] != expected {
			t.Errorf("Column %d: expected %v, got %v", i, expected, row[i])
		}
	}
}

func TestStateChangeManagerStorePreviousMemberStates(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	sheetName := "Previous States - 123"
	untilTimestamp := int64(1640995200)

	members := map[string]app.FactionMember{
		"Player1": {
			Name:  "Player1",
			Level: 50,
			Status: app.MemberStatus{
				Description: "Okay",
				Details:     "Active player",
				Until:       &untilTimestamp,
			},
			Position: "Torn City",
			LastAction: app.LastAction{
				Timestamp: 1640995100, // Earlier timestamp
			},
		},
		"Player2": {
			Name:  "Player2",
			Level: 75,
			Status: app.MemberStatus{
				Description: "Hospital",
				Details:     "Injured",
				Until:       nil,
			},
			Position: "Japan",
			LastAction: app.LastAction{
				Timestamp: 1640995150,
			},
		},
	}

	err := manager.StorePreviousMemberStates(context.Background(), "test_spreadsheet", sheetName, members)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify data was written
	sheetData := mockAPI.GetSheetData(sheetName)
	if len(sheetData) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(sheetData))
	}

	// Check that data contains expected player names
	playerNames := make(map[string]bool)
	for _, row := range sheetData {
		if len(row) > 1 {
			if name, ok := row[1].(string); ok {
				playerNames[name] = true
			}
		}
	}

	if !playerNames["Player1"] {
		t.Error("Expected Player1 to be in stored data")
	}

	if !playerNames["Player2"] {
		t.Error("Expected Player2 to be in stored data")
	}
}

func TestStateChangeManagerLoadPreviousMemberStates(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	sheetName := "Previous States - 123"

	// Set up mock data
	mockData := [][]interface{}{
		{0, "Player1", 50, "Okay", int64(1640995100), int64(1640995200), "Active", "Torn City", "2022-01-01 00:00:00"},
		{0, "Player2", 75, "Hospital", int64(1640995150), nil, "Injured", "Japan", "2022-01-01 00:00:00"},
		{}, // Empty row to test filtering
	}
	mockAPI.SetSheetData(sheetName, mockData)

	members, err := manager.LoadPreviousMemberStates(context.Background(), "test_spreadsheet", sheetName)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// Check Player1
	if player1, exists := members["Player1"]; exists {
		if player1.Name != "Player1" {
			t.Errorf("Expected name 'Player1', got '%s'", player1.Name)
		}
		if player1.Level != 50 {
			t.Errorf("Expected level 50, got %d", player1.Level)
		}
		if player1.Position != "Torn City" {
			t.Errorf("Expected position 'Torn City', got '%s'", player1.Position)
		}
	} else {
		t.Error("Expected Player1 to be loaded")
	}

	// Check Player2
	if player2, exists := members["Player2"]; exists {
		if player2.Name != "Player2" {
			t.Errorf("Expected name 'Player2', got '%s'", player2.Name)
		}
		if player2.Level != 75 {
			t.Errorf("Expected level 75, got %d", player2.Level)
		}
		if player2.Position != "Japan" {
			t.Errorf("Expected position 'Japan', got '%s'", player2.Position)
		}
	} else {
		t.Error("Expected Player2 to be loaded")
	}
}

func TestStateChangeManagerConvertMembersToStateRows(t *testing.T) {
	manager := NewStateChangeManager(NewMockSheetsAPI())

	untilTimestamp := int64(1640995200)
	members := map[string]app.FactionMember{
		"TestPlayer": {
			Name:  "TestPlayer",
			Level: 60,
			Status: app.MemberStatus{
				Description: "Hospital",
				Details:     "Recovering",
				Until:       &untilTimestamp,
			},
			Position: "Switzerland",
			LastAction: app.LastAction{
				Timestamp: 1640995100,
			},
		},
	}

	rows := manager.ConvertMembersToStateRows(members)

	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}

	if len(rows) > 0 {
		row := rows[0]
		if len(row) != 9 {
			t.Errorf("Expected 9 columns, got %d", len(row))
		}

		// Check key fields
		if row[1] != "TestPlayer" {
			t.Errorf("Expected player name 'TestPlayer', got %v", row[1])
		}

		if row[2] != 60 {
			t.Errorf("Expected level 60, got %v", row[2])
		}

		if row[3] != "Hospital" {
			t.Errorf("Expected status 'Hospital', got %v", row[3])
		}

		if row[4] != int64(1640995100) {
			t.Errorf("Expected last action timestamp 1640995100, got %v", row[4])
		}

		if row[5] != int64(1640995200) {
			t.Errorf("Expected until timestamp 1640995200, got %v", row[5])
		}

		if row[6] != "Recovering" {
			t.Errorf("Expected details 'Recovering', got %v", row[6])
		}

		if row[7] != "Switzerland" {
			t.Errorf("Expected position 'Switzerland', got %v", row[7])
		}
	}
}

func TestStateChangeManagerWithAPIErrors(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	mockAPI.SetError(true)
	manager := NewStateChangeManager(mockAPI)

	// Test EnsureStateChangeRecordsSheet with error
	_, err := manager.EnsureStateChangeRecordsSheet(context.Background(), "test_spreadsheet", 123)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test EnsurePreviousStateSheet with error
	_, err = manager.EnsurePreviousStateSheet(context.Background(), "test_spreadsheet", 123)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test AddStateChangeRecord with error
	record := app.StateChangeRecord{
		Timestamp:  time.Now(),
		MemberID:   123,
		MemberName: "Test",
		OldState:   "Okay",
		NewState:   "Hospital",
	}
	err = manager.AddStateChangeRecord(context.Background(), "test_spreadsheet", "State Changes - 123", record)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test StorePreviousMemberStates with error
	members := map[string]app.FactionMember{"Test": {Name: "Test", Level: 1}}
	err = manager.StorePreviousMemberStates(context.Background(), "test_spreadsheet", "Previous States - 123", members)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test LoadPreviousMemberStates with error
	_, err = manager.LoadPreviousMemberStates(context.Background(), "test_spreadsheet", "Previous States - 123")
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}
}

func TestStateChangeManagerEmptyData(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewStateChangeManager(mockAPI)

	// Test StorePreviousMemberStates with empty map
	err := manager.StorePreviousMemberStates(context.Background(), "test_spreadsheet", "Previous States - 123", map[string]app.FactionMember{})
	if err != nil {
		t.Fatalf("Expected no error for empty members, got %v", err)
	}

	// Should not have written any data
	sheetData := mockAPI.GetSheetData("Previous States - 123")
	if len(sheetData) > 0 {
		t.Error("Expected no data to be written for empty members")
	}
}
