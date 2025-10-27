package torn

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/app"
	"torn_rw_stats/internal/domain/attack"
)

// MockTornAPI implements TornAPI for testing
type MockTornAPI struct {
	warResponse         *app.WarResponse
	attackResponse      *app.AttackResponse
	factionResponse     *app.FactionBasicResponse
	factionInfoResponse *app.FactionInfoResponse
	apiCallCount        int64
	shouldError         bool
}

func (m *MockTornAPI) GetFactionWars(ctx context.Context) (*app.WarResponse, error) {
	if m.shouldError {
		return nil, &mockError{msg: "mock error"}
	}
	m.apiCallCount++
	return m.warResponse, nil
}

func (m *MockTornAPI) GetFactionAttacks(ctx context.Context, from, to int64) (*app.AttackResponse, error) {
	if m.shouldError {
		return nil, &mockError{msg: "mock error"}
	}
	m.apiCallCount++
	return m.attackResponse, nil
}

func (m *MockTornAPI) GetFactionBasic(ctx context.Context, factionID int) (*app.FactionBasicResponse, error) {
	if m.shouldError {
		return nil, &mockError{msg: "mock error"}
	}
	m.apiCallCount++
	return m.factionResponse, nil
}

func (m *MockTornAPI) GetOwnFaction(ctx context.Context) (*app.FactionInfoResponse, error) {
	if m.shouldError {
		return nil, &mockError{msg: "mock error"}
	}
	m.apiCallCount++
	return m.factionInfoResponse, nil
}

func (m *MockTornAPI) GetAPICallCount() int64 {
	return m.apiCallCount
}

func (m *MockTornAPI) IncrementAPICall() {
	m.apiCallCount++
}

func (m *MockTornAPI) ResetAPICallCount() {
	m.apiCallCount = 0
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func TestProcessorCalculateTimeRange(t *testing.T) {
	mockAPI := &MockTornAPI{}
	processor := NewAttackProcessor(mockAPI)

	endTime := time.Now().Unix() + 3600
	war := &app.War{
		ID:    123,
		Start: time.Now().Unix() - 3600, // 1 hour ago
		End:   &endTime,                 // 1 hour from now
	}

	t.Run("NoExistingTimestamp", func(t *testing.T) {
		timeRange := processor.CalculateTimeRange(war, nil)

		if timeRange.FromTime != war.Start {
			t.Errorf("Expected FromTime to be war start time %d, got %d", war.Start, timeRange.FromTime)
		}

		if timeRange.ToTime != *war.End {
			t.Errorf("Expected ToTime to be war end time %d, got %d", *war.End, timeRange.ToTime)
		}

		if timeRange.UpdateMode != "full" {
			t.Errorf("Expected UpdateMode to be 'full', got '%s'", timeRange.UpdateMode)
		}
	})

	t.Run("WithExistingTimestamp", func(t *testing.T) {
		existing := war.Start + 1800 // 30 minutes after war start
		timeRange := processor.CalculateTimeRange(war, &existing)

		// Should use existing timestamp minus 1 hour buffer, but not before war start
		expectedFromTime := existing - 3600 // 1 hour buffer
		if expectedFromTime < war.Start {
			expectedFromTime = war.Start
		}

		if timeRange.FromTime != expectedFromTime {
			t.Errorf("Expected FromTime to be %d, got %d", expectedFromTime, timeRange.FromTime)
		}

		if timeRange.ToTime != *war.End {
			t.Errorf("Expected ToTime to be war end time %d, got %d", *war.End, timeRange.ToTime)
		}

		if timeRange.UpdateMode != "incremental" {
			t.Errorf("Expected UpdateMode to be 'incremental', got '%s'", timeRange.UpdateMode)
		}
	})

	t.Run("OngoingWar", func(t *testing.T) {
		ongoingWar := &app.War{
			ID:    456,
			Start: time.Now().Unix() - 3600,
			End:   nil, // Ongoing war
		}

		timeRange := processor.CalculateTimeRange(ongoingWar, nil)

		if timeRange.ToTime <= time.Now().Unix()-10 {
			t.Error("Expected ToTime to be close to current time for ongoing war")
		}
	})
}

func TestProcessorShouldUseSimpleApproach(t *testing.T) {
	testCases := []struct {
		name           string
		startTime      time.Time
		endTime        time.Time
		expectedSimple bool
	}{
		{
			name:           "SmallTimeRange6Hours",
			startTime:      time.Unix(1000, 0),
			endTime:        time.Unix(1000+(6*60*60), 0), // 6 hours
			expectedSimple: true,                         // Small ranges use simple approach
		},
		{
			name:           "TimeRangeUnder24Hours",
			startTime:      time.Unix(1000, 0),
			endTime:        time.Unix(1000+(23*60*60), 0), // 23 hours
			expectedSimple: true,                          // Under 24 hours uses simple approach
		},
		{
			name:           "TimeRangeOver24Hours",
			startTime:      time.Unix(1000, 0),
			endTime:        time.Unix(1000+(25*60*60), 0), // 25 hours
			expectedSimple: false,                         // Over 24 hours uses pagination
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := attack.ShouldUseSimpleApproach(tc.startTime, tc.endTime)
			if result != tc.expectedSimple {
				t.Errorf("Expected %v, got %v", tc.expectedSimple, result)
			}
		})
	}
}

