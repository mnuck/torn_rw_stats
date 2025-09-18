package mocks

import (
	"context"

	"torn_rw_stats/internal/app"
)

// TornClient interface defines the methods used by WarProcessor from torn.Client
type TornClient interface {
	GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error)
	GetFactionWars(ctx context.Context) (*app.WarResponse, error)
	GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error)
	GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error)
	GetAPICallCount() int64
	IncrementAPICall()
	ResetAPICallCount()
}

// MockTornClient is a test double for the torn.Client
type MockTornClient struct {
	// Responses to return
	OwnFactionResponse     *app.FactionInfoResponse
	FactionWarsResponse    *app.WarResponse
	FactionAttacksResponse *app.AttackResponse
	FactionBasicResponse   *app.FactionBasicResponse
	APICallCount           int64

	// Errors to return
	OwnFactionError     error
	FactionWarsError    error
	FactionAttacksError error
	FactionBasicError   error

	// Call tracking
	GetOwnFactionCalled         bool
	GetFactionWarsCalled        bool
	GetFactionAttacksCalled     bool
	GetFactionBasicCalled       bool
	GetFactionBasicCalledWithID int
	GetFactionAttacksCalledWith struct {
		From int64
		To   int64
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

func (m *MockTornClient) GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error) {
	m.GetFactionAttacksCalled = true
	m.GetFactionAttacksCalledWith.From = from
	m.GetFactionAttacksCalledWith.To = to
	return m.FactionAttacksResponse, m.FactionAttacksError
}

func (m *MockTornClient) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	m.GetFactionBasicCalled = true
	m.GetFactionBasicCalledWithID = factionID
	return m.FactionBasicResponse, m.FactionBasicError
}

func (m *MockTornClient) GetAPICallCount() int64 {
	return m.APICallCount
}

func (m *MockTornClient) IncrementAPICall() {
	m.APICallCount++
}

func (m *MockTornClient) ResetAPICallCount() {
	m.APICallCount = 0
}

// Reset clears all call tracking and responses
func (m *MockTornClient) Reset() {
	m.OwnFactionResponse = nil
	m.FactionWarsResponse = nil
	m.FactionAttacksResponse = nil
	m.FactionBasicResponse = nil
	m.APICallCount = 0

	m.OwnFactionError = nil
	m.FactionWarsError = nil
	m.FactionAttacksError = nil
	m.FactionBasicError = nil

	m.GetOwnFactionCalled = false
	m.GetFactionWarsCalled = false
	m.GetFactionAttacksCalled = false
	m.GetFactionBasicCalled = false
	m.GetFactionBasicCalledWithID = 0
	m.GetFactionAttacksCalledWith = struct {
		From int64
		To   int64
	}{}
}
