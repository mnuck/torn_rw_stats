package mocks

import (
	"context"

	"torn_rw_stats/internal/app"
)

// TornClient interface defines the methods used by WarProcessor from torn.Client
type TornClient interface {
	GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error)
	GetFactionWars(ctx context.Context) (*app.WarResponse, error)
	GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error)
	GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error)
	GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error)
}

// MockTornClient is a test double for the torn.Client
type MockTornClient struct {
	// Responses to return
	OwnFactionResponse          *app.FactionInfoResponse
	FactionWarsResponse         *app.WarResponse
	AllAttacksForWarResponse    []app.Attack
	AttacksForTimeRangeResponse []app.Attack
	FactionBasicResponse        *app.FactionBasicResponse

	// Errors to return
	OwnFactionError          error
	FactionWarsError         error
	AllAttacksForWarError    error
	AttacksForTimeRangeError error
	FactionBasicError        error

	// Call tracking
	GetOwnFactionCalled              bool
	GetFactionWarsCalled             bool
	GetAllAttacksForWarCalled        bool
	GetAttacksForTimeRangeCalled     bool
	GetFactionBasicCalled            bool
	GetFactionBasicCalledWithID      int
	GetAllAttacksForWarCalledWith    *app.War
	GetAttacksForTimeRangeCalledWith struct {
		War                     *app.War
		FromTime                int64
		LatestExistingTimestamp *int64
	}
}

// NewMockTornClient creates a new mock torn client
func NewMockTornClient() *MockTornClient {
	return &MockTornClient{}
}

func (m *MockTornClient) GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error) {
	m.GetOwnFactionCalled = true
	return m.OwnFactionResponse, m.OwnFactionError
}

func (m *MockTornClient) GetFactionWars(ctx context.Context) (*app.WarResponse, error) {
	m.GetFactionWarsCalled = true
	return m.FactionWarsResponse, m.FactionWarsError
}

func (m *MockTornClient) GetAllAttacksForWar(ctx context.Context, war *app.War) ([]app.Attack, error) {
	m.GetAllAttacksForWarCalled = true
	m.GetAllAttacksForWarCalledWith = war
	return m.AllAttacksForWarResponse, m.AllAttacksForWarError
}

func (m *MockTornClient) GetAttacksForTimeRange(ctx context.Context, war *app.War, fromTime int64, latestExistingTimestamp *int64) ([]app.Attack, error) {
	m.GetAttacksForTimeRangeCalled = true
	m.GetAttacksForTimeRangeCalledWith.War = war
	m.GetAttacksForTimeRangeCalledWith.FromTime = fromTime
	m.GetAttacksForTimeRangeCalledWith.LatestExistingTimestamp = latestExistingTimestamp
	return m.AttacksForTimeRangeResponse, m.AttacksForTimeRangeError
}

func (m *MockTornClient) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	m.GetFactionBasicCalled = true
	m.GetFactionBasicCalledWithID = factionID
	return m.FactionBasicResponse, m.FactionBasicError
}

// Reset clears all call tracking and responses
func (m *MockTornClient) Reset() {
	m.OwnFactionResponse = nil
	m.FactionWarsResponse = nil
	m.AllAttacksForWarResponse = nil
	m.AttacksForTimeRangeResponse = nil
	m.FactionBasicResponse = nil

	m.OwnFactionError = nil
	m.FactionWarsError = nil
	m.AllAttacksForWarError = nil
	m.AttacksForTimeRangeError = nil
	m.FactionBasicError = nil

	m.GetOwnFactionCalled = false
	m.GetFactionWarsCalled = false
	m.GetAllAttacksForWarCalled = false
	m.GetAttacksForTimeRangeCalled = false
	m.GetFactionBasicCalled = false
	m.GetFactionBasicCalledWithID = 0
	m.GetAllAttacksForWarCalledWith = nil
	m.GetAttacksForTimeRangeCalledWith = struct {
		War                     *app.War
		FromTime                int64
		LatestExistingTimestamp *int64
	}{}
}
