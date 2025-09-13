package torn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

type Client struct {
	apiKey       string
	client       *http.Client
	apiCallCount int64
	apiCallMutex sync.Mutex
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IncrementAPICall safely increments the API call counter
func (c *Client) IncrementAPICall() {
	c.apiCallMutex.Lock()
	c.apiCallCount++
	c.apiCallMutex.Unlock()
}

// GetAPICallCount returns the current API call count
func (c *Client) GetAPICallCount() int64 {
	c.apiCallMutex.Lock()
	defer c.apiCallMutex.Unlock()
	return c.apiCallCount
}

// ResetAPICallCount resets the API call counter to zero
func (c *Client) ResetAPICallCount() {
	c.apiCallMutex.Lock()
	c.apiCallCount = 0
	c.apiCallMutex.Unlock()
}

// makeAPIRequest creates and executes an HTTP GET request to the Torn API
func (c *Client) makeAPIRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Debug().
			Err(err).
			Str("url", url).
			Msg("API request failed")
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	c.IncrementAPICall()
	return resp, nil
}

// handleAPIResponse processes the HTTP response and returns the body bytes
func (c *Client) handleAPIResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// GetFactionWars fetches faction wars from the API
func (c *Client) GetFactionWars(ctx context.Context) (*app.WarResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/v2/faction/wars?key=%s", c.apiKey)

	log.Debug().Str("url", url).Msg("Fetching faction wars")

	resp, err := c.makeAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	body, err := c.handleAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	var warResponse app.WarResponse
	if err := json.Unmarshal(body, &warResponse); err != nil {
		return nil, fmt.Errorf("failed to decode war response: %w", err)
	}

	log.Debug().
		Bool("has_ranked_war", warResponse.Wars.Ranked != nil).
		Int("raid_wars", len(warResponse.Wars.Raids)).
		Int("territory_wars", len(warResponse.Wars.Territory)).
		Msg("Successfully fetched faction wars")

	return &warResponse, nil
}

// GetFactionAttacks fetches faction attacks from the API using timestamp pagination
func (c *Client) GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/v2/faction/attacks?key=%s&from=%d&to=%d", c.apiKey, from, to)

	log.Debug().
		Str("url", url).
		Int64("from", from).
		Int64("to", to).
		Str("from_time", time.Unix(from, 0).Format("2006-01-02 15:04:05")).
		Str("to_time", time.Unix(to, 0).Format("2006-01-02 15:04:05")).
		Msg("Fetching faction attacks")

	resp, err := c.makeAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	body, err := c.handleAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	var attackResponse app.AttackResponse
	if err := json.Unmarshal(body, &attackResponse); err != nil {
		return nil, fmt.Errorf("failed to decode attack response: %w", err)
	}

	log.Debug().
		Int("attacks_count", len(attackResponse.Attacks)).
		Int64("from", from).
		Int64("to", to).
		Msg("Successfully fetched faction attacks")

	return &attackResponse, nil
}

// GetAllAttacksForWar fetches all attacks for a specific war timeframe using proper API pagination
func (c *Client) GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error) {
	return c.GetAttacksForTimeRange(ctx, war, war.Start, nil)
}

// GetAttacksForTimeRange fetches attacks for a specific time range within a war
func (c *Client) GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error) {
	if war == nil {
		return nil, fmt.Errorf("war cannot be nil")
	}

	// Calculate time range and update mode
	timeRange := c.calculateTimeRange(war, latestExistingTimestamp)

	log.Info().
		Int("war_id", war.ID).
		Str("update_mode", timeRange.UpdateMode).
		Int64("fetch_from", timeRange.FromTime).
		Int64("fetch_to", timeRange.ToTime).
		Str("fetch_from_str", time.Unix(timeRange.FromTime, 0).Format("2006-01-02 15:04:05")).
		Str("fetch_to_str", time.Unix(timeRange.ToTime, 0).Format("2006-01-02 15:04:05")).
		Msg("Fetching attacks for war")

	// Use simple approach for small incremental updates
	if c.shouldUseSimpleApproach(timeRange) {
		return c.fetchAttacksSimple(ctx, war, timeRange)
	}

	// Use paginated approach for large ranges
	return c.fetchAttacksPaginated(ctx, war, timeRange)
}

// TimeRange holds the calculated time range and update mode
type TimeRange struct {
	FromTime   int64
	ToTime     int64
	UpdateMode string
}

