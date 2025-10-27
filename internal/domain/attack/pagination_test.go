package attack

import (
	"testing"
	"torn_rw_stats/internal/app"
)

func TestShouldStopPagination(t *testing.T) {
	const pageSize = 100
	const fetchStartTime = 1000

	tests := []struct {
		name                string
		totalAttacksInPage  int
		oldestAttackTime    int64
		fetchStartTime      int64
		expectedStop        bool
		expectedReason      string
	}{
		{
			name:               "NoAttacksReturned",
			totalAttacksInPage: 0,
			oldestAttackTime:   1500,
			fetchStartTime:     fetchStartTime,
			expectedStop:       true,
			expectedReason:     "no_more_attacks",
		},
		{
			name:               "PartialPage",
			totalAttacksInPage: 50,
			oldestAttackTime:   1500,
			fetchStartTime:     fetchStartTime,
			expectedStop:       true,
			expectedReason:     "partial_page",
		},
		{
			name:               "ReachedStartTime",
			totalAttacksInPage: pageSize,
			oldestAttackTime:   900, // Before fetchStartTime
			fetchStartTime:     fetchStartTime,
			expectedStop:       true,
			expectedReason:     "reached_start_time",
		},
		{
			name:               "ReachedStartTimeExactly",
			totalAttacksInPage: pageSize,
			oldestAttackTime:   fetchStartTime,
			fetchStartTime:     fetchStartTime,
			expectedStop:       true,
			expectedReason:     "reached_start_time",
		},
		{
			name:               "ContinuePagination",
			totalAttacksInPage: pageSize,
			oldestAttackTime:   1500, // After fetchStartTime
			fetchStartTime:     fetchStartTime,
			expectedStop:       false,
			expectedReason:     "continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := ShouldStopPagination(
				tt.totalAttacksInPage,
				tt.oldestAttackTime,
				tt.fetchStartTime,
				pageSize,
			)

			if decision.ShouldStop != tt.expectedStop {
				t.Errorf("Expected ShouldStop=%v, got %v", tt.expectedStop, decision.ShouldStop)
			}

			if decision.Reason != tt.expectedReason {
				t.Errorf("Expected Reason=%q, got %q", tt.expectedReason, decision.Reason)
			}

			if decision.OldestTimestamp != tt.oldestAttackTime {
				t.Errorf("Expected OldestTimestamp=%d, got %d", tt.oldestAttackTime, decision.OldestTimestamp)
			}

			if decision.AttacksProcessed != tt.totalAttacksInPage {
				t.Errorf("Expected AttacksProcessed=%d, got %d", tt.totalAttacksInPage, decision.AttacksProcessed)
			}
		})
	}
}

func TestFindOldestAttackTime(t *testing.T) {
	tests := []struct {
		name         string
		attacks      []app.Attack
		defaultTime  int64
		expectedTime int64
	}{
		{
			name:         "EmptySlice",
			attacks:      []app.Attack{},
			defaultTime:  1000,
			expectedTime: 1000,
		},
		{
			name: "SingleAttack",
			attacks: []app.Attack{
				{Started: 500},
			},
			defaultTime:  1000,
			expectedTime: 500,
		},
		{
			name: "MultipleAttacks",
			attacks: []app.Attack{
				{Started: 1000},
				{Started: 500},
				{Started: 1500},
				{Started: 750},
			},
			defaultTime:  2000,
			expectedTime: 500,
		},
		{
			name: "AllAttacksNewerThanDefault",
			attacks: []app.Attack{
				{Started: 2000},
				{Started: 2500},
			},
			defaultTime:  1000,
			expectedTime: 1000, // defaultTime is older
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindOldestAttackTime(tt.attacks, tt.defaultTime)
			if result != tt.expectedTime {
				t.Errorf("Expected %d, got %d", tt.expectedTime, result)
			}
		})
	}
}
