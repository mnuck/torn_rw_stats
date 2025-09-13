package processing

import (
	"fmt"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// WarSummaryService handles war summary generation and statistics calculation
type WarSummaryService struct {
	attackService *AttackProcessingService
}

// NewWarSummaryService creates a new war summary service
func NewWarSummaryService(attackService *AttackProcessingService) *WarSummaryService {
	return &WarSummaryService{
		attackService: attackService,
	}
}

// GenerateWarSummary creates a comprehensive summary of war statistics
func (wss *WarSummaryService) GenerateWarSummary(war *app.War, attacks []app.Attack) *app.WarSummary {
	ourFactionID := wss.attackService.getOurFactionID(war)

	summary := &app.WarSummary{
		WarID:       war.ID,
		StartTime:   time.Unix(war.Start, 0),
		Status:      "Active",
		LastUpdated: time.Now(),
	}

	if war.End != nil {
		endTime := time.Unix(*war.End, 0)
		summary.EndTime = &endTime
		summary.Status = "Completed"
	}

	// Find our faction and enemy faction
	for _, faction := range war.Factions {
		if faction.ID == ourFactionID {
			summary.OurFaction = faction
		} else {
			summary.EnemyFaction = faction
		}
	}

	// Calculate attack statistics
	stats := wss.calculateAttackStatistics(attacks, ourFactionID)
	summary.TotalAttacks = stats.TotalAttacks
	summary.AttacksWon = stats.AttacksWon
	summary.AttacksLost = stats.AttacksLost
	summary.RespectGained = stats.RespectGained
	summary.RespectLost = stats.RespectLost

	// Set war name based on factions
	summary.WarName = fmt.Sprintf("%s vs %s", summary.OurFaction.Name, summary.EnemyFaction.Name)

	log.Debug().
		Int("war_id", war.ID).
		Int("total_attacks", summary.TotalAttacks).
		Int("attacks_won", summary.AttacksWon).
		Int("attacks_lost", summary.AttacksLost).
		Float64("respect_gained", summary.RespectGained).
		Float64("respect_lost", summary.RespectLost).
		Msg("Generated war summary")

	return summary
}

// AttackStatistics holds calculated attack statistics
type AttackStatistics struct {
	TotalAttacks  int
	AttacksWon    int
	AttacksLost   int
	RespectGained float64
	RespectLost   float64
}

// calculateAttackStatistics computes comprehensive attack statistics for our faction
func (wss *WarSummaryService) calculateAttackStatistics(attacks []app.Attack, ourFactionID int) AttackStatistics {
	var stats AttackStatistics

	for _, attack := range attacks {
		if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
			// Our attack
			stats.TotalAttacks++
			stats.RespectGained += attack.RespectGain
			stats.RespectLost += attack.RespectLoss

			// Determine if we won (simplified - you may need more sophisticated logic)
			if wss.isSuccessfulAttack(attack.Result) {
				stats.AttacksWon++
			} else {
				stats.AttacksLost++
			}
		} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
			// Attack against us
			stats.TotalAttacks++

			// For defensive stats, respect gain/loss is inverted
			stats.RespectLost += attack.RespectGain
			stats.RespectGained += attack.RespectLoss

			// We "won" if we defended successfully
			if wss.isSuccessfulDefense(attack.Result) {
				stats.AttacksWon++
			} else {
				stats.AttacksLost++
			}
		}
	}

	return stats
}

// isSuccessfulAttack determines if an attack result represents a successful attack
func (wss *WarSummaryService) isSuccessfulAttack(result string) bool {
	successfulResults := []string{"Hospitalized", "Mugged", "Left"}
	for _, successResult := range successfulResults {
		if result == successResult {
			return true
		}
	}
	return false
}

// isSuccessfulDefense determines if an attack result represents a successful defense
func (wss *WarSummaryService) isSuccessfulDefense(result string) bool {
	successfulDefenseResults := []string{"Stalemate", "Escape", "Assist"}
	for _, defenseResult := range successfulDefenseResults {
		if result == defenseResult {
			return true
		}
	}
	return false
}
