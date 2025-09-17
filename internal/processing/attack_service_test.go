package processing

import (
	"testing"

	"torn_rw_stats/internal/app"
)

func TestAttackProcessingServiceProcessAttacksIntoRecords(t *testing.T) {
	service := NewAttackProcessingService()

	// Create test war data
	war := &app.War{
		ID: 1001,
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction"},
			{ID: 67890, Name: "Enemy Faction"},
		},
	}

	// Create test attacks
	attacks := []app.Attack{
		{
			ID:      100001,
			Code:    "1234abcd",
			Started: 1640995200, // 2022-01-01 00:00:00 UTC
			Ended:   1640995260, // 2022-01-01 00:01:00 UTC
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
			Modifiers: app.AttackModifiers{
				FairFight: 1.0,
				War:       2.0,
			},
			FinishingHitEffects: []app.FinishingHitEffect{
				{Name: "Critical Hit", Value: 1.5},
			},
		},
	}

	records := service.ProcessAttacksIntoRecords(attacks, war, 12345)

	// Verify we got the expected number of records
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	record := records[0]

	// Verify key fields
	if record.AttackID != 100001 {
		t.Errorf("Expected AttackID 100001, got %d", record.AttackID)
	}
	if record.Code != "1234abcd" {
		t.Errorf("Expected Code '1234abcd', got %q", record.Code)
	}
	if record.AttackerName != "TestAttacker" {
		t.Errorf("Expected AttackerName 'TestAttacker', got %q", record.AttackerName)
	}
	if record.DefenderName != "TestDefender" {
		t.Errorf("Expected DefenderName 'TestDefender', got %q", record.DefenderName)
	}
	if record.Direction != "Outgoing" {
		t.Errorf("Expected Direction 'Outgoing', got %q", record.Direction)
	}
	if record.RespectGain != 2.5 {
		t.Errorf("Expected RespectGain 2.5, got %f", record.RespectGain)
	}
	if record.FinishingHitName != "Critical Hit" {
		t.Errorf("Expected FinishingHitName 'Critical Hit', got %q", record.FinishingHitName)
	}
}

func TestAttackProcessingServiceDetermineAttackDirection(t *testing.T) {
	service := NewAttackProcessingService()

	tests := []struct {
		name         string
		attack       app.Attack
		ourFactionID int
		expected     string
	}{
		{
			name: "Outgoing attack",
			attack: app.Attack{
				Attacker: app.User{
					Faction: &app.Faction{ID: 12345},
				},
				Defender: app.User{
					Faction: &app.Faction{ID: 67890},
				},
			},
			ourFactionID: 12345,
			expected:     "Outgoing",
		},
		{
			name: "Incoming attack",
			attack: app.Attack{
				Attacker: app.User{
					Faction: &app.Faction{ID: 67890},
				},
				Defender: app.User{
					Faction: &app.Faction{ID: 12345},
				},
			},
			ourFactionID: 12345,
			expected:     "Incoming",
		},
		{
			name: "Unknown attack",
			attack: app.Attack{
				Attacker: app.User{
					Faction: &app.Faction{ID: 99999},
				},
				Defender: app.User{
					Faction: &app.Faction{ID: 88888},
				},
			},
			ourFactionID: 12345,
			expected:     "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineAttackDirection(tt.attack, tt.ourFactionID)
			if result != tt.expected {
				t.Errorf("Expected direction %q, got %q", tt.expected, result)
			}
		})
	}
}
