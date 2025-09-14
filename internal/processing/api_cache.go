package processing

import (
	"context"
	"sync"
	"time"

	"torn_rw_stats/internal/app"

	"github.com/rs/zerolog/log"
)

// APICacheConfig configures caching behavior
type APICacheConfig struct {
	// FactionInfoTTL is how long to cache faction info (rarely changes)
	FactionInfoTTL time.Duration
	// WarsTTL is how long to cache war list (changes infrequently during active wars)
	WarsTTL time.Duration
	// FactionBasicTTL is how long to cache faction member data
	FactionBasicTTL time.Duration
	// WarStateManager provides context for war-state-aware caching
	WarStateManager WarStateManagerInterface
}

// DefaultAPICacheConfig returns sensible cache defaults
func DefaultAPICacheConfig() APICacheConfig {
	return APICacheConfig{
		FactionInfoTTL:  30 * time.Minute, // Faction info rarely changes
		WarsTTL:         2 * time.Minute,  // Wars can start/end but not frequently
		FactionBasicTTL: 5 * time.Minute,  // Member data changes but not constantly
		WarStateManager: nil,              // Must be set separately
	}
}

// NewAPICacheConfigWithWarStateManager creates cache config with war state awareness
func NewAPICacheConfigWithWarStateManager(wsm WarStateManagerInterface) APICacheConfig {
	config := DefaultAPICacheConfig()
	config.WarStateManager = wsm
	return config
}

// CachedTornClient wraps a TornClient with intelligent caching
type CachedTornClient struct {
	client  TornClientInterface
	config  APICacheConfig
	tracker *APICallTracker
	mutex   sync.RWMutex

	// Cache entries
	factionInfo      *cachedFactionInfo
	wars             *cachedWars
	factionBasicData map[int]*cachedFactionBasic
}

type cachedFactionInfo struct {
	data      *app.FactionInfoResponse
	timestamp time.Time
}

type cachedWars struct {
	data      *app.WarResponse
	timestamp time.Time
}

type cachedFactionBasic struct {
	data      *app.FactionBasicResponse
	timestamp time.Time
}

// NewCachedTornClient creates a caching wrapper around a TornClient
func NewCachedTornClient(client TornClientInterface, tracker *APICallTracker) *CachedTornClient {
	return &CachedTornClient{
		client:           client,
		config:           DefaultAPICacheConfig(),
		tracker:          tracker,
		factionBasicData: make(map[int]*cachedFactionBasic),
	}
}

// NewCachedTornClientWithWarStateManager creates a caching wrapper with war-state-aware caching
func NewCachedTornClientWithWarStateManager(client TornClientInterface, tracker *APICallTracker, wsm WarStateManagerInterface) *CachedTornClient {
	return &CachedTornClient{
		client:           client,
		config:           NewAPICacheConfigWithWarStateManager(wsm),
		tracker:          tracker,
		factionBasicData: make(map[int]*cachedFactionBasic),
	}
}

// GetOwnFaction returns cached faction info or fetches fresh data with war-state-aware caching
func (c *CachedTornClient) GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error) {
	c.mutex.RLock()
	cached := c.factionInfo
	c.mutex.RUnlock()

	// Determine cache TTL based on war state (faction locking during wars)
	cacheTTL := c.getFactionInfoCacheTTL()

	// Check if cache is valid
	if cached != nil && time.Since(cached.timestamp) < cacheTTL {
		log.Debug().
			Dur("cache_age", time.Since(cached.timestamp)).
			Dur("cache_ttl", cacheTTL).
			Str("reason", c.getFactionCacheReason()).
			Msg("Using cached faction info (API call saved)")
		return cached.data, nil
	}

	// Cache miss or expired - fetch fresh data
	log.Debug().
		Dur("cache_ttl", cacheTTL).
		Str("reason", c.getFactionCacheReason()).
		Msg("Fetching fresh faction info from API")
	data, err := c.client.GetOwnFaction(ctx)
	if err != nil {
		return nil, err
	}

	c.tracker.RecordCall("GetOwnFaction")

	// Update cache
	c.mutex.Lock()
	c.factionInfo = &cachedFactionInfo{
		data:      data,
		timestamp: time.Now(),
	}
	c.mutex.Unlock()

	return data, nil
}

// GetFactionWars returns cached war data or fetches fresh data
func (c *CachedTornClient) GetFactionWars(ctx context.Context) (*app.WarResponse, error) {
	c.mutex.RLock()
	cached := c.wars
	c.mutex.RUnlock()

	// Check if cache is valid
	if cached != nil && time.Since(cached.timestamp) < c.config.WarsTTL {
		log.Debug().
			Dur("cache_age", time.Since(cached.timestamp)).
			Msg("Using cached war data (API call saved)")
		return cached.data, nil
	}

	// Cache miss or expired - fetch fresh data
	log.Debug().Msg("Fetching fresh war data from API")
	data, err := c.client.GetFactionWars(ctx)
	if err != nil {
		return nil, err
	}

	c.tracker.RecordCall("GetFactionWars")

	// Update cache
	c.mutex.Lock()
	c.wars = &cachedWars{
		data:      data,
		timestamp: time.Now(),
	}
	c.mutex.Unlock()

	return data, nil
}

