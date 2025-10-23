package torn

import (
	"context"
	"fmt"
	"sort"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/attack"

	"github.com/rs/zerolog/log"
)

const (
	// TornAPIPageSize is the typical page size returned by the Torn API for attacks
	TornAPIPageSize = 100
)

// AttackProcessor handles business logic for processing attacks
// Separated from infrastructure concerns for better testability
type AttackProcessor struct {
	api TornAPI
}

// NewAttackProcessor creates a new attack processor with the given API client
func NewAttackProcessor(api TornAPI) *AttackProcessor {
	return &AttackProcessor{
		api: api,
	}
}

// TimeRange holds the calculated time range and update mode for fetching attacks.
// FromTime and ToTime are Unix timestamps. UpdateMode indicates whether this is a
// "full" fetch or an "incremental" update.
type TimeRange struct {
	FromTime   int64
	ToTime     int64
	UpdateMode string
}

// FetchStrategy represents the strategy for fetching attacks from the Torn API.
// StrategySimple uses a single API call, while StrategyPaginated uses backwards
// pagination for large time ranges.
type FetchStrategy int

const (
	// StrategySimple uses a single API call for fetching attacks
	StrategySimple FetchStrategy = iota
	// StrategyPaginated uses backwards pagination for large time ranges
	StrategyPaginated
)

// PageResult holds the results from fetching a single page of attacks during
// backwards pagination through the Torn API.
type PageResult struct {
	RelevantAttacks   []app.Attack
	OldestAttackTime  int64
	TotalAttacksCount int
}

// GetAllAttacksForWar fetches all attacks for a specific war timeframe
func (p *AttackProcessor) GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error) {
	return p.GetAttacksForTimeRange(ctx, war, war.Start, nil)
}

// GetAttacksForTimeRange fetches attacks for a specific time range within a war
func (p *AttackProcessor) GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error) {
	if war == nil {
		return nil, fmt.Errorf("war cannot be nil")
	}

	// Calculate time range and update mode
	timeRange := p.CalculateTimeRange(war, latestExistingTimestamp)

	// Functional core: Determine fetch strategy
	startTime := time.Unix(timeRange.FromTime, 0)
	endTime := time.Unix(timeRange.ToTime, 0)
	strategy := attack.DetermineFetchStrategy(startTime, endTime)

	// Log strategy and estimated API calls for observability
	estimatedCalls := attack.EstimateAPICallsRequired(strategy)
	log.Info().
		Int("war_id", war.ID).
		Str("update_mode", timeRange.UpdateMode).
		Str("fetch_strategy", string(strategy.Method)).
		Int("estimated_api_calls", estimatedCalls).
		Int64("fetch_from", timeRange.FromTime).
		Int64("fetch_to", timeRange.ToTime).
		Str("fetch_from_str", startTime.Format("2006-01-02 15:04:05")).
		Str("fetch_to_str", endTime.Format("2006-01-02 15:04:05")).
		Msg("Fetching attacks for war")

	// Imperative shell: Execute the strategy
	return p.executeFetchStrategy(ctx, war, timeRange, strategy)
}

// CalculateTimeRange determines the time range and update mode for fetching attacks
func (p *AttackProcessor) CalculateTimeRange(war *app.War, latestExistingTimestamp *int64) TimeRange {
	var fromTime, toTime int64
	updateMode := "full"

	if latestExistingTimestamp != nil && *latestExistingTimestamp > 0 {
		// Incremental update mode - only fetch new attacks
		updateMode = "incremental"

		// Add 1-hour buffer to handle timing discrepancies
		bufferTime := int64(3600) // 1 hour in seconds
		fromTime = *latestExistingTimestamp - bufferTime

		// Ensure we don't go before war start
		if fromTime < war.Start {
			fromTime = war.Start
		}
	} else {
		// Full population mode - fetch entire war
		fromTime = war.Start
	}

	// Set end time
	if war.End != nil {
		toTime = *war.End
	} else {
		toTime = time.Now().Unix()
	}

	return TimeRange{
		FromTime:   fromTime,
		ToTime:     toTime,
		UpdateMode: updateMode,
	}
}

// FilterRelevantAttacks filters attacks to only those relevant to the war
func (p *AttackProcessor) FilterRelevantAttacks(attacks []app.Attack, war *app.War) []app.Attack {
	var relevantAttacks []app.Attack

	// Get faction IDs from the war
	warFactionIDs := make(map[int]bool)
	for _, faction := range war.Factions {
		warFactionIDs[faction.ID] = true
	}

	for _, attack := range attacks {
		if p.isAttackRelevantToWar(attack, warFactionIDs) {
			relevantAttacks = append(relevantAttacks, attack)
		}
	}

	return relevantAttacks
}

// SortAttacksChronologically sorts attacks by timestamp (oldest first)
func (p *AttackProcessor) SortAttacksChronologically(attacks []app.Attack) {
	sort.Slice(attacks, func(i, j int) bool {
		return attacks[i].Started < attacks[j].Started
	})
}

// isAttackRelevantToWar checks if an attack involves factions from the specified war
func (p *AttackProcessor) isAttackRelevantToWar(attack app.Attack, warFactionIDs map[int]bool) bool {
	// Check if attacker or defender faction is involved in the war
	if attack.Attacker.Faction != nil && warFactionIDs[attack.Attacker.Faction.ID] {
		return true
	}
	if attack.Defender.Faction != nil && warFactionIDs[attack.Defender.Faction.ID] {
		return true
	}

	return false
}

