package war

import "torn_rw_stats/internal/app"

// FactionPair represents our faction and the enemy faction in a war
type FactionPair struct {
	OurFaction   app.Faction
	EnemyFaction app.Faction
}

// IdentifyWarFactions determines which faction is ours and which is the enemy
// in a war based on our known faction ID.
//
// Pure function: No I/O operations, fully testable with direct inputs.
func IdentifyWarFactions(war *app.War, ourFactionID int) FactionPair {
	var pair FactionPair

	for _, faction := range war.Factions {
		if faction.ID == ourFactionID {
			pair.OurFaction = faction
		} else {
			pair.EnemyFaction = faction
		}
	}

	return pair
}
