package attack

import (
	"testing"
	"torn_rw_stats/internal/app"
)

func TestBuildFactionIDMap(t *testing.T) {
	war := &app.War{
		Factions: []app.Faction{
			{ID: 100},
			{ID: 200},
		},
	}

	result := BuildFactionIDMap(war)

	if !result[100] || !result[200] {
		t.Errorf("Expected faction IDs 100 and 200 in map")
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 factions, got %d", len(result))
	}
}

func TestFilterRelevantAttacks(t *testing.T) {
	warFactionIDs := map[int]bool{100: true, 200: true}

	attacks := []app.Attack{
		{Attacker: app.User{Faction: &app.Faction{ID: 100}}}, // Relevant
		{Attacker: app.User{Faction: &app.Faction{ID: 300}}}, // Not relevant
		{Defender: app.User{Faction: &app.Faction{ID: 200}}}, // Relevant
	}

	result := FilterRelevantAttacks(attacks, warFactionIDs)

	if len(result) != 2 {
		t.Errorf("Expected 2 relevant attacks, got %d", len(result))
	}
}

func TestIsAttackRelevantToWar(t *testing.T) {
	warFactionIDs := map[int]bool{100: true}

	tests := []struct {
		name     string
		attack   app.Attack
		expected bool
	}{
		{
			name:     "attacker in war",
			attack:   app.Attack{Attacker: app.User{Faction: &app.Faction{ID: 100}}},
			expected: true,
		},
		{
			name:     "defender in war",
			attack:   app.Attack{Defender: app.User{Faction: &app.Faction{ID: 100}}},
			expected: true,
		},
		{
			name:     "neither in war",
			attack:   app.Attack{Attacker: app.User{Faction: &app.Faction{ID: 999}}},
			expected: false,
		},
		{
			name:     "no factions",
			attack:   app.Attack{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAttackRelevantToWar(tt.attack, warFactionIDs)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
