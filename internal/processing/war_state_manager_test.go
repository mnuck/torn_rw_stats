package processing

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

// TestWarStateDetection tests the logic for determining war states
func TestWarStateDetection(t *testing.T) {
	wsm := NewWarStateManager()
	now := time.Now()

	t.Run("NoWars", func(t *testing.T) {
		emptyResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{},
		}

		state := wsm.UpdateState(emptyResponse)
		if state != NoWars {
			t.Errorf("Expected NoWars, got %s", state.String())
		}
	})

	t.Run("ActiveWar", func(t *testing.T) {
		// War that started 30 minutes ago
		activeWarResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: now.Add(-30 * time.Minute).Unix(),
					End:   nil, // Still active
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy Faction"},
					},
				},
			},
		}

		state := wsm.UpdateState(activeWarResponse)
		if state != ActiveWar {
			t.Errorf("Expected ActiveWar, got %s", state.String())
		}

		if wsm.GetCurrentWar() == nil {
			t.Error("Expected current war to be set")
		}
	})

	t.Run("PreWar", func(t *testing.T) {
		// War scheduled to start in 2 hours
		preWarResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Raids: []app.War{{
					ID:    12346,
					Start: now.Add(2 * time.Hour).Unix(),
					End:   nil,
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1003, Name: "Enemy Faction"},
					},
				}},
			},
		}

		state := wsm.UpdateState(preWarResponse)
		if state != PreWar {
			t.Errorf("Expected PreWar, got %s", state.String())
		}
	})

	t.Run("PostWar", func(t *testing.T) {
		// War that ended 30 minutes ago
		endTime := now.Add(-30 * time.Minute).Unix()
		postWarResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12347,
					Start: now.Add(-2 * time.Hour).Unix(),
					End:   &endTime,
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1004, Name: "Enemy Faction"},
					},
				},
			},
		}

		state := wsm.UpdateState(postWarResponse)
		if state != PostWar {
			t.Errorf("Expected PostWar, got %s", state.String())
		}
	})
}

// TestTuesdayMatchmakingLogic tests the Tuesday matchmaking time calculations
func TestTuesdayMatchmakingLogic(t *testing.T) {
	wsm := NewWarStateManager()

	testCases := []struct {
		name        string
		currentTime time.Time
		expected    time.Time
	}{
		{
			name:        "Monday before matchmaking",
			currentTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), // Monday 10:00 UTC
			expected:    time.Date(2024, 1, 2, 12, 5, 0, 0, time.UTC), // Tuesday 12:05 UTC
		},
		{
			name:        "Tuesday before matchmaking",
			currentTime: time.Date(2024, 1, 2, 11, 0, 0, 0, time.UTC), // Tuesday 11:00 UTC
			expected:    time.Date(2024, 1, 2, 12, 5, 0, 0, time.UTC), // Same Tuesday 12:05 UTC
		},
		{
			name:        "Tuesday after matchmaking",
			currentTime: time.Date(2024, 1, 2, 13, 0, 0, 0, time.UTC), // Tuesday 13:00 UTC
			expected:    time.Date(2024, 1, 9, 12, 5, 0, 0, time.UTC), // Next Tuesday 12:05 UTC
		},
		{
			name:        "Wednesday",
			currentTime: time.Date(2024, 1, 3, 15, 0, 0, 0, time.UTC), // Wednesday 15:00 UTC
			expected:    time.Date(2024, 1, 9, 12, 5, 0, 0, time.UTC), // Next Tuesday 12:05 UTC
		},
		{
			name:        "Sunday",
			currentTime: time.Date(2024, 1, 7, 20, 0, 0, 0, time.UTC), // Sunday 20:00 UTC
			expected:    time.Date(2024, 1, 9, 12, 5, 0, 0, time.UTC), // Next Tuesday 12:05 UTC
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := wsm.getNextTuesdayMatchmaking(tc.currentTime)
			if !result.Equal(tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}

			// Verify it's always a Tuesday at 12:05 UTC
			if result.Weekday() != time.Tuesday {
				t.Errorf("Expected Tuesday, got %s", result.Weekday())
			}
			if result.Hour() != 12 || result.Minute() != 5 {
				t.Errorf("Expected 12:05, got %02d:%02d", result.Hour(), result.Minute())
			}
			if result.Location() != time.UTC {
				t.Error("Expected UTC timezone")
			}
		})
	}
}

