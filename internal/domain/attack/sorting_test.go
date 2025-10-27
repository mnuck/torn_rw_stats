package attack

import (
	"testing"
	"torn_rw_stats/internal/app"
)

func TestSortAttacksChronologically(t *testing.T) {
	attacks := []app.Attack{
		{Started: 300},
		{Started: 100},
		{Started: 200},
	}

	sorted := SortAttacksChronologically(attacks)

	// Verify sorted order
	if sorted[0].Started != 100 || sorted[1].Started != 200 || sorted[2].Started != 300 {
		t.Errorf("Attacks not sorted correctly: %v", sorted)
	}

	// Verify original slice unchanged
	if attacks[0].Started != 300 {
		t.Errorf("Original slice was modified")
	}
}

func TestSortAttacksChronologically_EmptySlice(t *testing.T) {
	attacks := []app.Attack{}
	sorted := SortAttacksChronologically(attacks)

	if len(sorted) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(sorted))
	}
}

func TestSortAttacksChronologically_SingleItem(t *testing.T) {
	attacks := []app.Attack{{Started: 100}}
	sorted := SortAttacksChronologically(attacks)

	if len(sorted) != 1 || sorted[0].Started != 100 {
		t.Errorf("Single item not handled correctly")
	}
}
