package torn

import (
	"context"

	"torn_rw_stats/internal/app"
)

// TornAPI defines the interface for interacting with the Torn API
// This separates infrastructure concerns from business logic
type TornAPI interface {
	// Core API endpoints
	GetFactionWars(ctx context.Context) (*app.WarResponse, error)
	GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error)
	GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error)
	GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error)

	// API call tracking
	GetAPICallCount() int64
	IncrementAPICall()
	ResetAPICallCount()
}