// TestStateTransitions tests state transitions and their timing
func TestStateTransitions(t *testing.T) {
	wsm := NewWarStateManager()
	now := time.Now()

	// Start with no wars
	emptyResponse := &app.WarResponse{
		Wars: struct {
			Ranked    *app.War    `json:"ranked"`
			Raids     []app.War   `json:"raids"`
			Territory []app.War   `json:"territory"`
		}{},
	}

	state := wsm.UpdateState(emptyResponse)
	if state != NoWars {
		t.Errorf("Expected initial state NoWars, got %s", state.String())
	}

	// Transition to PreWar
	preWarResponse := &app.WarResponse{
		Wars: struct {
			Ranked    *app.War    `json:"ranked"`
			Raids     []app.War   `json:"raids"`
			Territory []app.War   `json:"territory"`
		}{
			Ranked: &app.War{
				ID:    12345,
				Start: now.Add(1 * time.Hour).Unix(),
				Factions: []app.Faction{
					{ID: 1001, Name: "Our Faction"},
					{ID: 1002, Name: "Enemy Faction"},
				},
			},
		},
	}

	state = wsm.UpdateState(preWarResponse)
	if state != PreWar {
		t.Errorf("Expected PreWar after transition, got %s", state.String())
	}

	// Simulate time passing - war starts
	activeWarResponse := &app.WarResponse{
		Wars: struct {
			Ranked    *app.War    `json:"ranked"`
			Raids     []app.War   `json:"raids"`
			Territory []app.War   `json:"territory"`
		}{
			Ranked: &app.War{
				ID:    12345,
				Start: now.Add(-5 * time.Minute).Unix(), // Started 5 minutes ago
				Factions: []app.Faction{
					{ID: 1001, Name: "Our Faction"},
					{ID: 1002, Name: "Enemy Faction"},
				},
			},
		},
	}

	state = wsm.UpdateState(activeWarResponse)
	if state != ActiveWar {
		t.Errorf("Expected ActiveWar after war starts, got %s", state.String())
	}

	// War ends
	endTime := now.Add(-1 * time.Minute).Unix()
	postWarResponse := &app.WarResponse{
		Wars: struct {
			Ranked    *app.War    `json:"ranked"`
			Raids     []app.War   `json:"raids"`
			Territory []app.War   `json:"territory"`
		}{
			Ranked: &app.War{
				ID:    12345,
				Start: now.Add(-2 * time.Hour).Unix(),
				End:   &endTime,
				Factions: []app.Faction{
					{ID: 1001, Name: "Our Faction"},
					{ID: 1002, Name: "Enemy Faction"},
				},
			},
		},
	}

	state = wsm.UpdateState(postWarResponse)
	if state != PostWar {
		t.Errorf("Expected PostWar after war ends, got %s", state.String())
	}
}

// TestUpdateIntervals tests the different update intervals for each state
func TestUpdateIntervals(t *testing.T) {
	wsm := NewWarStateManager()

	testCases := []struct {
		state            WarState
		expectedInterval time.Duration
		expectedStrategy CheckStrategy
	}{
		{NoWars, 24 * time.Hour, UntilTuesdayMatchmaking}, // Placeholder interval
		{PreWar, 5 * time.Minute, FixedInterval},
		{ActiveWar, 1 * time.Minute, FixedInterval},
		{PostWar, 24 * time.Hour, UntilNextWeekMatchmaking}, // Placeholder interval
	}

	for _, tc := range testCases {
		t.Run(tc.state.String(), func(t *testing.T) {
			wsm.currentState = tc.state
			config := wsm.GetStateConfig()

			if config.UpdateInterval != tc.expectedInterval {
				t.Errorf("Expected interval %v, got %v", tc.expectedInterval, config.UpdateInterval)
			}

			if config.NextCheckStrategy != tc.expectedStrategy {
				t.Errorf("Expected strategy %v, got %v", tc.expectedStrategy, config.NextCheckStrategy)
			}
		})
	}
}

