package services

import (
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/attack"
	wardomain "torn_rw_stats/internal/domain/war"

	"github.com/rs/zerolog/log"
)

// WarSummaryService handles war summary generation and statistics calculation,
// aggregating attack data into comprehensive war statistics.
type WarSummaryService struct {
	attackService *attack.AttackProcessingService
}

// NewWarSummaryService creates a new war summary service
func NewWarSummaryService(attackService *attack.AttackProcessingService) *WarSummaryService {
	return &WarSummaryService{
		attackService: attackService,
	}
}

// GenerateWarSummary creates a comprehensive summary of war statistics
func (wss *WarSummaryService) GenerateWarSummary(war *app.War, attacks []app.Attack, ourFactionID int) *app.WarSummary {

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

	// Use domain function to identify factions
	factions := wardomain.IdentifyWarFactions(war, ourFactionID)
	summary.OurFaction = factions.OurFaction
	summary.EnemyFaction = factions.EnemyFaction

	// Use domain function to calculate attack statistics
	stats := attack.CalculateAttackStatistics(attacks, ourFactionID)
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
