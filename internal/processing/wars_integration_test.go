package processing

import (
	"context"
	"errors"
	"testing"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
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

// TODO: Create comprehensive integration tests once WarProcessor supports dependency injection with interfaces
// This would allow testing the full war processing flow with controlled mock responses