package processing

import (
	"context"
	"testing"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

func TestStateChangeDetectionServiceNormalizeHospitalDescription(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	tests := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "Generic hospital with countdown",
			description: "In hospital for 2hrs 15mins",
			expected:    "In hospital",
		},
		{
			name:        "South African hospital with countdown",
			description: "In a South African hospital for 33 mins",
			expected:    "In hospital",
		},
		{
			name:        "British hospital with countdown",
			description: "In a British hospital for 2hrs 30mins",
			expected:    "In hospital",
		},
		{
			name:        "Traveling status",
			description: "Traveling to Mexico",
			expected:    "Traveling to Mexico",
		},
		{
			name:        "Okay status",
			description: "Okay",
			expected:    "Okay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.NormalizeHospitalDescription(tt.description)
			if result != tt.expected {
				t.Errorf("NormalizeHospitalDescription(%q) = %q, expected %q",
					tt.description, result, tt.expected)
			}
		})
	}
}

func TestStateChangeDetectionServiceHasStatusChanged(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	tests := []struct {
		name         string
		oldMember    app.FactionMember
		newMember    app.FactionMember
		expectChange bool
	}{
		{
			name: "No change - identical members",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
					Color:       "green",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
					Color:       "green",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
			expectChange: false,
		},
		{
			name: "Hospital countdown change - should NOT trigger change",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a South African hospital for 33 mins",
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{
					Status: "Idle",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a South African hospital for 28 mins",
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{
					Status: "Idle",
				},
			},
			expectChange: false,
		},
		{
			name: "LastAction status change",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
				},
				LastAction: app.LastAction{
					Status: "Offline",
				},
			},
			expectChange: true,
		},
		{
			name: "Status state change",
			oldMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Okay",
					State:       "Okay",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
			newMember: app.FactionMember{
				Status: app.MemberStatus{
					Description: "Traveling to Mexico",
					State:       "Traveling",
				},
				LastAction: app.LastAction{
					Status: "Online",
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.HasStatusChanged(tt.oldMember, tt.newMember)
			if result != tt.expectChange {
				t.Errorf("HasStatusChanged() = %v, expected %v", result, tt.expectChange)
			}
		})
	}
}

func TestStateChangeDetectionServiceProcessStateChanges(t *testing.T) {
	mockSheets := mocks.NewMockSheetsClient()
	service := NewStateChangeDetectionService(mockSheets)

	// Setup mock responses
	mockSheets.EnsureStateChangeRecordsSheetResponse = "State Changes - 12345"

	oldMembers := map[string]app.FactionMember{
		"123": {
			Name:  "TestPlayer",
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
	}

	newMembers := map[string]app.FactionMember{
		"123": {
			Name:  "TestPlayer",
			Level: 50,
			Status: app.MemberStatus{
				Description: "Traveling to Mexico",
				State:       "Traveling",
				Color:       "blue",
			},
			LastAction: app.LastAction{
				Status: "Online",
			},
		},
	}

	ctx := context.Background()
	err := service.ProcessStateChanges(ctx, 12345, "Test Faction", oldMembers, newMembers, "test-spreadsheet")

	// Verify no error
	if err != nil {
		t.Fatalf("ProcessStateChanges failed: %v", err)
	}

	// Verify sheets client calls
	if !mockSheets.EnsureStateChangeRecordsSheetCalled {
		t.Error("Expected EnsureStateChangeRecordsSheet to be called")
	}

	if !mockSheets.AddStateChangeRecordCalled {
		t.Error("Expected AddStateChangeRecord to be called")
	}

	// Verify the state change record was created correctly
	if mockSheets.AddStateChangeRecordCalledWith.Record.MemberName != "TestPlayer" {
		t.Errorf("Expected member name 'TestPlayer', got %q",
			mockSheets.AddStateChangeRecordCalledWith.Record.MemberName)
	}

	if mockSheets.AddStateChangeRecordCalledWith.Record.StatusState != "Traveling" {
		t.Errorf("Expected status state 'Traveling', got %q",
			mockSheets.AddStateChangeRecordCalledWith.Record.StatusState)
	}
}