package sheets

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// TestConvertRecordsToRows tests the record conversion logic directly
func TestConvertRecordsToRows(t *testing.T) {
	client := &Client{}

	records := []app.AttackRecord{
		{
			AttackID:            100001,
			Code:                "test_code_1",
			Started:             time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Ended:               time.Date(2024, 1, 15, 12, 1, 0, 0, time.UTC),
			Direction:           "Outgoing",
			AttackerID:          123,
			AttackerName:        "TestAttacker",
			AttackerLevel:       50,
			AttackerFactionID:   intPtr(1001),
			AttackerFactionName: "Our Faction",
			DefenderID:          456,
			DefenderName:        "TestDefender",
			DefenderLevel:       45,
			DefenderFactionID:   intPtr(1002),
			DefenderFactionName: "Enemy Faction",
			Result:              "Hospitalized",
			RespectGain:         2.75,
			RespectLoss:         0.25,
			Chain:               10,
			IsInterrupted:       false,
			IsStealthed:         false,
			IsRaid:              false,
			IsRankedWar:         true,
			ModifierFairFight:   1.0,
			ModifierWar:         2.0,
			ModifierRetaliation: 0.0,
			ModifierGroup:       0.0,
			ModifierOverseas:    0.0,
			ModifierChain:       1.5,
			ModifierWarlord:     0.0,
			FinishingHitName:    "Critical Hit",
			FinishingHitValue:   1.2,
		},
	}

	processor := NewAttackRecordsProcessor(client)
	rows := processor.ConvertRecordsToRows(records)

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if len(row) != 32 {
		t.Fatalf("Expected 32 columns, got %d", len(row))
	}

	// Test specific values
	if row[0] != int64(100001) {
		t.Errorf("Expected AttackID 100001, got %v", row[0])
	}
	if row[1] != "test_code_1" {
		t.Errorf("Expected Code 'test_code_1', got %v", row[1])
	}
	if row[2] != "2024-01-15 12:00:00" {
		t.Errorf("Expected Started '2024-01-15 12:00:00', got %v", row[2])
	}
	if row[4] != "Outgoing" {
		t.Errorf("Expected Direction 'Outgoing', got %v", row[4])
	}
	if row[8] != 1001 {
		t.Errorf("Expected AttackerFactionID 1001, got %v", row[8])
	}
	if row[16] != "2.75" {
		t.Errorf("Expected RespectGain '2.75', got %v", row[16])
	}
	if row[30] != "Critical Hit" {
		t.Errorf("Expected FinishingHitName 'Critical Hit', got %v", row[30])
	}
}

// TestConvertRecordsToRowsWithNilFactionIDs tests handling of nil faction IDs
func TestConvertRecordsToRowsWithNilFactionIDs(t *testing.T) {
	client := &Client{}

	records := []app.AttackRecord{
		{
			AttackID:          100001,
			Code:              "test_code",
			Started:           time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Ended:             time.Date(2024, 1, 15, 12, 1, 0, 0, time.UTC),
			AttackerFactionID: nil, // Nil faction ID
			DefenderFactionID: nil, // Nil faction ID
			RespectGain:       1.5,
			RespectLoss:       0.0,
		},
	}

	processor := NewAttackRecordsProcessor(client)
	rows := processor.ConvertRecordsToRows(records)

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row := rows[0]

	// Check that nil faction IDs are converted to empty strings
	if row[8] != "" {
		t.Errorf("Expected empty string for nil AttackerFactionID, got %v", row[8])
	}
	if row[13] != "" {
		t.Errorf("Expected empty string for nil DefenderFactionID, got %v", row[13])
	}
}

