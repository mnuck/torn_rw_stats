package services

import (
	"context"
	"errors"
	"testing"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

// factionBasicWithMember builds a minimal FactionBasicResponse for use in tests.
func factionBasicWithMember(factionID int, memberID, memberName, statusState, statusDescription string) *app.FactionBasicResponse {
	return &app.FactionBasicResponse{
		ID:   factionID,
		Name: "TestFaction",
		Members: map[string]app.FactionMember{
			memberID: {
				Name: memberName,
				Status: app.MemberStatus{
					State:       statusState,
					Description: statusDescription,
				},
				LastAction: app.LastAction{Status: "Online"},
			},
		},
	}
}

func TestStateTrackingService_BigQueryCalledWhenClientNonNil(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	sheetsMock := mocks.NewMockSheetsClient()
	sheetsMock.SheetExistsResponse = true // Changed States sheet already exists
	// ReadSheetResponse nil → no previous records → member is "new" → change detected

	bqMock := mocks.NewMockBigQueryClient()

	svc := NewStateTrackingServiceWithBigQuery(tornMock, sheetsMock, bqMock)
	if err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{100}); err != nil {
		t.Fatalf("ProcessStateChanges() returned unexpected error: %v", err)
	}

	if !bqMock.InsertStateRecordsCalled {
		t.Error("expected BigQuery InsertStateRecords to be called, but it was not")
	}
	if len(bqMock.InsertStateRecordsCalledWith) == 0 {
		t.Error("expected BigQuery InsertStateRecords to be called with records, but got none")
	}
}

func TestStateTrackingService_BigQuerySkippedWhenClientNil(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	sheetsMock := mocks.NewMockSheetsClient()
	sheetsMock.SheetExistsResponse = true

	// Use the constructor without BigQuery — must not panic
	svc := NewStateTrackingService(tornMock, sheetsMock)
	if err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{100}); err != nil {
		t.Fatalf("ProcessStateChanges() returned unexpected error: %v", err)
	}
	// No assertion needed beyond "did not panic"
}

func TestStateTrackingService_BigQueryFailureIsNonFatal(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	sheetsMock := mocks.NewMockSheetsClient()
	sheetsMock.SheetExistsResponse = true

	bqMock := mocks.NewMockBigQueryClient()
	bqMock.InsertStateRecordsError = errors.New("simulated BigQuery failure")

	svc := NewStateTrackingServiceWithBigQuery(tornMock, sheetsMock, bqMock)
	err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{100})
	if err != nil {
		t.Errorf("ProcessStateChanges() should succeed even when BigQuery fails, but got: %v", err)
	}
}

func TestStateTrackingService_BigQueryNotCalledWhenNoChanges(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	sheetsMock := mocks.NewMockSheetsClient()
	sheetsMock.SheetExistsResponse = true
	// Return a previous record for the same member with the same state → no change
	sheetsMock.ReadSheetResponse = [][]interface{}{
		{"2026-01-01 00:00:00", "42", "Player1", "100", "TestFaction", "Online", "Okay", "okay", "", ""},
	}

	bqMock := mocks.NewMockBigQueryClient()

	svc := NewStateTrackingServiceWithBigQuery(tornMock, sheetsMock, bqMock)
	if err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{100}); err != nil {
		t.Fatalf("ProcessStateChanges() returned unexpected error: %v", err)
	}

	if bqMock.InsertStateRecordsCalled {
		t.Error("expected BigQuery InsertStateRecords NOT to be called when there are no changes")
	}
}

func TestStateTrackingService_BigQueryNotCalledForEmptyFactions(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	sheetsMock := mocks.NewMockSheetsClient()
	sheetsMock.SheetExistsResponse = true
	bqMock := mocks.NewMockBigQueryClient()

	svc := NewStateTrackingServiceWithBigQuery(tornMock, sheetsMock, bqMock)
	// Pass empty faction list — GetFactionBasic should never be called
	if err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{}); err != nil {
		t.Fatalf("ProcessStateChanges() returned unexpected error: %v", err)
	}

	if bqMock.InsertStateRecordsCalled {
		t.Error("expected BigQuery InsertStateRecords NOT to be called for empty faction list")
	}
}
