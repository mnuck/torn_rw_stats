package torn

import (
	"context"
	"fmt"
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

	// Functional core: Calculate time range and update mode
	timeRangeResult := attack.CalculateTimeRange(war, latestExistingTimestamp, time.Now().Unix())
	timeRange := TimeRange{
		FromTime:   timeRangeResult.FromTime,
		ToTime:     timeRangeResult.ToTime,
		UpdateMode: timeRangeResult.UpdateMode,
	}

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



// fetchAttacksSimple fetches attacks using a single API call (for small time ranges)
func (p *AttackProcessor) fetchAttacksSimple(ctx context.Context, war *app.War, timeRange TimeRange) ([]app.Attack, error) {
	log.Debug().Msg("Using simple API call for incremental update")

	attackResp, err := p.api.GetFactionAttacks(ctx, timeRange.FromTime, timeRange.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incremental attacks: %w", err)
	}

	// Filter and collect relevant attacks
	warFactionIDs := attack.BuildFactionIDMap(war)
	filtered := attack.FilterRelevantAttacks(attackResp.Attacks, warFactionIDs)
	allAttacks := attack.SortAttacksChronologically(filtered)

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
	allAttacks = attack.SortAttacksChronologically(allAttacks)

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
	warFactionIDs := attack.BuildFactionIDMap(war)
	relevantAttacks := attack.FilterRelevantAttacks(attacks, warFactionIDs)
	oldestAttackTime := attack.FindOldestAttackTime(attacks, currentTo)

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
	decision := attack.ShouldStopPagination(
		pageResult.TotalAttacksCount,
		pageResult.OldestAttackTime,
		fromTime,
		TornAPIPageSize,
	)

	if decision.ShouldStop {
		switch decision.Reason {
		case "no_more_attacks":
			log.Debug().Msg("No more attacks returned, stopping pagination")
		case "partial_page":
			log.Debug().
				Int("attacks_received", decision.AttacksProcessed).
				Msg("Received less than full page, stopping pagination")
		case "reached_start_time":
			log.Debug().
				Int64("oldest_attack", decision.OldestTimestamp).
				Int64("fetch_start", fromTime).
				Msg("Reached fetch start time, stopping pagination")
		}
	}

	return decision.ShouldStop
}