// GetAllAttacksForWar delegates to underlying client (no caching for dynamic data)
func (c *CachedTornClient) GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error) {
	c.tracker.RecordCall("GetAllAttacksForWar")
	return c.client.GetAllAttacksForWar(ctx, war)
}

// GetAttacksForTimeRange delegates to underlying client (no caching for dynamic data)
func (c *CachedTornClient) GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error) {
	c.tracker.RecordCall("GetAttacksForTimeRange")
	return c.client.GetAttacksForTimeRange(ctx, war, fromTime, latestExistingTimestamp)
}

// GetFactionBasic returns cached faction member data or fetches fresh data
func (c *CachedTornClient) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	c.mutex.RLock()
	cached := c.factionBasicData[factionID]
	c.mutex.RUnlock()

	// Check if cache is valid
	if cached != nil && time.Since(cached.timestamp) < c.config.FactionBasicTTL {
		log.Debug().
			Int("faction_id", factionID).
			Dur("cache_age", time.Since(cached.timestamp)).
			Msg("Using cached faction basic data (API call saved)")
		return cached.data, nil
	}

	// Cache miss or expired - fetch fresh data
	log.Debug().
		Int("faction_id", factionID).
		Msg("Fetching fresh faction basic data from API")
	data, err := c.client.GetFactionBasic(ctx, factionID)
	if err != nil {
		return nil, err
	}

	c.tracker.RecordCall("GetFactionBasic")

	// Update cache
	c.mutex.Lock()
	c.factionBasicData[factionID] = &cachedFactionBasic{
		data:      data,
		timestamp: time.Now(),
	}
	c.mutex.Unlock()

	return data, nil
}

// ClearCache invalidates all cached data
func (c *CachedTornClient) ClearCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.factionInfo = nil
	c.wars = nil
	c.factionBasicData = make(map[int]*cachedFactionBasic)

	log.Info().Msg("API cache cleared")
}

// GetCacheStats returns cache hit/miss statistics
func (c *CachedTornClient) GetCacheStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var validEntries, expiredEntries int

	if c.factionInfo != nil {
		if time.Since(c.factionInfo.timestamp) < c.config.FactionInfoTTL {
			validEntries++
		} else {
			expiredEntries++
		}
	}

	if c.wars != nil {
		if time.Since(c.wars.timestamp) < c.config.WarsTTL {
			validEntries++
		} else {
			expiredEntries++
		}
	}

	for _, cached := range c.factionBasicData {
		if time.Since(cached.timestamp) < c.config.FactionBasicTTL {
			validEntries++
		} else {
			expiredEntries++
		}
	}

	return CacheStats{
		ValidEntries:   validEntries,
		ExpiredEntries: expiredEntries,
		TotalEntries:   validEntries + expiredEntries,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	ValidEntries   int
	ExpiredEntries int
	TotalEntries   int
}

// getFactionInfoCacheTTL returns appropriate cache TTL based on war state
func (c *CachedTornClient) getFactionInfoCacheTTL() time.Duration {
	// If no war state manager, use default TTL
	if c.config.WarStateManager == nil {
		return c.config.FactionInfoTTL
	}

	state := c.config.WarStateManager.GetCurrentState()
	currentWar := c.config.WarStateManager.GetCurrentWar()

	switch state {
	case PreWar, ActiveWar:
		// Faction is locked during wars - cache until war ends + buffer
		if currentWar != nil && currentWar.End != nil {
			warEnd := time.Unix(*currentWar.End, 0)
			timeUntilWarEnd := time.Until(warEnd)
			if timeUntilWarEnd > 0 {
				// Cache until war end + 1 hour buffer
				return timeUntilWarEnd + time.Hour
			}
		}
		// If no end time or war already ended, use extended cache
		return 4 * time.Hour // Extended cache for locked faction
	case NoWars, PostWar:
		// Faction can change - use default TTL
		return c.config.FactionInfoTTL
	default:
		return c.config.FactionInfoTTL
	}
}

// getFactionCacheReason returns human-readable reason for current cache strategy
func (c *CachedTornClient) getFactionCacheReason() string {
	if c.config.WarStateManager == nil {
		return "no_war_state_manager"
	}

	state := c.config.WarStateManager.GetCurrentState()
	switch state {
	case PreWar:
		return "faction_locked_prewar"
	case ActiveWar:
		return "faction_locked_activewar"
	case NoWars:
		return "faction_unlocked_nowars"
	case PostWar:
		return "faction_unlocked_postwar"
	default:
		return "unknown_state"
	}
}
