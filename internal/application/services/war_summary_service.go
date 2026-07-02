package services

import (
	"fmt"
	"time"

	"log/slog"
	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/attack"

	wardomain "torn_rw_stats/internal/domain/war"
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
func (wss *WarSummaryService) GenerateWarSummary(war *domain.War, attacks []domain.Attack, ourFactionID int) *domain.WarSummary {

	summary := &domain.WarSummary{
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

	slog.Debug("Generated war summary",
		"war_id", war.ID,
		"total_attacks", summary.TotalAttacks,
		"attacks_won", summary.AttacksWon,
		"attacks_lost", summary.AttacksLost,
		"respect_gained", summary.RespectGained,
		"respect_lost", summary.RespectLost)

	return summary
}
