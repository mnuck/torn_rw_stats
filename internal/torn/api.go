package torn

import (
	"context"

	"torn_rw_stats/internal/domain"
)

// TornAPI defines the interface for interacting with the Torn API
// This separates infrastructure concerns from business logic
type TornAPI interface {
	// Core API endpoints
	GetFactionWars(ctx context.Context) (*domain.FactionWars, error)
	GetFactionAttacks(ctx context.Context, from, to int64) ([]domain.Attack, error)
	GetFactionBasic(ctx context.Context, factionID int) (*domain.FactionInfo, error)
	GetOwnFaction(ctx context.Context) (*domain.FactionInfo, error)

	// API call tracking
	GetAPICallCount() int64
	IncrementAPICall()
	ResetAPICallCount()
}
