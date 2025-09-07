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

	var allAttacks []app.Attack
	
	// Determine time range based on whether we're doing full population or incremental update
	var actualFromTime, actualToTime int64
	updateMode := "full"
	
	if latestExistingTimestamp != nil && *latestExistingTimestamp > 0 {
		// Incremental update mode - only fetch new attacks
		updateMode = "incremental"
		
		// Add 1-hour buffer to handle timing discrepancies
		bufferTime := int64(3600) // 1 hour in seconds
		actualFromTime = *latestExistingTimestamp - bufferTime
		
		// Ensure we don't go before war start
		if actualFromTime < war.Start {
			actualFromTime = war.Start
		}
		
		// Set end time
		if war.End != nil {
			actualToTime = *war.End
		} else {
			actualToTime = time.Now().Unix()
		}
	} else {
		// Full population mode - fetch entire war
		actualFromTime = war.Start
		if war.End != nil {
			actualToTime = *war.End
		} else {
			actualToTime = time.Now().Unix()
		}
	}

	log.Info().
		Int("war_id", war.ID).
		Str("update_mode", updateMode).
		Int64("fetch_from", actualFromTime).
		Int64("fetch_to", actualToTime).
		Str("fetch_from_str", time.Unix(actualFromTime, 0).Format("2006-01-02 15:04:05")).
		Str("fetch_to_str", time.Unix(actualToTime, 0).Format("2006-01-02 15:04:05")).
		Msg("Fetching attacks for war")

	// For incremental updates with small time ranges, we can use simple API call
	if updateMode == "incremental" {
		timeRange := actualToTime - actualFromTime
		const maxSimpleRange = 24 * 60 * 60 // 24 hours
		
		if timeRange <= maxSimpleRange {
			log.Debug().Msg("Using simple API call for incremental update")
			
			attackResp, err := c.GetFactionAttacks(ctx, actualFromTime, actualToTime)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch incremental attacks: %w", err)
			}
			
			// Filter and return relevant attacks
			for _, attack := range attackResp.Attacks {
				if c.isAttackRelevantToWar(attack, war) {
					allAttacks = append(allAttacks, attack)
				}
			}
			
			// Sort chronologically for consistent output
			sort.Slice(allAttacks, func(i, j int) bool {
				return allAttacks[i].Started < allAttacks[j].Started
			})
			
			log.Info().
				Int("total_relevant_attacks", len(allAttacks)).
				Int("war_id", war.ID).
				Str("mode", "incremental_simple").
				Msg("Completed fetching attacks for war")
			
			return allAttacks, nil
		}
	}

	// Use paginated approach for full population or large incremental updates
	currentTo := actualToTime
	
	for {
		log.Debug().
			Int64("current_to", currentTo).
			Str("current_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05")).
			Msg("Fetching attacks page (backwards pagination)")

		// Fetch attacks up to currentTo timestamp
		attackResp, err := c.GetFactionAttacks(ctx, actualFromTime, currentTo)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch attacks for timeframe %d-%d: %w", actualFromTime, currentTo, err)
		}

		log.Debug().
			Int("attacks_in_page", len(attackResp.Attacks)).
			Msg("Received attacks from API")

		if len(attackResp.Attacks) == 0 {
			log.Debug().Msg("No more attacks returned, stopping pagination")
			break
		}

		// Filter attacks to only include those involving war participants
		var relevantAttacks []app.Attack
		var oldestAttackTime int64 = currentTo
		
		for _, attack := range attackResp.Attacks {
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

		allAttacks = append(allAttacks, relevantAttacks...)

		// If we got less than 100 attacks (the typical page size), we've reached the end
		if len(attackResp.Attacks) < 100 {
			log.Debug().
				Int("attacks_received", len(attackResp.Attacks)).
				Msg("Received less than full page, stopping pagination")
			break
		}

		// If the oldest attack is before or at our fetch start time, we're done
		if oldestAttackTime <= actualFromTime {
			log.Debug().
				Int64("oldest_attack", oldestAttackTime).
				Int64("fetch_start", actualFromTime).
				Msg("Reached fetch start time, stopping pagination")
			break
		}

		// Set up next page: use oldest attack time minus 1 second as new "to" time
		currentTo = oldestAttackTime - 1
		
		log.Debug().
			Int64("next_to", currentTo).
			Str("next_to_str", time.Unix(currentTo, 0).Format("2006-01-02 15:04:05")).
			Int("total_attacks_so_far", len(allAttacks)).
			Msg("Preparing next pagination request")
	}

	// Sort all attacks chronologically (oldest first) for consistent sheet ordering
	sort.Slice(allAttacks, func(i, j int) bool {
		return allAttacks[i].Started < allAttacks[j].Started
	})

	log.Info().
		Int("total_relevant_attacks", len(allAttacks)).
		Int("war_id", war.ID).
		Str("mode", updateMode+"_paginated").
		Msg("Completed fetching attacks for war")

	return allAttacks, nil
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