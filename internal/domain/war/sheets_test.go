package war

import (
	"testing"
	"time"
)

func TestSheetTitleRoundTrip(t *testing.T) {
	for _, warID := range []int{1, 45796, 999999} {
		if got, ok := ParseWarSheetTitle(SummarySheetTitle(warID)); !ok || got != warID {
			t.Errorf("summary round-trip for %d: got (%d, %v)", warID, got, ok)
		}
		if got, ok := ParseWarSheetTitle(RecordsSheetTitle(warID)); !ok || got != warID {
			t.Errorf("records round-trip for %d: got (%d, %v)", warID, got, ok)
		}
	}
}

func TestParseWarSheetTitle_NonMatching(t *testing.T) {
	cases := []string{
		"Status v2 - 40692",
		"Summary",
		"Summary - ",
		"Summary - abc",
		"Records - 12x",
		"Sheet1",
		"",
	}
	for _, title := range cases {
		if _, ok := ParseWarSheetTitle(title); ok {
			t.Errorf("expected %q to not parse as a war sheet", title)
		}
	}
}

func TestWarSheetTitles(t *testing.T) {
	got := WarSheetTitles(45796)
	want := []string{"Summary - 45796", "Records - 45796"}
	if len(got) != len(want) {
		t.Fatalf("got %d titles, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("title[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPrunableWarIDs(t *testing.T) {
	titles := []string{
		"Summary - 100",
		"Records - 100",
		"Summary - 200",
		"Records - 200",
		"Status v2 - 38482", // not a war sheet
		"Summary - 300",     // active, must be excluded
		"Records - 300",
		"Config", // unrelated
	}
	active := map[int]bool{300: true}

	got := PrunableWarIDs(titles, active)
	want := []int{100, 200}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestPrunableWarIDs_Empty(t *testing.T) {
	if got := PrunableWarIDs(nil, nil); len(got) != 0 {
		t.Errorf("expected no candidates, got %v", got)
	}
}

func TestShouldPruneWar(t *testing.T) {
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	retention := 30 * 24 * time.Hour

	tests := []struct {
		name         string
		lastActivity time.Time
		want         bool
	}{
		{"zero time is kept", time.Time{}, false},
		{"just concluded", now.Add(-time.Hour), false},
		{"exactly at retention edge", now.Add(-retention), false},
		{"one second past retention", now.Add(-retention - time.Second), true},
		{"long concluded", now.Add(-90 * 24 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldPruneWar(tt.lastActivity, now, retention); got != tt.want {
				t.Errorf("ShouldPruneWar(%v) = %v, want %v", tt.lastActivity, got, tt.want)
			}
		})
	}
}