func TestFilterRelevantAttacks(t *testing.T) {
	war := &app.War{
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
			{ID: 1002, Name: "Faction B"},
		},
	}

	attacks := []app.Attack{
		{
			ID:       1,
			Started:  1000,
			Attacker: app.User{Faction: &app.Faction{ID: 1001}},
			Defender: app.User{Faction: &app.Faction{ID: 1002}},
		},
		{
			ID:       2,
			Started:  1100,
			Attacker: app.User{Faction: &app.Faction{ID: 9999}},
			Defender: app.User{Faction: &app.Faction{ID: 8888}},
		},
		{
			ID:       3,
			Started:  1200,
			Attacker: app.User{Faction: &app.Faction{ID: 1001}},
			Defender: app.User{Faction: &app.Faction{ID: 9999}},
		},
		{
			ID:       4,
			Started:  1300,
			Attacker: app.User{Faction: nil},
			Defender: app.User{Faction: &app.Faction{ID: 1002}},
		},
	}

	warFactionIDs := attack.BuildFactionIDMap(war)
	relevantAttacks := attack.FilterRelevantAttacks(attacks, warFactionIDs)

	// Should have 3 relevant attacks (1, 3, 4) but not attack 2
	expectedCount := 3
	if len(relevantAttacks) != expectedCount {
		t.Errorf("Expected %d relevant attacks, got %d", expectedCount, len(relevantAttacks))
	}

	// Check specific attacks are included
	foundIDs := make(map[int64]bool)
	for _, attack := range relevantAttacks {
		foundIDs[attack.ID] = true
	}

	expectedIDs := []int64{1, 3, 4}
	for _, id := range expectedIDs {
		if !foundIDs[id] {
			t.Errorf("Expected attack ID %d to be relevant", id)
		}
	}

	if foundIDs[2] {
		t.Error("Attack ID 2 should not be relevant")
	}
}

func TestProcessorSortAttacksChronologically(t *testing.T) {
	attacks := []app.Attack{
		{ID: 1, Started: 1000},
		{ID: 2, Started: 500},
		{ID: 3, Started: 1500},
		{ID: 4, Started: 750},
	}

	sorted := attack.SortAttacksChronologically(attacks)

	// Should be sorted in ascending order
	expected := []int64{500, 750, 1000, 1500}
	for i, att := range sorted {
		if att.Started != expected[i] {
			t.Errorf("Position %d: expected timestamp %d, got %d", i, expected[i], att.Started)
		}
	}

	// Verify original slice unchanged
	if attacks[0].Started != 1000 {
		t.Error("Original slice was modified")
	}
}

