package processing

import (
	"testing"

	"torn_rw_stats/internal/app"
)

// Test attack processing into records
func TestProcessAttacksIntoRecords(t *testing.T) {
	wp := newTestWarProcessor(&app.Config{})
	wp.ourFactionID = 12345

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

	records := wp.attackService.ProcessAttacksIntoRecords(attacks, war)

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

// Test war summary generation
func TestGenerateWarSummary(t *testing.T) {
	wp := newTestWarProcessor(&app.Config{})
	wp.ourFactionID = 12345

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

	summary := wp.summaryService.GenerateWarSummary(war, attacks)

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

// Test faction ID helper functions
func TestGetOurFactionID(t *testing.T) {
	wp := newTestWarProcessor(&app.Config{})
	wp.ourFactionID = 12345

	war := &app.War{
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction"},
			{ID: 67890, Name: "Enemy Faction"},
		},
	}

	result := wp.getOurFactionID(war)
	if result != 12345 {
		t.Errorf("Expected 12345, got %d", result)
	}
}

func TestGetEnemyFactionID(t *testing.T) {
	wp := newTestWarProcessor(&app.Config{})
	wp.ourFactionID = 12345

	war := &app.War{
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction"},
			{ID: 67890, Name: "Enemy Faction"},
		},
	}

	result := wp.getEnemyFactionID(war)
	if result != 67890 {
		t.Errorf("Expected 67890, got %d", result)
	}
}

func TestGetFactionName(t *testing.T) {
	wp := newTestWarProcessor(&app.Config{})
	wp.ourFactionID = 12345

	war := &app.War{
		Factions: []app.Faction{
			{ID: 12345, Name: "Our Faction"},
			{ID: 67890, Name: "Enemy Faction"},
		},
	}

	// Test known faction
	result := wp.getFactionName(war, 12345)
	if result != "Our Faction" {
		t.Errorf("Expected 'Our Faction', got %q", result)
	}

	// Test unknown faction
	result = wp.getFactionName(war, 99999)
	if result != "Faction 99999" {
		t.Errorf("Expected 'Faction 99999', got %q", result)
	}
}

// Test status change detection
func TestHasStatusChanged(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	// Test identical members - no change
	member1 := app.FactionMember{
		Name: "TestPlayer",
		Status: app.MemberStatus{
			Description: "Okay",
			State:       "Okay",
			Color:       "green",
		},
		LastAction: app.LastAction{
			Status: "Online",
		},
	}
	member2 := member1 // Identical

	if service.HasStatusChanged(member1, member2) {
		t.Error("Expected no status change for identical members")
	}

	// Test different last action status
	member2.LastAction.Status = "Offline"
	if !service.HasStatusChanged(member1, member2) {
		t.Error("Expected status change when LastAction.Status differs")
	}

	// Test hospital countdown change (should NOT be considered a change)
	member2 = member1 // Reset
	member2.Status.Description = "In hospital for 30mins"
	member1.Status.Description = "In hospital for 25mins"
	if service.HasStatusChanged(member1, member2) {
		t.Error("Expected no status change for hospital countdown differences")
	}

	// Test actual status description change
	member2.Status.Description = "Traveling to Mexico"
	if !service.HasStatusChanged(member1, member2) {
		t.Error("Expected status change for different status descriptions")
	}
}
