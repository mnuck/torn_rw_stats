package services

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"torn_rw_stats/internal/application/ports"
	"torn_rw_stats/internal/domain"
	"torn_rw_stats/internal/domain/attack"
)

const (
	// TornAPIPageSize is the typical page size returned by the Torn API for attacks
	TornAPIPageSize = 100
)

// AttackFetcher orchestrates fetching attacks from the Torn API, choosing
// between simple and paginated strategies based on the time range.
type AttackFetcher struct {
	api ports.TornClient
}

// NewAttackFetcher creates a new attack fetcher with the given API client
func NewAttackFetcher(api ports.TornClient) *AttackFetcher {
	return &AttackFetcher{
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

// PageResult holds the results from fetching a single page of attacks during
// backwards pagination through the Torn API.
type PageResult struct {
	RelevantAttacks   []domain.Attack
	OldestAttackTime  int64
	TotalAttacksCount int
}

// GetAllAttacksForWar fetches all attacks for a specific war timeframe
func (p *AttackFetcher) GetAllAttacksForWar(ctx context.Context, war *domain.War) ([]domain.Attack, error) {
	return p.GetAttacksForTimeRange(ctx, war, war.Start, nil)
}

// GetAttacksForTimeRange fetches attacks for a specific time range within a war
func (p *AttackFetcher) GetAttacksForTimeRange(ctx context.Context, war *domain.War, fromTime int64, latestExistingTimestamp *int64) ([]domain.Attack, error) {
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
	slog.Info("Fetching attacks for war",
		"war_id", war.ID,
		"update_mode", timeRange.UpdateMode,
		"fetch_strategy", string(strategy.Method),
		"estimated_api_calls", estimatedCalls,
		"fetch_from", timeRange.FromTime,
		"fetch_to", timeRange.ToTime,
		"fetch_from_str", startTime.Format("2006-01-02 15:04:05"),
		"fetch_to_str", endTime.Format("2006-01-02 15:04:05"))

	// Imperative shell: Execute the strategy
	return p.executeFetchStrategy(ctx, war, timeRange, strategy)
}

// fetchAttacksSimple fetches attacks using a single API call (for small time ranges)
func (p *AttackFetcher) fetchAttacksSimple(ctx context.Context, war *domain.War, timeRange TimeRange) ([]domain.Attack, error) {
	slog.Debug("Using simple API call for incremental update")

	attackResp, err := p.api.GetFactionAttacks(ctx, timeRange.FromTime, timeRange.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incremental attacks: %w", err)
	}

	// Filter and collect relevant attacks
	warFactionIDs := attack.BuildFactionIDMap(war)
	filtered := attack.FilterRelevantAttacks(attackResp, warFactionIDs)
	allAttacks := attack.SortAttacksChronologically(filtered)

	slog.Info("Completed fetching attacks for war",
		"total_relevant_attacks", len(allAttacks),
		"war_id", war.ID,
		"mode", "incremental_simple")

	return allAttacks, nil
}

// fetchAttacksPaginated fetches attacks using backwards pagination (for large time ranges)
func (p *AttackFetcher) fetchAttacksPaginated(ctx context.Context, war *domain.War, timeRange TimeRange) ([]domain.Attack, error) {
	var allAttacks []domain.Attack
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

		slog.Debug("Preparing next pagination request",
			"next_to", currentTo,
			"next_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05"),
			"total_attacks_so_far", len(allAttacks))
	}

	// Sort all attacks chronologically (oldest first) for consistent sheet ordering
	allAttacks = attack.SortAttacksChronologically(allAttacks)

	slog.Info("Completed fetching attacks for war",
		"total_relevant_attacks", len(allAttacks),
		"war_id", war.ID,
		"mode", timeRange.UpdateMode+"_paginated")

	return allAttacks, nil
}

// fetchAttacksPage fetches and processes a single page of attacks
func (p *AttackFetcher) fetchAttacksPage(ctx context.Context, war *domain.War, fromTime, currentTo int64) (*PageResult, error) {
	slog.Debug("Fetching attacks page (backwards pagination)",
		"current_to", currentTo,
		"current_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05"))

	// Fetch attacks up to currentTo timestamp
	attackResp, err := p.api.GetFactionAttacks(ctx, fromTime, currentTo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attacks for timeframe %d-%d: %w", fromTime, currentTo, err)
	}

	slog.Debug("Received attacks from API",
		"attacks_in_page", len(attackResp))

	// Process the page
	return p.processAttacksPage(attackResp, war, currentTo), nil
}

// processAttacksPage filters attacks and tracks the oldest timestamp
func (p *AttackFetcher) processAttacksPage(attacks []domain.Attack, war *domain.War, currentTo int64) *PageResult {
	warFactionIDs := attack.BuildFactionIDMap(war)
	relevantAttacks := attack.FilterRelevantAttacks(attacks, warFactionIDs)
	oldestAttackTime := attack.FindOldestAttackTime(attacks, currentTo)

	slog.Debug("Filtered attacks for war relevance",
		"relevant_attacks_in_page", len(relevantAttacks),
		"oldest_attack_time", oldestAttackTime,
		"oldest_attack_str", time.Unix(oldestAttackTime, 0).Format("2006-01-02 15:04:05"))

	return &PageResult{
		RelevantAttacks:   relevantAttacks,
		OldestAttackTime:  oldestAttackTime,
		TotalAttacksCount: len(attacks),
	}
}

// executeFetchStrategy executes the determined fetch strategy (imperative shell)
func (p *AttackFetcher) executeFetchStrategy(
	ctx context.Context,
	war *domain.War,
	timeRange TimeRange,
	strategy attack.FetchStrategy,
) ([]domain.Attack, error) {
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
func (p *AttackFetcher) shouldStopPagination(pageResult *PageResult, fromTime int64) bool {
	decision := attack.ShouldStopPagination(
		pageResult.TotalAttacksCount,
		pageResult.OldestAttackTime,
		fromTime,
		TornAPIPageSize,
	)

	if decision.ShouldStop {
		switch decision.Reason {
		case "no_more_attacks":
			slog.Debug("No more attacks returned, stopping pagination")
		case "partial_page":
			slog.Debug("Received less than full page, stopping pagination",
				"attacks_received", decision.AttacksProcessed)
		case "reached_start_time":
			slog.Debug("Reached fetch start time, stopping pagination",
				"oldest_attack", decision.OldestTimestamp,
				"fetch_start", fromTime)
		}
	}

	return decision.ShouldStop
}
