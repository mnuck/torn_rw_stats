package torn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"log/slog"

	"torn_rw_stats/internal/app"
)

const (
	// HTTP client configuration
	HTTPClientTimeout = 30 * time.Second
)

// Client is an HTTP client for the Torn API that handles authentication,
// request formatting, and API call tracking.
type Client struct {
	apiKey       string
	client       *http.Client
	apiCallCount int64
	apiCallMutex sync.Mutex
}

// NewClient creates a new Torn API client with the provided API key.
// The client is configured with a 30-second timeout for all requests.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: HTTPClientTimeout,
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
		slog.Debug("API request failed",
			"err", err,
			"url", url)
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

	slog.Debug("Fetching faction wars", "url", url)

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

	slog.Debug("Successfully fetched faction wars",
		"has_ranked_war", warResponse.Wars.Ranked != nil,
		"raid_wars", len(warResponse.Wars.Raids),
		"territory_wars", len(warResponse.Wars.Territory))

	return &warResponse, nil
}

// GetFactionAttacks fetches faction attacks from the API using timestamp pagination
func (c *Client) GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/v2/faction/attacks?key=%s&from=%d&to=%d", c.apiKey, from, to)

	slog.Debug("Fetching faction attacks",
		"url", url,
		"from", from,
		"to", to,
		"from_time", time.Unix(from, 0).Format("2006-01-02 15:04:05"),
		"to_time", time.Unix(to, 0).Format("2006-01-02 15:04:05"))

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

	slog.Debug("Successfully fetched faction attacks",
		"attacks_count", len(attackResponse.Attacks),
		"from", from,
		"to", to)

	return &attackResponse, nil
}

// GetFactionBasic fetches faction basic data from the API
func (c *Client) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/faction/%d?selections=basic&key=%s", factionID, c.apiKey)

	slog.Debug("Fetching faction basic data",
		"url", url,
		"faction_id", factionID)

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

	slog.Debug("Successfully fetched faction basic data",
		"faction_id", factionID,
		"members_count", len(factionResponse.Members))

	return &factionResponse, nil
}

// GetOwnFaction gets the current user's faction information
func (c *Client) GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error) {
	url := fmt.Sprintf("https://api.torn.com/faction/?selections=basic&key=%s", c.apiKey)

	slog.Debug("Fetching own faction data", "url", url)

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

	slog.Debug("Successfully fetched own faction data",
		"faction_id", factionResponse.ID,
		"faction_name", factionResponse.Name,
		"faction_tag", factionResponse.Tag)

	return &factionResponse, nil
}
