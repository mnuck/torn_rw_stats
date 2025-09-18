package torn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
