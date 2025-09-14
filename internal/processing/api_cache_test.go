package processing

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

func TestCachedTornClient_GetFactionBasic(t *testing.T) {
	mockClient := &mocks.MockTornClient{}
	tracker := NewAPICallTracker()
	cachedClient := NewCachedTornClient(mockClient, tracker)

	// Modify the TTL for testing
	cachedClient.config.FactionBasicTTL = 5 * time.Minute

	ctx := context.Background()
	factionID := 12345

	expectedResponse := &app.FactionBasicResponse{
		Members: map[string]app.FactionMember{
			"123": {Name: "TestPlayer", Level: 50},
		},
	}

	// Set up mock to return our expected response
	mockClient.FactionBasicResponse = expectedResponse

	// First call should hit the API
	result1, err := cachedClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result1 == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result1.Members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(result1.Members))
	}

	// Verify API call was made
	if !mockClient.GetFactionBasicCalled {
		t.Error("Expected API call to be made")
	}

	if mockClient.GetFactionBasicCalledWithID != factionID {
		t.Errorf("Expected API call with faction ID %d, got %d", factionID, mockClient.GetFactionBasicCalledWithID)
	}

	// Reset call tracking for second test
	mockClient.GetFactionBasicCalled = false

	// Second call should use cache
	result2, err := cachedClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result2 == nil {
		t.Fatal("Expected result, got nil")
	}

	// API call should not have been made (cached)
	if mockClient.GetFactionBasicCalled {
		t.Error("Expected no API call due to cache")
	}

	// Results should be identical
	if len(result1.Members) != len(result2.Members) {
		t.Error("Cached result differs from original")
	}
}

func TestCachedTornClient_GetFactionBasic_CacheExpiry(t *testing.T) {
	mockClient := &mocks.MockTornClient{}
	tracker := NewAPICallTracker()
	cachedClient := NewCachedTornClient(mockClient, tracker)

	// Set very short TTL for testing
	cachedClient.config.FactionBasicTTL = 10 * time.Millisecond

	ctx := context.Background()
	factionID := 12345

	expectedResponse := &app.FactionBasicResponse{
		Members: map[string]app.FactionMember{
			"123": {Name: "TestPlayer", Level: 50},
		},
	}

	mockClient.FactionBasicResponse = expectedResponse

	// First call
	_, err := cachedClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Wait for cache to expire
	time.Sleep(15 * time.Millisecond)

	// Second call should hit API again due to expiry
	_, err = cachedClient.GetFactionBasic(ctx, factionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have made second API call due to cache expiry
	if !mockClient.GetFactionBasicCalled {
		t.Error("Expected second API call due to cache expiry")
	}
}

func TestCachedTornClient_ClearCache(t *testing.T) {
	mockClient := &mocks.MockTornClient{}
	tracker := NewAPICallTracker()
	cachedClient := NewCachedTornClient(mockClient, tracker)

	ctx := context.Background()

	// Set up some cache data first
	expectedWars := &app.WarResponse{
		Wars: struct {
			Ranked    *app.War  `json:"ranked"`
			Raids     []app.War `json:"raids"`
			Territory []app.War `json:"territory"`
		}{
			Ranked: &app.War{ID: 123, Target: 456},
		},
	}

	expectedFaction := &app.FactionInfoResponse{
		ID:   789,
		Name: "TestFaction",
		Members: map[string]app.FactionMember{
			"1": {Name: "Player1", Level: 30},
		},
	}

	mockClient.FactionWarsResponse = expectedWars
	mockClient.OwnFactionResponse = expectedFaction

	// Populate cache
	_, _ = cachedClient.GetFactionWars(ctx)
	_, _ = cachedClient.GetOwnFaction(ctx)

	// Verify cache is populated (should not make new API calls)
	mockClient.GetFactionWarsCalled = false
	mockClient.GetOwnFactionCalled = false

	_, _ = cachedClient.GetFactionWars(ctx)
	_, _ = cachedClient.GetOwnFaction(ctx)

	if mockClient.GetFactionWarsCalled {
		t.Error("Expected cached faction wars call")
	}
	if mockClient.GetOwnFactionCalled {
		t.Error("Expected cached own faction call")
	}

	// Clear cache
	cachedClient.ClearCache()

	// After clearing, should make API calls again
	_, _ = cachedClient.GetFactionWars(ctx)
	_, _ = cachedClient.GetOwnFaction(ctx)

	if !mockClient.GetFactionWarsCalled {
		t.Error("Expected faction wars call after cache clear")
	}
	if !mockClient.GetOwnFactionCalled {
		t.Error("Expected own faction call after cache clear")
	}
}