func TestGetAllAttacksForWar(t *testing.T) {
	mockAPI := &MockTornAPI{
		attackResponse: &app.AttackResponse{
			Attacks: []app.Attack{
				{
					ID:       1,
					Started:  1000,
					Attacker: app.User{Faction: &app.Faction{ID: 1001}},
					Defender: app.User{Faction: &app.Faction{ID: 1002}},
				},
			},
		},
	}
	processor := NewAttackProcessor(mockAPI)

	war := &app.War{
		ID:    123,
		Start: 900,
		End:   &[]int64{1100}[0],
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
			{ID: 1002, Name: "Faction B"},
		},
	}

	attacks, err := processor.GetAllAttacksForWar(context.Background(), war)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(attacks) != 1 {
		t.Errorf("Expected 1 attack, got %d", len(attacks))
	}

	if mockAPI.GetAPICallCount() != 1 {
		t.Errorf("Expected 1 API call, got %d", mockAPI.GetAPICallCount())
	}
}

func TestGetAttacksForTimeRangeError(t *testing.T) {
	mockAPI := &MockTornAPI{shouldError: true}
	processor := NewAttackProcessor(mockAPI)

	war := &app.War{
		ID:    123,
		Start: 900,
		End:   &[]int64{1100}[0],
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
		},
	}

	_, err := processor.GetAllAttacksForWar(context.Background(), war)
	if err == nil {
		t.Fatal("Expected error due to mock API failure, got nil")
	}
}

func TestGetAttacksForTimeRangeNilWar(t *testing.T) {
	mockAPI := &MockTornAPI{}
	processor := NewAttackProcessor(mockAPI)

	_, err := processor.GetAttacksForTimeRange(context.Background(), nil, 1000, nil)
	if err == nil {
		t.Fatal("Expected error for nil war, got nil")
	}

	expectedMsg := "war cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestProcessAttacksPage(t *testing.T) {
	mockAPI := &MockTornAPI{}
	processor := NewAttackProcessor(mockAPI)

	war := &app.War{
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
			{ID: 1002, Name: "Faction B"},
		},
	}

	attacks := []app.Attack{
		{
			ID:       1,
			Started:  1000,
			Attacker: app.User{Faction: &app.Faction{ID: 1001}},
			Defender: app.User{Faction: &app.Faction{ID: 1002}},
		},
		{
			ID:       2,
			Started:  500, // Older timestamp
			Attacker: app.User{Faction: &app.Faction{ID: 9999}},
			Defender: app.User{Faction: &app.Faction{ID: 8888}},
		},
	}

	currentTo := int64(1200)
	result := processor.processAttacksPage(attacks, war, currentTo)

	// Should have 1 relevant attack
	if len(result.RelevantAttacks) != 1 {
		t.Errorf("Expected 1 relevant attack, got %d", len(result.RelevantAttacks))
	}

	// Should track oldest timestamp (500)
	if result.OldestAttackTime != 500 {
		t.Errorf("Expected oldest attack time 500, got %d", result.OldestAttackTime)
	}

	// Should track total count
	if result.TotalAttacksCount != 2 {
		t.Errorf("Expected total attack count 2, got %d", result.TotalAttacksCount)
	}
}

func TestShouldStopPagination(t *testing.T) {
	mockAPI := &MockTornAPI{}
	processor := NewAttackProcessor(mockAPI)

	testCases := []struct {
		name       string
		pageResult *PageResult
		fromTime   int64
		shouldStop bool
	}{
		{
			name: "NoAttacksReturned",
			pageResult: &PageResult{
				TotalAttacksCount: 0,
			},
			fromTime:   1000,
			shouldStop: true,
		},
		{
			name: "LessThanFullPage",
			pageResult: &PageResult{
				TotalAttacksCount: 50,
			},
			fromTime:   1000,
			shouldStop: true,
		},
		{
			name: "ReachedFetchStartTime",
			pageResult: &PageResult{
				TotalAttacksCount: 100,
				OldestAttackTime:  900,
			},
			fromTime:   1000,
			shouldStop: true,
		},
		{
			name: "FullPageMoreDataAvailable",
			pageResult: &PageResult{
				TotalAttacksCount: 100,
				OldestAttackTime:  1100,
			},
			fromTime:   1000,
			shouldStop: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.shouldStopPagination(tc.pageResult, tc.fromTime)
			if result != tc.shouldStop {
				t.Errorf("Expected shouldStop %v, got %v", tc.shouldStop, result)
			}
		})
	}
}
