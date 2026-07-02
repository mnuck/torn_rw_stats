package attack

import "torn_rw_stats/internal/domain"

// BuildFactionIDMap creates a map of faction IDs from a war for O(1) lookup
// Pure function: No I/O, deterministic output from input
func BuildFactionIDMap(war *domain.War) map[int]bool {
	factionIDs := make(map[int]bool)
	for _, faction := range war.Factions {
		factionIDs[faction.ID] = true
	}
	return factionIDs
}

// FilterRelevantAttacks returns attacks where attacker or defender is in warFactionIDs
// Pure function: No I/O, returns new slice without modifying input
func FilterRelevantAttacks(attacks []domain.Attack, warFactionIDs map[int]bool) []domain.Attack {
	var relevantAttacks []domain.Attack
	for _, attack := range attacks {
		if IsAttackRelevantToWar(attack, warFactionIDs) {
			relevantAttacks = append(relevantAttacks, attack)
		}
	}
	return relevantAttacks
}

// IsAttackRelevantToWar checks if an attack involves any faction from the war
// Pure function: No I/O, simple boolean logic
func IsAttackRelevantToWar(attack domain.Attack, warFactionIDs map[int]bool) bool {
	if attack.Attacker.Faction != nil && warFactionIDs[attack.Attacker.Faction.ID] {
		return true
	}
	if attack.Defender.Faction != nil && warFactionIDs[attack.Defender.Faction.ID] {
		return true
	}
	return false
}