// TestEdgeCases tests edge cases and special scenarios
func TestEdgeCases(t *testing.T) {
	now := time.Now()

	t.Run("OverlappingWars", func(t *testing.T) {
		wsm := NewWarStateManager()
		// Test case where new matchmaking happens during active war
		// Should prioritize active war over potential new war
		overlappingResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: now.Add(-1 * time.Hour).Unix(), // Active war
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy A"},
					},
				},
				Raids: []app.War{{
					ID:    12346,
					Start: now.Add(30 * time.Minute).Unix(), // Future war
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1003, Name: "Enemy B"},
					},
				}},
			},
		}

		state := wsm.UpdateState(overlappingResponse)
		if state != ActiveWar {
			t.Errorf("Expected ActiveWar to take priority, got %s", state.String())
		}

		// Should track the active war, not the future one
		if wsm.GetCurrentWar().ID != 12345 {
			t.Errorf("Expected to track active war (12345), got %d", wsm.GetCurrentWar().ID)
		}
	})

	t.Run("MultipleActiveWars", func(t *testing.T) {
		wsm := NewWarStateManager()
		// Test case with multiple active wars - should choose most recent
		multiActiveResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: now.Add(-2 * time.Hour).Unix(), // Older active war
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy A"},
					},
				},
				Raids: []app.War{{
					ID:    12346,
					Start: now.Add(-1 * time.Hour).Unix(), // More recent active war
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1003, Name: "Enemy B"},
					},
				}},
			},
		}

		state := wsm.UpdateState(multiActiveResponse)
		if state != ActiveWar {
			t.Errorf("Expected ActiveWar, got %s", state.String())
		}

		// Should track the more recent war
		if wsm.GetCurrentWar().ID != 12346 {
			t.Errorf("Expected to track more recent war (12346), got %d", wsm.GetCurrentWar().ID)
		}
	})

	t.Run("MultiplePreWars", func(t *testing.T) {
		wsm := NewWarStateManager()
		// Test case with multiple pre-wars - should choose soonest
		multiPreResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Raids: []app.War{
					{
						ID:    12345,
						Start: now.Add(3 * time.Hour).Unix(), // Later pre-war
						Factions: []app.Faction{
							{ID: 1001, Name: "Our Faction"},
							{ID: 1002, Name: "Enemy A"},
						},
					},
					{
						ID:    12346,
						Start: now.Add(1 * time.Hour).Unix(), // Sooner pre-war
						Factions: []app.Faction{
							{ID: 1001, Name: "Our Faction"},
							{ID: 1003, Name: "Enemy B"},
						},
					},
				},
			},
		}

		state := wsm.UpdateState(multiPreResponse)
		if state != PreWar {
			t.Errorf("Expected PreWar, got %s", state.String())
		}

		// Should track the sooner war
		if wsm.GetCurrentWar().ID != 12346 {
			t.Errorf("Expected to track sooner war (12346), got %d", wsm.GetCurrentWar().ID)
		}
	})

	t.Run("VeryOldWar", func(t *testing.T) {
		wsm := NewWarStateManager()
		// War that ended hours ago should not be considered PostWar
		veryOldEndTime := now.Add(-3 * time.Hour).Unix()
		oldWarResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: now.Add(-5 * time.Hour).Unix(),
					End:   &veryOldEndTime,
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy Faction"},
					},
				},
			},
		}

		state := wsm.UpdateState(oldWarResponse)
		if state != NoWars {
			t.Errorf("Expected NoWars for very old war, got %s", state.String())
		}
	})

	t.Run("FarFutureWar", func(t *testing.T) {
		wsm := NewWarStateManager()
		// War scheduled way in the future should not be considered PreWar
		farFutureResponse := &app.WarResponse{
			Wars: struct {
				Ranked    *app.War    `json:"ranked"`
				Raids     []app.War   `json:"raids"`
				Territory []app.War   `json:"territory"`
			}{
				Ranked: &app.War{
					ID:    12345,
					Start: now.Add(10 * 24 * time.Hour).Unix(), // 10 days from now
					Factions: []app.Faction{
						{ID: 1001, Name: "Our Faction"},
						{ID: 1002, Name: "Enemy Faction"},
					},
				},
			},
		}

		state := wsm.UpdateState(farFutureResponse)
		if state != NoWars {
			t.Errorf("Expected NoWars for far future war, got %s", state.String())
		}
	})
}

