package sheets

import (
	"context"
	"testing"

	"torn_rw_stats/internal/app"
)

func TestTravelStatusManagerEnsureTravelStatusSheet(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewTravelStatusManager(mockAPI)

	factionID := 12345
	expectedSheetName := "Status - 12345"

	sheetName, err := manager.EnsureTravelStatusSheet(context.Background(), "test_spreadsheet", factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sheetName != expectedSheetName {
		t.Errorf("Expected sheet name '%s', got '%s'", expectedSheetName, sheetName)
	}

	// Verify sheet was created
	if !mockAPI.sheets[expectedSheetName] {
		t.Error("Expected travel sheet to be created")
	}

	// Verify headers were written
	headerData := mockAPI.GetSheetData(expectedSheetName)
	if len(headerData) == 0 {
		t.Error("Expected headers to be written")
	}
}

func TestTravelStatusManagerEnsureTravelStatusSheetAlreadyExists(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewTravelStatusManager(mockAPI)

	factionID := 12345
	expectedSheetName := "Status - 12345"

	// Pre-create the sheet
	mockAPI.sheets[expectedSheetName] = true

	sheetName, err := manager.EnsureTravelStatusSheet(context.Background(), "test_spreadsheet", factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sheetName != expectedSheetName {
		t.Errorf("Expected sheet name '%s', got '%s'", expectedSheetName, sheetName)
	}

	// Should not have written headers since sheet already existed
	headerData := mockAPI.GetSheetData(expectedSheetName)
	if len(headerData) > 0 {
		t.Error("Expected no headers to be written for existing sheet")
	}
}

func TestTravelStatusManagerGenerateTravelSheetName(t *testing.T) {
	manager := NewTravelStatusManager(NewMockSheetsAPI())

	testCases := []struct {
		factionID    int
		expectedName string
	}{
		{123, "Status - 123"},
		{456, "Status - 456"},
		{0, "Status - 0"},
		{999999, "Status - 999999"},
	}

	for _, tc := range testCases {
		result := manager.GenerateTravelSheetName(tc.factionID)
		if result != tc.expectedName {
			t.Errorf("Expected sheet name '%s', got '%s'", tc.expectedName, result)
		}
	}
}

func TestTravelStatusManagerGenerateTravelStatusHeaders(t *testing.T) {
	manager := NewTravelStatusManager(NewMockSheetsAPI())

	headers := manager.GenerateTravelStatusHeaders()
	if len(headers) != 1 {
		t.Errorf("Expected 1 header row, got %d", len(headers))
	}

	headerRow := headers[0]
	expectedColumns := []string{
		"Player Name", "Level", "Status", "Location",
		"Countdown", "Departure", "Arrival",
	}

	if len(headerRow) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(headerRow))
	}

	for i, expected := range expectedColumns {
		if i < len(headerRow) && headerRow[i] != expected {
			t.Errorf("Expected column %d to be '%s', got '%v'", i, expected, headerRow[i])
		}
	}
}

func TestTravelStatusManagerUpdateTravelStatus(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewTravelStatusManager(mockAPI)

	sheetName := "Status - 123"
	records := []app.TravelRecord{
		{
			Name:      "TestPlayer1",
			Level:     50,
			State:     "Traveling",
			Location:  "Torn City",
			Countdown: "5 minutes",
		},
		{
			Name:      "TestPlayer2",
			Level:     75,
			State:     "Okay",
			Location:  "Japan",
			Countdown: "",
		},
	}

	err := manager.UpdateTravelStatus(context.Background(), "test_spreadsheet", sheetName, records)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify data was written
	sheetData := mockAPI.GetSheetData(sheetName)
	if len(sheetData) != len(records) {
		t.Errorf("Expected %d rows, got %d", len(records), len(sheetData))
	}

	// Check first record
	if len(sheetData) > 0 {
		row := sheetData[0]
		if len(row) != 7 {
			t.Errorf("Expected 7 columns, got %d", len(row))
		}

		if row[0] != "TestPlayer1" {
			t.Errorf("Expected player name 'TestPlayer1', got %v", row[0])
		}

		if row[1] != 50 {
			t.Errorf("Expected level 50, got %v", row[1])
		}

		if row[2] != "Traveling" {
			t.Errorf("Expected state 'Traveling', got %v", row[2])
		}
	}
}

func TestTravelStatusManagerUpdateTravelStatusEmpty(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	manager := NewTravelStatusManager(mockAPI)

	sheetName := "Status - 123"
	records := []app.TravelRecord{}

	err := manager.UpdateTravelStatus(context.Background(), "test_spreadsheet", sheetName, records)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should not have written any data
	sheetData := mockAPI.GetSheetData(sheetName)
	if len(sheetData) > 0 {
		t.Error("Expected no data to be written for empty records")
	}
}

func TestTravelStatusManagerConvertTravelRecordsToRows(t *testing.T) {
	manager := NewTravelStatusManager(NewMockSheetsAPI())

	records := []app.TravelRecord{
		{
			Name:      "Player1",
			Level:     25,
			State:     "Hospital",
			Location:  "Torn City",
			Countdown: "2 hours",
		},
		{
			Name:      "Player2",
			Level:     100,
			State:     "Okay",
			Location:  "Japan",
			Countdown: "",
		},
	}

	rows := manager.ConvertTravelRecordsToRows(records)

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Check first row
	row1 := rows[0]
	if len(row1) != 7 {
		t.Errorf("Expected 7 columns, got %d", len(row1))
	}

	expectedValues := []interface{}{
		"Player1",   // Player Name
		25,          // Level
		"Hospital",  // Status
		"Torn City", // Location
		"2 hours",   // Countdown
		"",          // Departure (empty for hospital)
		"",          // Arrival (empty for hospital)
	}

	for i, expected := range expectedValues {
		if i < len(row1) && row1[i] != expected {
			t.Errorf("Row 1, column %d: expected %v, got %v", i, expected, row1[i])
		}
	}

	// Check second row
	row2 := rows[1]
	if row2[0] != "Player2" || row2[1] != 100 || row2[2] != "Okay" {
		t.Error("Second row data doesn't match expected values")
	}
}

func TestTravelStatusManagerWithAPIError(t *testing.T) {
	mockAPI := NewMockSheetsAPI()
	mockAPI.SetError(true)
	manager := NewTravelStatusManager(mockAPI)

	_, err := manager.EnsureTravelStatusSheet(context.Background(), "test_spreadsheet", 123)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}

	// Test update with error
	records := []app.TravelRecord{{Name: "Test", Level: 1}}
	err = manager.UpdateTravelStatus(context.Background(), "test_spreadsheet", "Travel - 123", records)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}
}
