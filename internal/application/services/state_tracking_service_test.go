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

	bqMock := mocks.NewMockBigQueryClient()

	svc := NewStateTrackingService(tornMock, bqMock)
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

func TestStateTrackingService_BigQueryFailureIsFatal(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	bqMock := mocks.NewMockBigQueryClient()
	bqMock.InsertStateRecordsError = errors.New("simulated BigQuery failure")

	svc := NewStateTrackingService(tornMock, bqMock)
	err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{100})
	if err == nil {
		t.Error("ProcessStateChanges() should fail when BigQuery write fails")
	}
}

func TestStateTrackingService_BigQueryNotCalledWhenNoChanges(t *testing.T) {
	ctx := context.Background()

	tornMock := mocks.NewMockTornClient()
	tornMock.FactionBasicResponse = factionBasicWithMember(100, "42", "Player1", "okay", "Okay")

	bqMock := mocks.NewMockBigQueryClient()
	// Return a previous record for the same member with the same state → no change
	bqMock.QueryLatestStatePerMemberResponse = []app.StateRecord{
		{MemberID: "42", MemberName: "Player1", FactionID: "100", FactionName: "TestFaction",
			LastActionStatus: "Online", StatusDescription: "Okay", StatusState: "okay"},
	}

	svc := NewStateTrackingService(tornMock, bqMock)
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
	bqMock := mocks.NewMockBigQueryClient()

	svc := NewStateTrackingService(tornMock, bqMock)
	// Pass empty faction list — GetFactionBasic should never be called
	if err := svc.ProcessStateChanges(ctx, "spreadsheet-id", []int{}); err != nil {
		t.Fatalf("ProcessStateChanges() returned unexpected error: %v", err)
	}

	if bqMock.InsertStateRecordsCalled {
		t.Error("expected BigQuery InsertStateRecords NOT to be called for empty faction list")
	}
}
