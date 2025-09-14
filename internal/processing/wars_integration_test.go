package processing

import (
	"context"
	"errors"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
	"torn_rw_stats/internal/sheets"
)

// TestMockTornClient demonstrates torn client mock usage
func TestMockTornClient(t *testing.T) {
	mock := mocks.NewMockTornClient()

	// Setup response
	mock.FactionBasicResponse = &app.FactionBasicResponse{
		Members: map[string]app.FactionMember{
			"123": {
				Name:  "TestMember",
				Level: 50,
			},
		},
	}

	// Call method
	ctx := context.Background()
	result, err := mock.GetFactionBasic(ctx, 12345)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !mock.GetFactionBasicCalled {
		t.Error("Expected GetFactionBasic to be called")
	}

	if mock.GetFactionBasicCalledWithID != 12345 {
		t.Errorf("Expected faction ID 12345, got %d", mock.GetFactionBasicCalledWithID)
	}

	if len(result.Members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(result.Members))
	}
}

// TestMockSheetsClient demonstrates sheets client mock usage
func TestMockSheetsClient(t *testing.T) {
	mock := mocks.NewMockSheetsClient()

	// Setup response
	mock.EnsureWarSheetsResponse = &app.SheetConfig{
		SummaryTabName: "Summary - 1001",
		RecordsTabName: "Records - 1001",
	}

	// Call method
	ctx := context.Background()
	testWar := &app.War{ID: 1001}
	result, err := mock.EnsureWarSheets(ctx, "test-spreadsheet", testWar)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !mock.EnsureWarSheetsCalled {
		t.Error("Expected EnsureWarSheets to be called")
	}

	if mock.EnsureWarSheetsCalledWith.SpreadsheetID != "test-spreadsheet" {
		t.Errorf("Expected spreadsheet ID 'test-spreadsheet', got %q",
			mock.EnsureWarSheetsCalledWith.SpreadsheetID)
	}

	if result.SummaryTabName != "Summary - 1001" {
		t.Errorf("Expected summary tab name 'Summary - 1001', got %q", result.SummaryTabName)
	}
}

// TestMockErrorHandling demonstrates error handling with mocks
func TestMockErrorHandling(t *testing.T) {
	mock := mocks.NewMockTornClient()

	// Setup error
	mock.FactionBasicError = errors.New("API error")

	// Call method
	ctx := context.Background()
	result, err := mock.GetFactionBasic(ctx, 12345)

	// Verify error handling
	if err == nil {
		t.Error("Expected error but got none")
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}

	if err.Error() != "API error" {
		t.Errorf("Expected error 'API error', got %q", err.Error())
	}

	// Verify call tracking still works
	if !mock.GetFactionBasicCalled {
		t.Error("Expected GetFactionBasic to be called even on error")
	}
}

// TestMockReset demonstrates resetting mock state
func TestMockReset(t *testing.T) {
	mock := mocks.NewMockTornClient()

	// Setup initial state
	mock.GetFactionBasicCalled = true
	mock.GetFactionBasicCalledWithID = 999
	mock.FactionBasicResponse = &app.FactionBasicResponse{}

	// Reset
	mock.Reset()

	// Verify reset
	if mock.GetFactionBasicCalled {
		t.Error("Expected GetFactionBasicCalled to be false after reset")
	}

	if mock.GetFactionBasicCalledWithID != 0 {
		t.Error("Expected GetFactionBasicCalledWithID to be 0 after reset")
	}

	if mock.FactionBasicResponse != nil {
		t.Error("Expected FactionBasicResponse to be nil after reset")
	}
}

