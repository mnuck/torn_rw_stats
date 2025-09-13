package processing

import (
	"testing"

	"torn_rw_stats/internal/app"
)

func TestWarSummaryServiceGenerateWarSummary(t *testing.T) {
	attackService := NewAttackProcessingService(12345)
	summaryService := NewWarSummaryService(attackService)

	// Create test war data
	war := &app.War{
		ID:    2001,
		Start: 1640995200, // 2022-01-01 00:00:00 UTC
		End:   nil,        // Active war
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction", Score: 150},
			{ID: 67890, Name: "Enemy Faction", Score: 120},
		},
	}

	// Create test attacks - mix of wins and losses
	attacks := []app.Attack{
		{
			ID: 1,
			Attacker: app.User{
				ID:      123,
				Faction: &app.Faction{ID: 12345}, // Our attack
			},
			Defender: app.User{
				ID:      456,
				Faction: &app.Faction{ID: 67890},
			},
			Result:      "Hospitalized", // Win
			RespectGain: 3.0,
			RespectLoss: 0.0,
		},
		{
			ID: 2,
			Attacker: app.User{
				ID:      789,
				Faction: &app.Faction{ID: 67890}, // Enemy attack on us
			},
			Defender: app.User{
				ID:      321,
				Faction: &app.Faction{ID: 12345},
			},
			Result:      "Escape", // We defended successfully
			RespectGain: 1.5,      // Enemy gained
			RespectLoss: 0.5,      // Enemy lost
		},
	}

	summary := summaryService.GenerateWarSummary(war, attacks)

	// Verify basic info
	if summary.WarID != 2001 {
		t.Errorf("Expected WarID 2001, got %d", summary.WarID)
	}
	if summary.Status != "Active" {
		t.Errorf("Expected Status 'Active', got %q", summary.Status)
	}
	if summary.WarName != "Our Faction vs Enemy Faction" {
		t.Errorf("Expected WarName 'Our Faction vs Enemy Faction', got %q", summary.WarName)
	}

	// Verify attack statistics
	if summary.TotalAttacks != 2 {
		t.Errorf("Expected TotalAttacks 2, got %d", summary.TotalAttacks)
	}
	if summary.AttacksWon != 2 {
		t.Errorf("Expected AttacksWon 2, got %d", summary.AttacksWon)
	}
	if summary.AttacksLost != 0 {
		t.Errorf("Expected AttacksLost 0, got %d", summary.AttacksLost)
	}

	// Check respect calculations
	expectedRespectGain := 3.5 // 3.0 from our attack + 0.5 from defending
	if summary.RespectGained != expectedRespectGain {
		t.Errorf("Expected RespectGained %f, got %f", expectedRespectGain, summary.RespectGained)
	}

	expectedRespectLost := 1.5 // 1.5 from enemy attack
	if summary.RespectLost != expectedRespectLost {
		t.Errorf("Expected RespectLost %f, got %f", expectedRespectLost, summary.RespectLost)
	}

	// Verify factions are set correctly
	if summary.OurFaction.ID != 12345 {
		t.Errorf("Expected OurFaction.ID 12345, got %d", summary.OurFaction.ID)
	}
	if summary.EnemyFaction.ID != 67890 {
		t.Errorf("Expected EnemyFaction.ID 67890, got %d", summary.EnemyFaction.ID)
	}
}

func TestWarSummaryServiceIsSuccessfulAttack(t *testing.T) {
	attackService := NewAttackProcessingService(12345)
	summaryService := NewWarSummaryService(attackService)

	tests := []struct {
		name     string
		result   string
		expected bool
	}{
		{"Hospitalized result", "Hospitalized", true},
		{"Mugged result", "Mugged", true},
		{"Left result", "Left", true},
		{"Escape result", "Escape", false},
		{"Stalemate result", "Stalemate", false},
		{"Unknown result", "Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summaryService.isSuccessfulAttack(tt.result)
			if result != tt.expected {
				t.Errorf("Expected isSuccessfulAttack(%q) = %v, got %v", tt.result, tt.expected, result)
			}
		})
	}
}

func TestWarSummaryServiceIsSuccessfulDefense(t *testing.T) {
	attackService := NewAttackProcessingService(12345)
	summaryService := NewWarSummaryService(attackService)

	tests := []struct {
		name     string
		result   string
		expected bool
	}{
		{"Stalemate result", "Stalemate", true},
		{"Escape result", "Escape", true},
		{"Assist result", "Assist", true},
		{"Hospitalized result", "Hospitalized", false},
		{"Mugged result", "Mugged", false},
		{"Unknown result", "Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summaryService.isSuccessfulDefense(tt.result)
			if result != tt.expected {
				t.Errorf("Expected isSuccessfulDefense(%q) = %v, got %v", tt.result, tt.expected, result)
			}
		})
	}
}
