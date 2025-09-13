package torn

import (
	"testing"
	"time"

	"torn_rw_stats/internal/app"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test_api_key")

	if client.apiKey != "test_api_key" {
		t.Errorf("Expected API key 'test_api_key', got '%s'", client.apiKey)
	}

	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}

	if client.apiCallCount != 0 {
		t.Errorf("Expected API call count 0, got %d", client.apiCallCount)
	}
}

func TestAPICallCounter(t *testing.T) {
	client := NewClient("test_api_key")

	// Test initial count
	if count := client.GetAPICallCount(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Test increment
	client.IncrementAPICall()
	if count := client.GetAPICallCount(); count != 1 {
		t.Errorf("Expected count 1 after increment, got %d", count)
	}

	// Test multiple increments
	client.IncrementAPICall()
	client.IncrementAPICall()
	if count := client.GetAPICallCount(); count != 3 {
		t.Errorf("Expected count 3 after multiple increments, got %d", count)
	}

	// Test reset
	client.ResetAPICallCount()
	if count := client.GetAPICallCount(); count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}
}

func TestCalculateTimeRange(t *testing.T) {
	client := NewClient("test_api_key")

	endTime := time.Now().Unix() + 3600
	war := &app.War{
		Start: time.Now().Unix() - 3600, // 1 hour ago
		End:   &endTime,                 // 1 hour from now (pointer required)
	}

	t.Run("NoExistingTimestamp", func(t *testing.T) {
		timeRange := client.calculateTimeRange(war, nil)

		if timeRange.FromTime != war.Start {
			t.Errorf("Expected FromTime to be war start time %d, got %d", war.Start, timeRange.FromTime)
		}

		if timeRange.ToTime != *war.End {
			t.Errorf("Expected ToTime to be war end time %d, got %d", *war.End, timeRange.ToTime)
		}
	})

	t.Run("WithExistingTimestamp", func(t *testing.T) {
		existing := war.Start + 1800 // 30 minutes after war start
		timeRange := client.calculateTimeRange(war, &existing)

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
}

func TestShouldUseSimpleApproach(t *testing.T) {
	client := NewClient("test_api_key")

	testCases := []struct {
		name           string
		timeRange      TimeRange
		expectedSimple bool
	}{
		{
			name: "FullUpdateMode",
			timeRange: TimeRange{
				FromTime:   1000,
				ToTime:     1000 + (6 * 60 * 60), // 6 hours
				UpdateMode: "full",
			},
			expectedSimple: false, // Full mode always uses pagination
		},
		{
			name: "IncrementalSmallTimeRange",
			timeRange: TimeRange{
				FromTime:   1000,
				ToTime:     1000 + (6 * 60 * 60), // 6 hours
				UpdateMode: "incremental",
			},
			expectedSimple: true, // Small incremental updates use simple approach
		},
		{
			name: "IncrementalLargeTimeRange",
			timeRange: TimeRange{
				FromTime:   1000,
				ToTime:     1000 + (25 * 60 * 60), // 25 hours
				UpdateMode: "incremental",
			},
			expectedSimple: false, // Large time ranges use pagination
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.shouldUseSimpleApproach(tc.timeRange)
			if result != tc.expectedSimple {
				t.Errorf("Expected %v, got %v", tc.expectedSimple, result)
			}
		})
	}
}

func TestIsAttackRelevantToWar(t *testing.T) {
	client := NewClient("test_api_key")

	war := &app.War{
		Factions: []app.Faction{
			{ID: 1001, Name: "Faction A"},
			{ID: 1002, Name: "Faction B"},
		},
	}

	testCases := []struct {
		name           string
		attack         app.Attack
		expectedResult bool
	}{
		{
			name: "RelevantAttack",
			attack: app.Attack{
				Attacker: app.User{Faction: &app.Faction{ID: 1001}},
				Defender: app.User{Faction: &app.Faction{ID: 1002}},
			},
			expectedResult: true,
		},
		{
			name: "IrrelevantAttack",
			attack: app.Attack{
				Attacker: app.User{Faction: &app.Faction{ID: 9999}},
				Defender: app.User{Faction: &app.Faction{ID: 8888}},
			},
			expectedResult: false,
		},
		{
			name: "PartiallyRelevantAttack",
			attack: app.Attack{
				Attacker: app.User{Faction: &app.Faction{ID: 1001}},
				Defender: app.User{Faction: &app.Faction{ID: 9999}},
			},
			expectedResult: true, // Should be true because attacker faction is in war
		},
		{
			name: "NilFactions",
			attack: app.Attack{
				Attacker: app.User{Faction: nil},
				Defender: app.User{Faction: &app.Faction{ID: 1002}},
			},
			expectedResult: true, // Should be true because defender faction is in war
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.isAttackRelevantToWar(tc.attack, war)
			if result != tc.expectedResult {
				t.Errorf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestSortAttacksChronologically(t *testing.T) {
	client := NewClient("test_api_key")

	attacks := []app.Attack{
		{Started: 1000},
		{Started: 500},
		{Started: 1500},
		{Started: 750},
	}

	client.sortAttacksChronologically(attacks)

	// Should be sorted in ascending order
	expected := []int64{500, 750, 1000, 1500}
	for i, attack := range attacks {
		if attack.Started != expected[i] {
			t.Errorf("Position %d: expected timestamp %d, got %d", i, expected[i], attack.Started)
		}
	}
}