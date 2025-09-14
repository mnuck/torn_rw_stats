package processing

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/processing/mocks"
)

// MockWarStateManager implements WarStateManagerInterface for testing
type MockWarStateManager struct {
	currentState WarState
	currentWar   *app.War
}

func (m *MockWarStateManager) GetCurrentState() WarState {
	return m.currentState
}

func (m *MockWarStateManager) GetCurrentWar() *app.War {
	return m.currentWar
}

func TestFactionLockingOptimization(t *testing.T) {
	t.Run("NoWars_UseDefaultTTL", func(t *testing.T) {
		mockClient := &mocks.MockTornClient{}
		tracker := NewAPICallTracker()
		mockWSM := &MockWarStateManager{
			currentState: NoWars,
			currentWar:   nil,
		}

		cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
		ctx := context.Background()

		// Set up mock response
		factionResponse := &app.FactionInfoResponse{
			ID:   12345,
			Name: "Test Faction",
			Tag:  "TEST",
		}
		mockClient.OwnFactionResponse = factionResponse

		// First call should hit API
		result1, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result1.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result1.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call, got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}

		// Second call within 30 minutes should use cache
		result2, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result2.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result2.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call (cached), got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}
	})

	t.Run("PreWar_ExtendedCaching", func(t *testing.T) {
		mockClient := &mocks.MockTornClient{}
		tracker := NewAPICallTracker()

		// War ending in 2 hours
		warEndTime := time.Now().Add(2 * time.Hour).Unix()
		mockWSM := &MockWarStateManager{
			currentState: PreWar,
			currentWar: &app.War{
				ID:    67890,
				Start: time.Now().Add(30 * time.Minute).Unix(),
				End:   &warEndTime,
			},
		}

		cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
		ctx := context.Background()

		// Set up mock response
		factionResponse := &app.FactionInfoResponse{
			ID:   12345,
			Name: "Test Faction",
			Tag:  "TEST",
		}
		mockClient.OwnFactionResponse = factionResponse

		// First call should hit API
		result1, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result1.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result1.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call, got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}

		// Multiple calls should all use cache (faction locked during PreWar)
		for i := 0; i < 5; i++ {
			result, err := cachedClient.GetOwnFaction(ctx)
			if err != nil {
				t.Fatalf("Call %d: Expected no error, got %v", i+1, err)
			}
			if result.ID != 12345 {
				t.Errorf("Call %d: Expected faction ID 12345, got %d", i+1, result.ID)
			}
		}

		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call total (all others cached during PreWar), got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}
	})

	t.Run("ActiveWar_ExtendedCaching", func(t *testing.T) {
		mockClient := &mocks.MockTornClient{}
		tracker := NewAPICallTracker()

		// Active war ending in 3 hours
		warEndTime := time.Now().Add(3 * time.Hour).Unix()
		mockWSM := &MockWarStateManager{
			currentState: ActiveWar,
			currentWar: &app.War{
				ID:    67890,
				Start: time.Now().Add(-30 * time.Minute).Unix(), // Started 30 min ago
				End:   &warEndTime,
			},
		}

		cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
		ctx := context.Background()

		// Set up mock response
		factionResponse := &app.FactionInfoResponse{
			ID:   12345,
			Name: "Test Faction",
			Tag:  "TEST",
		}
		mockClient.OwnFactionResponse = factionResponse

		// First call should hit API
		result1, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result1.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result1.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call, got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}

		// Multiple calls should all use cache (faction locked during ActiveWar)
		for i := 0; i < 10; i++ {
			result, err := cachedClient.GetOwnFaction(ctx)
			if err != nil {
				t.Fatalf("Call %d: Expected no error, got %v", i+1, err)
			}
			if result.ID != 12345 {
				t.Errorf("Call %d: Expected faction ID 12345, got %d", i+1, result.ID)
			}
		}

		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call total (all others cached during ActiveWar), got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}
	})

	t.Run("PostWar_UseDefaultTTL", func(t *testing.T) {
		mockClient := &mocks.MockTornClient{}
		tracker := NewAPICallTracker()

		// Recently ended war
		warEndTime := time.Now().Add(-10 * time.Minute).Unix() // Ended 10 min ago
		mockWSM := &MockWarStateManager{
			currentState: PostWar,
			currentWar: &app.War{
				ID:    67890,
				Start: time.Now().Add(-2 * time.Hour).Unix(),
				End:   &warEndTime,
			},
		}

		cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
		ctx := context.Background()

		// Set up mock response
		factionResponse := &app.FactionInfoResponse{
			ID:   12345,
			Name: "Test Faction",
			Tag:  "TEST",
		}
		mockClient.OwnFactionResponse = factionResponse

		// First call should hit API
		result1, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result1.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result1.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call, got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}

		// Second call within default TTL should use cache
		result2, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result2.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result2.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call (cached), got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}
	})

	t.Run("WarStateTransition_CacheStillValid", func(t *testing.T) {
		mockClient := &mocks.MockTornClient{}
		tracker := NewAPICallTracker()

		// Start in NoWars state
		mockWSM := &MockWarStateManager{
			currentState: NoWars,
			currentWar:   nil,
		}

		cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
		ctx := context.Background()

		// Set up mock response
		factionResponse := &app.FactionInfoResponse{
			ID:   12345,
			Name: "Test Faction",
			Tag:  "TEST",
		}
		mockClient.OwnFactionResponse = factionResponse

		// First call in NoWars state
		result1, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result1.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result1.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call, got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}

		// Transition to PreWar state with war ending in 2 hours
		warEndTime := time.Now().Add(2 * time.Hour).Unix()
		mockWSM.currentState = PreWar
		mockWSM.currentWar = &app.War{
			ID:    67890,
			Start: time.Now().Add(30 * time.Minute).Unix(),
			End:   &warEndTime,
		}

		// Call in PreWar state should still use cache (cache duration extended)
		result2, err := cachedClient.GetOwnFaction(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result2.ID != 12345 {
			t.Errorf("Expected faction ID 12345, got %d", result2.ID)
		}
		if tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"] != 1 {
			t.Errorf("Expected 1 API call (cache should still be valid after state transition), got %d", tracker.GetSessionStats().CallsByEndpoint["GetOwnFaction"])
		}
	})
}

func TestFactionCacheReasonLogging(t *testing.T) {
	mockClient := &mocks.MockTornClient{}
	tracker := NewAPICallTracker()

	testCases := []struct {
		name         string
		state        WarState
		expectedReason string
	}{
		{"NoWars", NoWars, "faction_unlocked_nowars"},
		{"PreWar", PreWar, "faction_locked_prewar"},
		{"ActiveWar", ActiveWar, "faction_locked_activewar"},
		{"PostWar", PostWar, "faction_unlocked_postwar"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWSM := &MockWarStateManager{
				currentState: tc.state,
				currentWar:   nil,
			}

			cachedClient := NewCachedTornClientWithWarStateManager(mockClient, tracker, mockWSM)
			reason := cachedClient.getFactionCacheReason()

			if reason != tc.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tc.expectedReason, reason)
			}
		})
	}

	t.Run("NoWarStateManager", func(t *testing.T) {
		cachedClient := NewCachedTornClient(mockClient, tracker)
		reason := cachedClient.getFactionCacheReason()

		if reason != "no_war_state_manager" {
			t.Errorf("Expected reason 'no_war_state_manager', got '%s'", reason)
		}
	})
}