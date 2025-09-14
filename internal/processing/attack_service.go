package processing

import (
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// AttackProcessingService handles attack data processing and analysis
type AttackProcessingService struct {
	ourFactionID int
}

// NewAttackProcessingService creates a new attack processing service
func NewAttackProcessingService(ourFactionID int) *AttackProcessingService {
	return &AttackProcessingService{
		ourFactionID: ourFactionID,
	}
}

// ProcessAttacksIntoRecords converts attack data into detailed attack records
func (aps *AttackProcessingService) ProcessAttacksIntoRecords(attacks []app.Attack, war *app.War) []app.AttackRecord {
	var records []app.AttackRecord

	// Determine our faction ID from the war
	ourFactionID := aps.getOurFactionID(war)

	for _, attack := range attacks {
		record := app.AttackRecord{
			AttackID:            attack.ID,
			Code:                attack.Code,
			Started:             time.Unix(attack.Started, 0),
			Ended:               time.Unix(attack.Ended, 0),
			AttackerID:          attack.Attacker.ID,
			AttackerName:        attack.Attacker.Name,
			AttackerLevel:       attack.Attacker.Level,
			DefenderID:          attack.Defender.ID,
			DefenderName:        attack.Defender.Name,
			DefenderLevel:       attack.Defender.Level,
			Result:              attack.Result,
			RespectGain:         attack.RespectGain,
			RespectLoss:         attack.RespectLoss,
			Chain:               attack.Chain,
			IsInterrupted:       attack.IsInterrupted,
			IsStealthed:         attack.IsStealthed,
			IsRaid:              attack.IsRaid,
			IsRankedWar:         attack.IsRankedWar,
			ModifierFairFight:   attack.Modifiers.FairFight,
			ModifierWar:         attack.Modifiers.War,
			ModifierRetaliation: attack.Modifiers.Retaliation,
			ModifierGroup:       attack.Modifiers.Group,
			ModifierOverseas:    attack.Modifiers.Overseas,
			ModifierChain:       attack.Modifiers.Chain,
			ModifierWarlord:     attack.Modifiers.Warlord,
		}

		// Handle attacker faction
		if attack.Attacker.Faction != nil {
			record.AttackerFactionID = &attack.Attacker.Faction.ID
			record.AttackerFactionName = attack.Attacker.Faction.Name
		}

		// Handle defender faction
		if attack.Defender.Faction != nil {
			record.DefenderFactionID = &attack.Defender.Faction.ID
			record.DefenderFactionName = attack.Defender.Faction.Name
		}

		// Handle finishing hit effects (take the first one if it exists)
		if len(attack.FinishingHitEffects) > 0 {
			record.FinishingHitName = attack.FinishingHitEffects[0].Name
			record.FinishingHitValue = attack.FinishingHitEffects[0].Value
		}

		// Determine attack direction
		record.Direction = aps.determineAttackDirection(attack, ourFactionID)

		records = append(records, record)
	}

	log.Debug().
		Int("total_attacks", len(attacks)).
		Int("records_created", len(records)).
		Int("our_faction_id", ourFactionID).
		Msg("Processed attacks into records")

	return records
}

// determineAttackDirection determines if an attack is outgoing, incoming, or unknown
func (aps *AttackProcessingService) determineAttackDirection(attack app.Attack, ourFactionID int) string {
	if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
		return "Outgoing"
	} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
		return "Incoming"
	}
	return "Unknown"
}

// getOurFactionID determines which faction is "ours" in the war
func (aps *AttackProcessingService) getOurFactionID(war *app.War) int {
	if aps.ourFactionID != 0 {
		// Check if our configured faction ID is in this war
		for _, faction := range war.Factions {
			if faction.ID == aps.ourFactionID {
				return aps.ourFactionID
			}
		}
	}

	// Fallback: return the first faction (could be enhanced with better logic)
	if len(war.Factions) > 0 {
		return war.Factions[0].ID
	}

	return 0
}