// TestProcessWarIntegrationWithMocks demonstrates full integration testing using mocks
func TestProcessWarIntegrationWithMocks(t *testing.T) {
	// Setup mocks
	mockTornClient := mocks.NewMockTornClient()
	mockSheetsClient := mocks.NewMockSheetsClient()

	// Create WarProcessor with mocks using new interface-based constructor
	wp := newTestWarProcessorWithMocks(
		mockTornClient,
		mockSheetsClient,
		&app.Config{SpreadsheetID: "test-spreadsheet"},
	)
	wp.ourFactionID = 12345

	// Setup test data
	testWar := &app.War{
		ID:    1001,
		Start: time.Now().Unix() - 3600, // 1 hour ago
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction", Score: 150},
			{ID: 67890, Name: "Enemy Faction", Score: 120},
		},
	}

	testAttacks := []app.Attack{
		{
			ID:      100001,
			Code:    "attack1abc",
			Started: time.Now().Unix() - 1800, // 30 minutes ago
			Ended:   time.Now().Unix() - 1740, // 29 minutes ago
			Attacker: app.User{
				ID:    123,
				Name:  "TestAttacker",
				Level: 50,
				Faction: &app.Faction{
					ID:   12345,
					Name: "Our Faction",
				},
			},
			Defender: app.User{
				ID:    456,
				Name:  "TestDefender",
				Level: 45,
				Faction: &app.Faction{
					ID:   67890,
					Name: "Enemy Faction",
				},
			},
			Result:      "Hospitalized",
			RespectGain: 2.5,
			RespectLoss: 0.0,
			Chain:       10,
		},
	}

	// Configure mock responses
	mockSheetsClient.EnsureWarSheetsResponse = &app.SheetConfig{
		SummaryTabName: "Summary - 1001",
		RecordsTabName: "Records - 1001",
	}

	mockSheetsClient.ReadExistingRecordsResponse = &sheets.ExistingRecordsInfo{
		AttackCodes:     make(map[string]bool),
		RecordCount:     0,
		LatestTimestamp: 0,
	}

	mockTornClient.AllAttacksForWarResponse = testAttacks

	// Setup mocks for travel status processing (which runs after attack processing)
	mockSheetsClient.EnsureTravelStatusSheetResponse = "Travel - 12345"
	mockSheetsClient.ReadSheetResponse = [][]interface{}{} // Empty existing travel data

	mockTornClient.FactionBasicResponse = &app.FactionBasicResponse{
		Members: map[string]app.FactionMember{
			"123": {
				Name:  "TestMember",
				Level: 50,
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
					Color:       "green",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
		},
	}

	mockSheetsClient.EnsurePreviousStateSheetResponse = "Previous State - 12345"
	mockSheetsClient.LoadPreviousMemberStatesResponse = map[string]app.FactionMember{} // No previous states

	// Do the same setup for enemy faction travel status
	mockSheetsClient.EnsureStateChangeRecordsSheetResponse = "State Changes - 67890"

	// Execute the method under test
	ctx := context.Background()
	err := wp.processWar(ctx, testWar)

	// Assertions
	if err != nil {
		t.Fatalf("processWar failed: %v", err)
	}

	// Verify torn client calls
	if !mockTornClient.GetAllAttacksForWarCalled {
		t.Error("Expected GetAllAttacksForWar to be called")
	}

	if mockTornClient.GetAllAttacksForWarCalledWith != testWar {
		t.Error("GetAllAttacksForWar called with wrong war")
	}

	// Verify sheets client calls
	expectedSheetsCalls := []struct {
		called bool
		name   string
	}{
		{mockSheetsClient.EnsureWarSheetsCalled, "EnsureWarSheets"},
		{mockSheetsClient.ReadExistingRecordsCalled, "ReadExistingRecords"},
		{mockSheetsClient.UpdateWarSummaryCalled, "UpdateWarSummary"},
		{mockSheetsClient.UpdateAttackRecordsCalled, "UpdateAttackRecords"},
	}

	for _, call := range expectedSheetsCalls {
		if !call.called {
			t.Errorf("Expected %s to be called", call.name)
		}
	}

	// Verify parameter passing
	if mockSheetsClient.EnsureWarSheetsCalledWith.SpreadsheetID != "test-spreadsheet" {
		t.Errorf("Expected spreadsheet ID 'test-spreadsheet', got %q",
			mockSheetsClient.EnsureWarSheetsCalledWith.SpreadsheetID)
	}

	// Verify attack records processing
	if len(mockSheetsClient.UpdateAttackRecordsCalledWith.Records) != 1 {
		t.Errorf("Expected 1 attack record, got %d",
			len(mockSheetsClient.UpdateAttackRecordsCalledWith.Records))
	}

	record := mockSheetsClient.UpdateAttackRecordsCalledWith.Records[0]
	if record.AttackID != 100001 {
		t.Errorf("Expected attack ID 100001, got %d", record.AttackID)
	}

	if record.Direction != "Outgoing" {
		t.Errorf("Expected direction 'Outgoing', got %q", record.Direction)
	}

	// Verify war summary generation
	summary := mockSheetsClient.UpdateWarSummaryCalledWith.Summary
	if summary.WarID != 1001 {
		t.Errorf("Expected war ID 1001, got %d", summary.WarID)
	}

	if summary.TotalAttacks != 1 {
		t.Errorf("Expected 1 total attack, got %d", summary.TotalAttacks)
	}

	if summary.AttacksWon != 1 {
		t.Errorf("Expected 1 attack won, got %d", summary.AttacksWon)
	}

	if summary.RespectGained != 2.5 {
		t.Errorf("Expected 2.5 respect gained, got %f", summary.RespectGained)
	}
}