// fetchAttacksSimple fetches attacks using a single API call (for small time ranges)
func (p *AttackProcessor) fetchAttacksSimple(ctx context.Context, war *app.War, timeRange TimeRange) ([]app.Attack, error) {
	log.Debug().Msg("Using simple API call for incremental update")

	attackResp, err := p.api.GetFactionAttacks(ctx, timeRange.FromTime, timeRange.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incremental attacks: %w", err)
	}

	// Filter and collect relevant attacks
	allAttacks := p.FilterRelevantAttacks(attackResp.Attacks, war)

	// Sort chronologically for consistent output
	p.SortAttacksChronologically(allAttacks)

	log.Info().
		Int("total_relevant_attacks", len(allAttacks)).
		Int("war_id", war.ID).
		Str("mode", "incremental_simple").
		Msg("Completed fetching attacks for war")

	return allAttacks, nil
}

// fetchAttacksPaginated fetches attacks using backwards pagination (for large time ranges)
func (p *AttackProcessor) fetchAttacksPaginated(ctx context.Context, war *app.War, timeRange TimeRange) ([]app.Attack, error) {
	var allAttacks []app.Attack
	currentTo := timeRange.ToTime

	for {
		// Fetch one page of attacks
		pageResult, err := p.fetchAttacksPage(ctx, war, timeRange.FromTime, currentTo)
		if err != nil {
			return nil, err
		}

		// Add relevant attacks to our collection
		allAttacks = append(allAttacks, pageResult.RelevantAttacks...)

		// Check if we should stop pagination
		if p.shouldStopPagination(pageResult, timeRange.FromTime) {
			break
		}

		// Set up next page
		currentTo = pageResult.OldestAttackTime - 1

		log.Debug().
			Int64("next_to", currentTo).
			Str("next_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05")).
			Int("total_attacks_so_far", len(allAttacks)).
			Msg("Preparing next pagination request")
	}

	// Sort all attacks chronologically (oldest first) for consistent sheet ordering
	p.SortAttacksChronologically(allAttacks)

	log.Info().
		Int("total_relevant_attacks", len(allAttacks)).
		Int("war_id", war.ID).
		Str("mode", timeRange.UpdateMode+"_paginated").
		Msg("Completed fetching attacks for war")

	return allAttacks, nil
}

// fetchAttacksPage fetches and processes a single page of attacks
func (p *AttackProcessor) fetchAttacksPage(ctx context.Context, war *app.War, fromTime, currentTo int64) (*PageResult, error) {
	log.Debug().
		Int64("current_to", currentTo).
		Str("current_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05")).
		Msg("Fetching attacks page (backwards pagination)")

	// Fetch attacks up to currentTo timestamp
	attackResp, err := p.api.GetFactionAttacks(ctx, fromTime, currentTo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attacks for timeframe %d-%d: %w", fromTime, currentTo, err)
	}

	log.Debug().
		Int("attacks_in_page", len(attackResp.Attacks)).
		Msg("Received attacks from API")

	// Process the page
	return p.processAttacksPage(attackResp.Attacks, war, currentTo), nil
}

// processAttacksPage filters attacks and tracks the oldest timestamp
func (p *AttackProcessor) processAttacksPage(attacks []app.Attack, war *app.War, currentTo int64) *PageResult {
	relevantAttacks := p.FilterRelevantAttacks(attacks, war)
	oldestAttackTime := currentTo

	for _, attack := range attacks {
		// Track the oldest attack timestamp for next pagination
		if attack.Started < oldestAttackTime {
			oldestAttackTime = attack.Started
		}
	}

	log.Debug().
		Int("relevant_attacks_in_page", len(relevantAttacks)).
		Int64("oldest_attack_time", oldestAttackTime).
		Str("oldest_attack_str", time.Unix(oldestAttackTime, 0).Format("2006-01-02 15:04:05")).
		Msg("Filtered attacks for war relevance")

	return &PageResult{
		RelevantAttacks:   relevantAttacks,
		OldestAttackTime:  oldestAttackTime,
		TotalAttacksCount: len(attacks),
	}
}

// executeFetchStrategy executes the determined fetch strategy (imperative shell)
func (p *AttackProcessor) executeFetchStrategy(
	ctx context.Context,
	war *app.War,
	timeRange TimeRange,
	strategy attack.FetchStrategy,
) ([]app.Attack, error) {
	switch strategy.Method {
	case attack.FetchMethodSimple:
		return p.fetchAttacksSimple(ctx, war, timeRange)
	case attack.FetchMethodPaginated:
		return p.fetchAttacksPaginated(ctx, war, timeRange)
	default:
		return nil, fmt.Errorf("unknown fetch method: %s", strategy.Method)
	}
}

// shouldStopPagination determines if we should stop the pagination loop
func (p *AttackProcessor) shouldStopPagination(pageResult *PageResult, fromTime int64) bool {
	// No more attacks returned
	if pageResult.TotalAttacksCount == 0 {
		log.Debug().Msg("No more attacks returned, stopping pagination")
		return true
	}

	// Got less than full page (typical page size is 100)
	if pageResult.TotalAttacksCount < TornAPIPageSize {
		log.Debug().
			Int("attacks_received", pageResult.TotalAttacksCount).
			Msg("Received less than full page, stopping pagination")
		return true
	}

	// Reached the fetch start time
	if pageResult.OldestAttackTime <= fromTime {
		log.Debug().
			Int64("oldest_attack", pageResult.OldestAttackTime).
			Int64("fetch_start", fromTime).
			Msg("Reached fetch start time, stopping pagination")
		return true
	}

	return false
}
