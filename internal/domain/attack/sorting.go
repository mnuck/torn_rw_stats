package attack

import (
	"sort"
	"torn_rw_stats/internal/app"
)

// SortAttacksChronologically returns a new slice with attacks sorted by timestamp (oldest first)
// Pure function: Does not modify input slice, returns new sorted slice
func SortAttacksChronologically(attacks []app.Attack) []app.Attack {
	sorted := make([]app.Attack, len(attacks))
	copy(sorted, attacks)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Started < sorted[j].Started
	})

	return sorted
}
