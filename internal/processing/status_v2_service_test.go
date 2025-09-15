package processing

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// TestParseStateRecordFromRow tests the parsing of Changed States sheet rows
func TestParseStateRecordFromRow(t *testing.T) {
	service := &StatusV2Service{}

	testCases := []struct {
		name     string
		row      []interface{}
		expected app.StateRecord
	}{
		{
			name: "complete row with travel type",
			row: []interface{}{
				"2022-01-01 00:00:00",        // Timestamp
				"2022-01-01",                 // Date
				"00:00:00",                   // Time
				"12345",                      // Member ID
				"TestPlayer",                 // Member Name
				"1001",                       // Faction ID
				"Test Faction",               // Faction Name
				"Online",                     // Last Action Status
				"Traveling to Mexico",        // Status Description
				"Traveling",                  // Status State
				"2022-01-01 00:26:00",       // Status Until
				"airstrip",                   // Status Travel Type
			},
			expected: app.StateRecord{
				Timestamp:         time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				MemberID:          "12345",
				MemberName:        "TestPlayer",
				FactionID:         "1001",
				FactionName:       "Test Faction",
				LastActionStatus:  "Online",
				StatusDescription: "Traveling to Mexico",
				StatusState:       "Traveling",
				StatusUntil:       time.Date(2022, 1, 1, 0, 26, 0, 0, time.UTC),
				StatusTravelType:  "airstrip",
			},
		},
		{
			name: "row without travel type",
			row: []interface{}{
				"2022-01-01 00:00:00",        // Timestamp
				"2022-01-01",                 // Date
				"00:00:00",                   // Time
				"12346",                      // Member ID
				"TestPlayer2",                // Member Name
				"1001",                       // Faction ID
				"Test Faction",               // Faction Name
				"Online",                     // Last Action Status
				"Okay",                       // Status Description
				"Okay",                       // Status State
				"",                           // Status Until (empty)
			},
			expected: app.StateRecord{
				Timestamp:         time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				MemberID:          "12346",
				MemberName:        "TestPlayer2",
				FactionID:         "1001",
				FactionName:       "Test Faction",
				LastActionStatus:  "Online",
				StatusDescription: "Okay",
				StatusState:       "Okay",
				StatusUntil:       time.Time{}, // Zero time
				StatusTravelType:  "",          // Empty
			},
		},
		{
			name: "row with regular travel type",
			row: []interface{}{
				"2022-01-01 00:00:00",        // Timestamp
				"2022-01-01",                 // Date
				"00:00:00",                   // Time
				"12347",                      // Member ID
				"TestPlayer3",                // Member Name
				"1001",                       // Faction ID
				"Test Faction",               // Faction Name
				"Online",                     // Last Action Status
				"Traveling to UK",            // Status Description
				"Traveling",                  // Status State
				"",                           // Status Until (empty)
				"regular",                    // Status Travel Type
			},
			expected: app.StateRecord{
				Timestamp:         time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				MemberID:          "12347",
				MemberName:        "TestPlayer3",
				FactionID:         "1001",
				FactionName:       "Test Faction",
				LastActionStatus:  "Online",
				StatusDescription: "Traveling to UK",
				StatusState:       "Traveling",
				StatusUntil:       time.Time{}, // Zero time
				StatusTravelType:  "regular",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record, err := service.parseStateRecordFromRow(tc.row)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// Compare all fields
			if record.Timestamp != tc.expected.Timestamp {
				t.Errorf("Expected timestamp %v, got %v", tc.expected.Timestamp, record.Timestamp)
			}
			if record.MemberID != tc.expected.MemberID {
				t.Errorf("Expected MemberID %s, got %s", tc.expected.MemberID, record.MemberID)
			}
			if record.MemberName != tc.expected.MemberName {
				t.Errorf("Expected MemberName %s, got %s", tc.expected.MemberName, record.MemberName)
			}
			if record.FactionID != tc.expected.FactionID {
				t.Errorf("Expected FactionID %s, got %s", tc.expected.FactionID, record.FactionID)
			}
			if record.FactionName != tc.expected.FactionName {
				t.Errorf("Expected FactionName %s, got %s", tc.expected.FactionName, record.FactionName)
			}
			if record.LastActionStatus != tc.expected.LastActionStatus {
				t.Errorf("Expected LastActionStatus %s, got %s", tc.expected.LastActionStatus, record.LastActionStatus)
			}
			if record.StatusDescription != tc.expected.StatusDescription {
				t.Errorf("Expected StatusDescription %s, got %s", tc.expected.StatusDescription, record.StatusDescription)
			}
			if record.StatusState != tc.expected.StatusState {
				t.Errorf("Expected StatusState %s, got %s", tc.expected.StatusState, record.StatusState)
			}
			if !record.StatusUntil.Equal(tc.expected.StatusUntil) {
				t.Errorf("Expected StatusUntil %v, got %v", tc.expected.StatusUntil, record.StatusUntil)
			}
			if record.StatusTravelType != tc.expected.StatusTravelType {
				t.Errorf("Expected StatusTravelType %s, got %s", tc.expected.StatusTravelType, record.StatusTravelType)
			}
		})
	}
}

// TestTravelSpeedBugFix verifies that the travel speed is properly preserved
func TestTravelSpeedBugFix(t *testing.T) {
	// Test that airstrip travel type is preserved and not overwritten with empty string
	row := []interface{}{
		"2022-01-01 00:00:00",        // Timestamp
		"2022-01-01",                 // Date
		"00:00:00",                   // Time
		"12345",                      // Member ID
		"TestPlayer",                 // Member Name
		"1001",                       // Faction ID
		"Test Faction",               // Faction Name
		"Online",                     // Last Action Status
		"Traveling to Mexico",        // Status Description
		"Traveling",                  // Status State
		"2022-01-01 00:26:00",       // Status Until
		"airstrip",                   // Status Travel Type - this should be preserved!
	}

	service := &StatusV2Service{}
	record, err := service.parseStateRecordFromRow(row)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The critical test: StatusTravelType should be "airstrip", not empty string
	if record.StatusTravelType != "airstrip" {
		t.Errorf("Expected StatusTravelType 'airstrip', got '%s' - travel speed bug not fixed!", record.StatusTravelType)
	}

	// Test regular travel too
	row[11] = "regular"
	record, err = service.parseStateRecordFromRow(row)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if record.StatusTravelType != "regular" {
		t.Errorf("Expected StatusTravelType 'regular', got '%s'", record.StatusTravelType)
	}
}