// TestConvertTravelRecordsToRows tests travel record conversion
func TestConvertTravelRecordsToRows(t *testing.T) {
	client := &Client{}

	records := []app.TravelRecord{
		{
			Name:      "TestPlayer1",
			Level:     50,
			Location:  "Mexico",
			State:     "Traveling",
			Departure: "2024-01-15 12:00:00",
			Countdown: "01:30:00",
			Arrival:   "2024-01-15 12:26:00",
		},
		{
			Name:      "TestPlayer2",
			Level:     35,
			Location:  "Torn",
			State:     "Okay",
			Departure: "",
			Countdown: "",
			Arrival:   "",
		},
	}

	rows := client.convertTravelRecordsToRows(records)

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Test first row (traveling player)
	row1 := rows[0]
	if len(row1) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(row1))
	}
	if row1[0] != "TestPlayer1" { // Player Name
		t.Errorf("Expected Name 'TestPlayer1', got %v", row1[0])
	}
	if row1[1] != 50 { // Level
		t.Errorf("Expected Level 50, got %v", row1[1])
	}
	if row1[2] != "Traveling" { // Status
		t.Errorf("Expected State 'Traveling', got %v", row1[2])
	}
	if row1[3] != "Mexico" { // Location
		t.Errorf("Expected Location 'Mexico', got %v", row1[3])
	}
	if row1[4] != "01:30:00" { // Countdown
		t.Errorf("Expected Countdown '01:30:00', got %v", row1[4])
	}
	if row1[5] != "2024-01-15 12:00:00" { // Departure
		t.Errorf("Expected Departure '2024-01-15 12:00:00', got %v", row1[5])
	}
	if row1[6] != "2024-01-15 12:26:00" { // Arrival
		t.Errorf("Expected Arrival '2024-01-15 12:26:00', got %v", row1[6])
	}

	// Test second row (non-traveling player)
	row2 := rows[1]
	if row2[0] != "TestPlayer2" { // Player Name
		t.Errorf("Expected Name 'TestPlayer2', got %v", row2[0])
	}
	if row2[2] != "Okay" { // Status
		t.Errorf("Expected State 'Okay', got %v", row2[2])
	}
	// Empty fields should be empty strings
	if row2[4] != "" { // Countdown
		t.Errorf("Expected empty Countdown, got %v", row2[4])
	}
	if row2[5] != "" { // Departure
		t.Errorf("Expected empty Departure, got %v", row2[5])
	}
	if row2[6] != "" { // Arrival
		t.Errorf("Expected empty Arrival, got %v", row2[6])
	}
}

// TestConvertMembersToStateRows tests member state conversion
func TestConvertMembersToStateRows(t *testing.T) {
	stateManager := &StateChangeManager{}

	members := map[string]app.FactionMember{
		"12345": {
			Name:     "TestMember1",
			Level:    45,
			Position: "Member",
			LastAction: app.LastAction{
				Status:    "Online",
				Timestamp: 1640995200,
				Relative:  "2 minutes ago",
			},
			Status: app.MemberStatus{
				Description: "Okay",
				State:       "Okay",
				Color:       "green",
				Details:     "",
				Until:       nil,
			},
		},
		"12346": {
			Name:     "TestMember2",
			Level:    60,
			Position: "Officer",
			LastAction: app.LastAction{
				Status:    "Offline",
				Timestamp: 1640991600,
				Relative:  "1 hour ago",
			},
			Status: app.MemberStatus{
				Description: "In hospital for 30mins",
				State:       "Hospital",
				Color:       "red",
				Details:     "Hospitalized",
				Until:       int64Ptr(1640997000),
			},
		},
	}

	rows := stateManager.ConvertMembersToStateRows(members)

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Verify both members are present (order not guaranteed due to map iteration)
	memberFound := make(map[string]bool)
	for _, row := range rows {
		if len(row) != 9 {
			t.Errorf("Expected 9 columns, got %d", len(row))
			continue
		}

		memberName := row[1].(string)
		memberFound[memberName] = true

		if memberName == "TestMember1" {
			if row[2] != 45 {
				t.Errorf("Expected level 45, got %v", row[2])
			}
			if row[3] != "Okay" {
				t.Errorf("Expected status description 'Okay', got %v", row[3])
			}
			if row[4] != int64(1640995200) {
				t.Errorf("Expected last action timestamp 1640995200, got %v", row[4])
			}
			if row[5] != nil {
				t.Errorf("Expected nil Until field, got %T: %v", row[5], row[5])
			}
			if row[7] != "Member" {
				t.Errorf("Expected position 'Member', got %v", row[7])
			}
		} else if memberName == "TestMember2" {
			if row[2] != 60 {
				t.Errorf("Expected level 60, got %v", row[2])
			}
			if row[3] != "In hospital for 30mins" {
				t.Errorf("Expected status description 'In hospital for 30mins', got %v", row[3])
			}
			if row[4] != int64(1640991600) {
				t.Errorf("Expected last action timestamp 1640991600, got %v", row[4])
			}
			if untilValue, ok := row[5].(int64); !ok || untilValue != int64(1640997000) {
				t.Errorf("Expected Until timestamp 1640997000, got %T: %v", row[5], row[5])
			}
			if row[7] != "Officer" {
				t.Errorf("Expected position 'Officer', got %v", row[7])
			}
		}
	}

	if !memberFound["TestMember1"] {
		t.Error("TestMember1 not found in converted rows")
	}
	if !memberFound["TestMember2"] {
		t.Error("TestMember2 not found in converted rows")
	}
}

