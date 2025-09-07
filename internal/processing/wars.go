package processing

import (
	"context"
	"fmt"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/sheets"
	"torn_rw_stats/internal/torn"

	"github.com/rs/zerolog/log"
)

// WarProcessor handles war detection and processing
type WarProcessor struct {
	tornClient   *torn.Client
	sheetsClient *sheets.Client
	config       *app.Config
}

func NewWarProcessor(tornClient *torn.Client, sheetsClient *sheets.Client, config *app.Config) *WarProcessor {
	return &WarProcessor{
		tornClient:   tornClient,
		sheetsClient: sheetsClient,
		config:       config,
	}
}

// ProcessActiveWars fetches current wars and processes each one
func (wp *WarProcessor) ProcessActiveWars(ctx context.Context) error {
	log.Info().Msg("Processing active wars")

	warResponse, err := wp.tornClient.GetFactionWars(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch faction wars: %w", err)
	}

	var processedWars int

	// Process ranked war if it exists
	if warResponse.Wars.Ranked != nil {
		log.Info().
			Int("war_id", warResponse.Wars.Ranked.ID).
			Msg("Processing ranked war")

		if err := wp.processWar(ctx, warResponse.Wars.Ranked); err != nil {
			log.Error().
				Err(err).
				Int("war_id", warResponse.Wars.Ranked.ID).
				Msg("Failed to process ranked war")
		} else {
			processedWars++
		}
	}

	// Process raid wars
	for _, war := range warResponse.Wars.Raids {
		log.Info().
			Int("war_id", war.ID).
			Msg("Processing raid war")

		if err := wp.processWar(ctx, &war); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Msg("Failed to process raid war")
		} else {
			processedWars++
		}
	}

	// Process territory wars
	for _, war := range warResponse.Wars.Territory {
		log.Info().
			Int("war_id", war.ID).
			Msg("Processing territory war")

		if err := wp.processWar(ctx, &war); err != nil {
			log.Error().
				Err(err).
				Int("war_id", war.ID).
				Msg("Failed to process territory war")
		} else {
			processedWars++
		}
	}

	log.Info().
		Int("processed_wars", processedWars).
		Msg("Completed processing active wars")

	return nil
}

// processWar handles processing a single war
func (wp *WarProcessor) processWar(ctx context.Context, war *app.War) error {
	log.Debug().
		Int("war_id", war.ID).
		Int("factions_count", len(war.Factions)).
		Int64("start_time", war.Start).
		Msg("Processing individual war")

	// Ensure sheets exist for this war
	sheetConfig, err := wp.sheetsClient.EnsureWarSheets(ctx, wp.config.SpreadsheetID, war)
	if err != nil {
		return fmt.Errorf("failed to ensure war sheets: %w", err)
	}

	// Check if we have existing records to determine update mode
	existingInfo, err := wp.sheetsClient.ReadExistingRecords(ctx, wp.config.SpreadsheetID, sheetConfig.RecordsTabName)
	if err != nil {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Fetch attacks based on existing data
	var attacks []app.Attack
	if existingInfo.RecordCount == 0 {
		// Full population mode
		log.Debug().Int("war_id", war.ID).Msg("Using full population mode - no existing records")
		attacks, err = wp.tornClient.GetAllAttacksForWar(ctx, war)
	} else {
		// Incremental update mode
		log.Debug().
			Int("war_id", war.ID).
			Int("existing_records", existingInfo.RecordCount).
			Int64("latest_timestamp", existingInfo.LatestTimestamp).
			Msg("Using incremental update mode - existing records found")
		attacks, err = wp.tornClient.GetAttacksForTimeRange(ctx, war, war.Start, &existingInfo.LatestTimestamp)
	}
	
	if err != nil {
		return fmt.Errorf("failed to fetch attacks for war: %w", err)
	}

	log.Debug().
		Int("war_id", war.ID).
		Int("attacks_count", len(attacks)).
		Msg("Fetched attacks for war")

	// Process attack data into records
	records := wp.processAttacksIntoRecords(attacks, war)

	// Generate war summary
	summary := wp.generateWarSummary(war, attacks)

	// Update sheets
	if err := wp.sheetsClient.UpdateWarSummary(ctx, wp.config.SpreadsheetID, sheetConfig, summary); err != nil {
		return fmt.Errorf("failed to update war summary: %w", err)
	}

	if err := wp.sheetsClient.UpdateAttackRecords(ctx, wp.config.SpreadsheetID, sheetConfig, records); err != nil {
		return fmt.Errorf("failed to update attack records: %w", err)
	}

	log.Info().
		Int("war_id", war.ID).
		Int("attacks_processed", len(attacks)).
		Int("records_created", len(records)).
		Msg("Successfully processed war")

	return nil
}

// processAttacksIntoRecords converts API attack data into sheet records
func (wp *WarProcessor) processAttacksIntoRecords(attacks []app.Attack, war *app.War) []app.AttackRecord {
	var records []app.AttackRecord

	// Determine our faction ID (we'll need to configure this)
	ourFactionID := wp.getOurFactionID(war)

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
		if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
			record.Direction = "Outgoing"
		} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
			record.Direction = "Incoming"
		} else {
			record.Direction = "Unknown"
		}

		records = append(records, record)
	}

	log.Debug().
		Int("total_attacks", len(attacks)).
		Int("records_created", len(records)).
		Int("our_faction_id", ourFactionID).
		Msg("Processed attacks into records")

	return records
}

// generateWarSummary creates a summary of the war statistics
func (wp *WarProcessor) generateWarSummary(war *app.War, attacks []app.Attack) *app.WarSummary {
	ourFactionID := wp.getOurFactionID(war)

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
	var attacksWon, attacksLost int
	var respectGained, respectLost float64

	for _, attack := range attacks {
		if attack.Attacker.Faction != nil && attack.Attacker.Faction.ID == ourFactionID {
			// Our attack
			summary.TotalAttacks++
			respectGained += attack.RespectGain
			respectLost += attack.RespectLoss

			// Determine if we won (simplified - you may need more sophisticated logic)
			if attack.Result == "Hospitalized" || attack.Result == "Mugged" || attack.Result == "Left" {
				attacksWon++
			} else {
				attacksLost++
			}
		} else if attack.Defender.Faction != nil && attack.Defender.Faction.ID == ourFactionID {
			// Attack against us
			summary.TotalAttacks++

			// For defensive stats, respect gain/loss is inverted
			respectLost += attack.RespectGain
			respectGained += attack.RespectLoss

			// We "won" if we defended successfully
			if attack.Result == "Stalemate" || attack.Result == "Escape" || attack.Result == "Assist" {
				attacksWon++
			} else {
				attacksLost++
			}
		}
	}

	summary.AttacksWon = attacksWon
	summary.AttacksLost = attacksLost
	summary.RespectGained = respectGained
	summary.RespectLost = respectLost

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

// getOurFactionID determines which faction is "ours" in the war
// This is a placeholder - you'll need to configure this based on your faction ID
func (wp *WarProcessor) getOurFactionID(war *app.War) int {
	// For now, we'll assume the first faction is ours
	// In a real implementation, you'd want to configure this
	if len(war.Factions) > 0 {
		return war.Factions[0].ID
	}
	return 0
}