// TestStateTransitionValidation tests the state transition validation logic
func TestStateTransitionValidation(t *testing.T) {
	wsm := NewWarStateManager()

	testCases := []struct {
		name        string
		fromState   WarState
		toState     WarState
		expected    bool
		description string
	}{
		{"NoWars to PreWar", NoWars, PreWar, true, "Can transition from NoWars to any state"},
		{"NoWars to ActiveWar", NoWars, ActiveWar, true, "Can transition from NoWars to any state"},
		{"PreWar to ActiveWar", PreWar, ActiveWar, true, "War started"},
		{"PreWar to NoWars", PreWar, NoWars, true, "War cancelled"},
		{"PreWar to PostWar", PreWar, PostWar, true, "War scheduled but immediately ended"},
		{"ActiveWar to PostWar", ActiveWar, PostWar, true, "War ended"},
		{"ActiveWar to PreWar", ActiveWar, PreWar, true, "Rare edge case"},
		{"ActiveWar to NoWars", ActiveWar, NoWars, false, "Cannot skip PostWar state"},
		{"PostWar to NoWars", PostWar, NoWars, true, "Post-war period expired"},
		{"PostWar to PreWar", PostWar, PreWar, true, "New war scheduled quickly"},
		{"PostWar to ActiveWar", PostWar, ActiveWar, false, "Cannot skip PreWar state"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the initial state with sufficient time elapsed
			wsm.currentState = tc.fromState
			wsm.lastStateChange = time.Now().Add(-1 * time.Minute) // Ensure enough time has passed

			result := wsm.isValidStateTransition(tc.fromState, tc.toState)
			if result != tc.expected {
				t.Errorf("Expected %v for transition %s -> %s (%s), got %v",
					tc.expected, tc.fromState.String(), tc.toState.String(), tc.description, result)
			}
		})
	}
}

// TestRapidStateTransitionPrevention tests prevention of rapid oscillation
func TestRapidStateTransitionPrevention(t *testing.T) {
	wsm := NewWarStateManager()

	// Set up initial state with recent change
	wsm.currentState = PreWar
	wsm.lastStateChange = time.Now().Add(-10 * time.Second) // Recent change

	// Attempt transition that would normally be valid but is too soon
	result := wsm.isValidStateTransition(PreWar, NoWars)
	if result {
		t.Error("Expected rapid transition PreWar -> NoWars to be blocked due to recent state change")
	}

	// Wait sufficient time and try again
	wsm.lastStateChange = time.Now().Add(-1 * time.Minute)
	result = wsm.isValidStateTransition(PreWar, NoWars)
	if !result {
		t.Error("Expected transition PreWar -> NoWars to be allowed after sufficient time")
	}
}

// TestStateInfo tests the state information reporting
func TestStateInfo(t *testing.T) {
	wsm := NewWarStateManager()

	// Set up a known state
	wsm.currentState = ActiveWar
	wsm.lastStateChange = time.Now().Add(-10 * time.Minute)

	info := wsm.GetStateInfo()

	if info.State != ActiveWar {
		t.Errorf("Expected state ActiveWar, got %s", info.State.String())
	}

	if info.TimeInState < 9*time.Minute || info.TimeInState > 11*time.Minute {
		t.Errorf("Expected time in state around 10 minutes, got %v", info.TimeInState)
	}

	if info.UpdateInterval != 1*time.Minute {
		t.Errorf("Expected 1 minute interval for ActiveWar, got %v", info.UpdateInterval)
	}

	if info.Description == "" {
		t.Error("Expected non-empty description")
	}
}