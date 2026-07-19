package war

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultSheetRetention is how long a war's sheets are kept after the war's
// last recorded activity before they become eligible for pruning.
const DefaultSheetRetention = 30 * 24 * time.Hour

// Per-war sheet title prefixes. These are the canonical naming convention for
// the sheets a single war owns; the sheets adapter builds titles via
// SummarySheetTitle / RecordsSheetTitle so this package is the single source of
// truth for the format, which lets us reliably parse war IDs back out.
const (
	summarySheetPrefix = "Summary - "
	recordsSheetPrefix = "Records - "
)

// SummarySheetTitle returns the summary sheet title for a war.
func SummarySheetTitle(warID int) string {
	return fmt.Sprintf("%s%d", summarySheetPrefix, warID)
}

// RecordsSheetTitle returns the records sheet title for a war.
func RecordsSheetTitle(warID int) string {
	return fmt.Sprintf("%s%d", recordsSheetPrefix, warID)
}

// WarSheetTitles returns every per-war sheet title owned by a war, in a stable
// order. These are the sheets pruning deletes together.
func WarSheetTitles(warID int) []string {
	return []string{SummarySheetTitle(warID), RecordsSheetTitle(warID)}
}

// ParseWarSheetTitle returns the war ID encoded in a per-war sheet title
// ("Summary - 45796" or "Records - 45796") and whether the title matched.
func ParseWarSheetTitle(title string) (warID int, ok bool) {
	for _, prefix := range []string{summarySheetPrefix, recordsSheetPrefix} {
		if suffix, found := strings.CutPrefix(title, prefix); found {
			if id, err := strconv.Atoi(strings.TrimSpace(suffix)); err == nil {
				return id, true
			}
		}
	}
	return 0, false
}

// PrunableWarIDs returns the distinct war IDs that own per-war sheets in titles,
// excluding any war IDs in the active set. A war that is still being monitored
// must never be pruned, so its ID is filtered out even if its sheets appear.
// The result is ordered by first appearance for deterministic behaviour.
func PrunableWarIDs(titles []string, active map[int]bool) []int {
	seen := make(map[int]bool)
	var ids []int
	for _, title := range titles {
		id, ok := ParseWarSheetTitle(title)
		if !ok || active[id] || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// ShouldPruneWar reports whether a war whose last recorded activity was at
// lastActivity should have its sheets pruned as of now, given the retention
// window. A zero lastActivity means the war's age cannot be established, so it
// is kept (fail safe: never delete a war we can't date).
func ShouldPruneWar(lastActivity, now time.Time, retention time.Duration) bool {
	if lastActivity.IsZero() {
		return false
	}
	return now.Sub(lastActivity) > retention
}