// calculateTimeRange determines the time range and update mode for fetching attacks
func (c *Client) calculateTimeRange(war *app.War, latestExistingTimestamp *int64) TimeRange {
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

// shouldUseSimpleApproach determines if we can use a single API call instead of pagination
func (c *Client) shouldUseSimpleApproach(timeRange TimeRange) bool {
	if timeRange.UpdateMode != "incremental" {
		return false
	}

	const maxSimpleRange = 24 * 60 * 60 // 24 hours
	duration := timeRange.ToTime - timeRange.FromTime
	return duration <= maxSimpleRange
}

// fetchAttacksSimple fetches attacks using a single API call (for small time ranges)
func (c *Client) fetchAttacksSimple(ctx context.Context, war *app.War, timeRange TimeRange) ([]app.Attack, error) {
	log.Debug().Msg("Using simple API call for incremental update")

	attackResp, err := c.GetFactionAttacks(ctx, timeRange.FromTime, timeRange.ToTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch incremental attacks: %w", err)
	}

	// Filter and collect relevant attacks
	var allAttacks []app.Attack
	for _, attack := range attackResp.Attacks {
		if c.isAttackRelevantToWar(attack, war) {
			allAttacks = append(allAttacks, attack)
		}
	}

	// Sort chronologically for consistent output
	c.sortAttacksChronologically(allAttacks)

	log.Info().
		Int("total_relevant_attacks", len(allAttacks)).
		Int("war_id", war.ID).
		Str("mode", "incremental_simple").
		Msg("Completed fetching attacks for war")

	return allAttacks, nil
}

// fetchAttacksPaginated fetches attacks using backwards pagination (for large time ranges)
func (c *Client) fetchAttacksPaginated(ctx context.Context, war *app.War, timeRange TimeRange) ([]app.Attack, error) {
	var allAttacks []app.Attack
	currentTo := timeRange.ToTime

	for {
		// Fetch one page of attacks
		pageResult, err := c.fetchAttacksPage(ctx, war, timeRange.FromTime, currentTo)
		if err != nil {
			return nil, err
		}

		// Add relevant attacks to our collection
		allAttacks = append(allAttacks, pageResult.RelevantAttacks...)

		// Check if we should stop pagination
		if c.shouldStopPagination(pageResult, timeRange.FromTime) {
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
	c.sortAttacksChronologically(allAttacks)

	log.Info().
		Int("total_relevant_attacks", len(allAttacks)).
		Int("war_id", war.ID).
		Str("mode", timeRange.UpdateMode+"_paginated").
		Msg("Completed fetching attacks for war")

	return allAttacks, nil
}

// PageResult holds the results from fetching a single page of attacks
type PageResult struct {
	RelevantAttacks   []app.Attack
	OldestAttackTime  int64
	TotalAttacksCount int
}

// fetchAttacksPage fetches and processes a single page of attacks
func (c *Client) fetchAttacksPage(ctx context.Context, war *app.War, fromTime, currentTo int64) (*PageResult, error) {
	log.Debug().
		Int64("current_to", currentTo).
		Str("current_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05")).
		Msg("Fetching attacks page (backwards pagination)")

	// Fetch attacks up to currentTo timestamp
	attackResp, err := c.GetFactionAttacks(ctx, fromTime, currentTo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attacks for timeframe %d-%d: %w", fromTime, currentTo, err)
	}

	log.Debug().
		Int("attacks_in_page", len(attackResp.Attacks)).
		Msg("Received attacks from API")

	// Process the page
	return c.processAttacksPage(attackResp.Attacks, war, currentTo), nil
}

// processAttacksPage filters attacks and tracks the oldest timestamp
func (c *Client) processAttacksPage(attacks []app.Attack, war *app.War, currentTo int64) *PageResult {
	var relevantAttacks []app.Attack
	oldestAttackTime := currentTo

	for _, attack := range attacks {
		if c.isAttackRelevantToWar(attack, war) {
			relevantAttacks = append(relevantAttacks, attack)
		}

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

// shouldStopPagination determines if we should stop the pagination loop
func (c *Client) shouldStopPagination(pageResult *PageResult, fromTime int64) bool {
	// No more attacks returned
	if pageResult.TotalAttacksCount == 0 {
		log.Debug().Msg("No more attacks returned, stopping pagination")
		return true
	}

	// Got less than full page (typical page size is 100)
	if pageResult.TotalAttacksCount < 100 {
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

// sortAttacksChronologically sorts attacks by timestamp (oldest first)
func (c *Client) sortAttacksChronologically(attacks []app.Attack) {
	sort.Slice(attacks, func(i, j int) bool {
		return attacks[i].Started < attacks[j].Started
	})
}

// isAttackRelevantToWar checks if an attack involves factions from the specified war
func (c *Client) isAttackRelevantToWar(attack app.Attack, war *app.War) bool {
	// Get faction IDs from the war
	warFactionIDs := make(map[int]bool)
	for _, faction := range war.Factions {
		warFactionIDs[faction.ID] = true
	}

	// Check if attacker or defender faction is involved in the war
	if attack.Attacker.Faction != nil && warFactionIDs[attack.Attacker.Faction.ID] {
		return true
	}
	if attack.Defender.Faction != nil && warFactionIDs[attack.Defender.Faction.ID] {
		return true
	}

	return false
}

// GetFactionBasic fetches faction basic data from the API
func (c *Client) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/faction/%d?selections=basic&key=%s", factionID, c.apiKey)

	log.Debug().
		Str("url", url).
		Int("faction_id", factionID).
		Msg("Fetching faction basic data")

	resp, err := c.makeAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	body, err := c.handleAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	var factionResponse app.FactionBasicResponse
	if err := json.Unmarshal(body, &factionResponse); err != nil {
		return nil, fmt.Errorf("failed to decode faction response: %w", err)
	}

	log.Debug().
		Int("faction_id", factionID).
		Int("members_count", len(factionResponse.Members)).
		Msg("Successfully fetched faction basic data")

	return &factionResponse, nil
}

// GetOwnFaction gets the current user's faction information
func (c *Client) GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/faction/?selections=basic&key=%s", c.apiKey)

	log.Debug().
		Str("url", url).
		Msg("Fetching own faction data")

	resp, err := c.makeAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	body, err := c.handleAPIResponse(resp)
	if err != nil {
		return nil, err
	}

	var factionResponse app.FactionInfoResponse
	if err := json.Unmarshal(body, &factionResponse); err != nil {
		return nil, fmt.Errorf("failed to decode faction response: %w", err)
	}

	log.Debug().
		Int("faction_id", factionResponse.ID).
		Str("faction_name", factionResponse.Name).
		Str("faction_tag", factionResponse.Tag).
		Msg("Successfully fetched own faction data")

	return &factionResponse, nil
}
