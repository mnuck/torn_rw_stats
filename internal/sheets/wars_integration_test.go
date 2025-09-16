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

// TestConvertMembersToStateRows tests member state conversion

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

}