// TestFilterAndSortRecords tests the record filtering and sorting logic
func TestFilterAndSortRecords(t *testing.T) {
	client := &Client{}

	records := []app.AttackRecord{
		{AttackID: 100003, Code: "code_3", Started: time.Date(2024, 1, 15, 12, 10, 0, 0, time.UTC)},
		{AttackID: 100001, Code: "code_1", Started: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)},
		{AttackID: 100002, Code: "code_2", Started: time.Date(2024, 1, 15, 12, 5, 0, 0, time.UTC)},
		{AttackID: 100004, Code: "code_1", Started: time.Date(2024, 1, 15, 12, 15, 0, 0, time.UTC)}, // Duplicate code
	}

	existing := &ExistingRecordsInfo{
		AttackCodes: map[string]bool{
			"code_1": true, // Already exists
		},
		LatestTimestamp: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix(),
		RecordCount:     1,
	}

	processor := NewAttackRecordsProcessor(client)
	filtered := processor.FilterAndSortRecords(records, existing)

	// Should have 2 records (code_2 and code_3) - code_1 duplicates filtered out
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 filtered records, got %d", len(filtered))
	}

	// Should be sorted chronologically (oldest first)
	if filtered[0].Code != "code_2" {
		t.Errorf("Expected first record to be code_2, got %s", filtered[0].Code)
	}
	if filtered[1].Code != "code_3" {
		t.Errorf("Expected second record to be code_3, got %s", filtered[1].Code)
	}

	// Verify chronological order
	if !filtered[0].Started.Before(filtered[1].Started) {
		t.Error("Records should be sorted chronologically (oldest first)")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// TestEmptyRecordHandling tests handling of empty record sets
func TestEmptyRecordHandling(t *testing.T) {
	client := &Client{}

	t.Run("empty attack records", func(t *testing.T) {
		records := []app.AttackRecord{}
		processor := NewAttackRecordsProcessor(client)
		rows := processor.ConvertRecordsToRows(records)
		if len(rows) != 0 {
			t.Errorf("Expected 0 rows for empty records, got %d", len(rows))
		}
	})

	t.Run("empty travel records", func(t *testing.T) {
		records := []app.TravelRecord{}
		rows := client.convertTravelRecordsToRows(records)
		if len(rows) != 0 {
			t.Errorf("Expected 0 rows for empty records, got %d", len(rows))
		}
	})

	t.Run("empty member map", func(t *testing.T) {
		members := make(map[string]app.FactionMember)
		rows := client.convertMembersToStateRows(members)
		if len(rows) != 0 {
			t.Errorf("Expected 0 rows for empty members, got %d", len(rows))
		}
	})
}
