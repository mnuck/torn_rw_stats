package services

import (
	"context"
	"testing"
	"time"

	"torn_rw_stats/internal/application/ports/mocks"
	"torn_rw_stats/internal/domain"
	wardomain "torn_rw_stats/internal/domain/war"
)

// recordsByWar builds a ReadExistingRecordsFunc that maps a war's records sheet
// to a RecordsInfo whose LatestTimestamp is the given time (zero => empty).
func recordsByWar(times map[int]time.Time) func(sheetName string) (*domain.RecordsInfo, error) {
	return func(sheetName string) (*domain.RecordsInfo, error) {
		for warID, t := range times {
			if sheetName == wardomain.RecordsSheetTitle(warID) {
				var ts int64
				if !t.IsZero() {
					ts = t.Unix()
				}
				return &domain.RecordsInfo{LatestTimestamp: ts}, nil
			}
		}
		return &domain.RecordsInfo{}, nil
	}
}

func newPrunerAt(sheetsClient *mocks.MockSheetsClient, retention time.Duration, now time.Time) *SheetPruner {
	p := NewSheetPruner(sheetsClient, "sheet-id", retention)
	p.now = func() time.Time { return now }
	return p
}

func TestSheetPruner_PrunesOnlyOldConcludedWars(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	retention := 30 * 24 * time.Hour

	sheetsClient := mocks.NewMockSheetsClient()
	sheetsClient.ListSheetTitlesResponse = []string{
		"Summary - 100", "Records - 100", // old -> prune
		"Summary - 200", "Records - 200", // recent -> keep
		"Summary - 300", "Records - 300", // active -> keep
		"Status v2 - 38482", // not a war sheet
	}
	sheetsClient.ReadExistingRecordsFunc = recordsByWar(map[int]time.Time{
		100: now.Add(-60 * 24 * time.Hour),
		200: now.Add(-2 * 24 * time.Hour),
		300: now.Add(-90 * 24 * time.Hour), // old, but active so excluded
	})

	p := newPrunerAt(sheetsClient, retention, now)
	p.MaybePrune(context.Background(), []int{300})

	want := map[string]bool{"Summary - 100": true, "Records - 100": true}
	if len(sheetsClient.DeletedSheets) != len(want) {
		t.Fatalf("deleted %v, want keys %v", sheetsClient.DeletedSheets, want)
	}
	for _, name := range sheetsClient.DeletedSheets {
		if !want[name] {
			t.Errorf("unexpectedly deleted %q", name)
		}
	}
}

func TestSheetPruner_KeepsWarsWithoutRecords(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	sheetsClient := mocks.NewMockSheetsClient()
	sheetsClient.ListSheetTitlesResponse = []string{"Summary - 100", "Records - 100"}
	sheetsClient.ReadExistingRecordsFunc = recordsByWar(map[int]time.Time{
		100: {}, // empty records -> unknown age -> keep
	})

	p := newPrunerAt(sheetsClient, 30*24*time.Hour, now)
	p.MaybePrune(context.Background(), nil)

	if len(sheetsClient.DeletedSheets) != 0 {
		t.Errorf("expected nothing pruned, deleted %v", sheetsClient.DeletedSheets)
	}
}

func TestSheetPruner_RateLimited(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	sheetsClient := mocks.NewMockSheetsClient()
	sheetsClient.ListSheetTitlesResponse = []string{"Summary - 100", "Records - 100"}
	sheetsClient.ReadExistingRecordsFunc = recordsByWar(map[int]time.Time{
		100: now.Add(-60 * 24 * time.Hour),
	})

	p := newPrunerAt(sheetsClient, 30*24*time.Hour, now)

	p.MaybePrune(context.Background(), nil) // runs
	if len(sheetsClient.DeletedSheets) != 2 {
		t.Fatalf("first run should prune war 100, deleted %v", sheetsClient.DeletedSheets)
	}

	// Advance only an hour; the next call must be skipped.
	p.now = func() time.Time { return now.Add(time.Hour) }
	p.MaybePrune(context.Background(), nil)
	if len(sheetsClient.DeletedSheets) != 2 {
		t.Errorf("second run within interval should be a no-op, deleted %v", sheetsClient.DeletedSheets)
	}

	// Advance past the interval; it runs again (sheets already gone, so no
	// new deletions occur because the titles list no longer includes them).
	sheetsClient.ListSheetTitlesResponse = nil
	p.now = func() time.Time { return now.Add(25 * time.Hour) }
	p.MaybePrune(context.Background(), nil)
	if len(sheetsClient.DeletedSheets) != 2 {
		t.Errorf("expected no further deletions, deleted %v", sheetsClient.DeletedSheets)
	}
}

func TestSheetPruner_DisabledWhenRetentionZero(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)

	sheetsClient := mocks.NewMockSheetsClient()
	sheetsClient.ListSheetTitlesResponse = []string{"Summary - 100", "Records - 100"}
	sheetsClient.ReadExistingRecordsFunc = recordsByWar(map[int]time.Time{
		100: now.Add(-365 * 24 * time.Hour),
	})

	p := newPrunerAt(sheetsClient, 0, now)
	p.MaybePrune(context.Background(), nil)

	if len(sheetsClient.DeletedSheets) != 0 {
		t.Errorf("pruning disabled, but deleted %v", sheetsClient.DeletedSheets)
	}
}
