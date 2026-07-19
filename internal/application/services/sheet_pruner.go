package services

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"torn_rw_stats/internal/application/ports"
	wardomain "torn_rw_stats/internal/domain/war"
)

// pruneCheckInterval is the minimum time between prune passes. Pruning is a
// housekeeping task with day-scale retention, so there is no reason to scan the
// spreadsheet more than once a day.
const pruneCheckInterval = 24 * time.Hour

// SheetPruner deletes the sheets belonging to wars that concluded longer than
// the retention window ago, keeping the workbook under Google Sheets' hard
// 10M-cell limit. A war's "conclusion" is approximated by the timestamp of its
// most recent recorded attack, since the app stops updating a war's sheets once
// it drops out of the Torn API response.
type SheetPruner struct {
	sheets        ports.SheetsClient
	spreadsheetID string
	retention     time.Duration
	interval      time.Duration
	lastRun       time.Time
	now           func() time.Time
}

// NewSheetPruner creates a pruner with the given retention window. A
// non-positive retention disables pruning.
func NewSheetPruner(sheetsClient ports.SheetsClient, spreadsheetID string, retention time.Duration) *SheetPruner {
	return &SheetPruner{
		sheets:        sheetsClient,
		spreadsheetID: spreadsheetID,
		retention:     retention,
		interval:      pruneCheckInterval,
		now:           time.Now,
	}
}

// MaybePrune runs a prune pass if at least the check interval has elapsed since
// the previous run (the first call always runs). Failures are logged and
// swallowed so pruning never disrupts the main processing loop. activeWarIDs
// lists wars currently being monitored; their sheets are never touched.
func (sp *SheetPruner) MaybePrune(ctx context.Context, activeWarIDs []int) {
	if sp.retention <= 0 {
		return // pruning disabled
	}

	now := sp.now()
	if !sp.lastRun.IsZero() && now.Sub(sp.lastRun) < sp.interval {
		return
	}
	sp.lastRun = now

	if err := sp.prune(ctx, activeWarIDs, now); err != nil {
		slog.Error("Failed to prune concluded war sheets", "err", err)
	}
}

// prune performs a single prune pass.
func (sp *SheetPruner) prune(ctx context.Context, activeWarIDs []int, now time.Time) error {
	titles, err := sp.sheets.ListSheetTitles(ctx, sp.spreadsheetID)
	if err != nil {
		return fmt.Errorf("failed to list sheets: %w", err)
	}

	active := make(map[int]bool, len(activeWarIDs))
	for _, id := range activeWarIDs {
		active[id] = true
	}

	candidates := wardomain.PrunableWarIDs(titles, active)
	if len(candidates) == 0 {
		slog.Debug("No concluded wars eligible for pruning")
		return nil
	}

	slog.Info("Evaluating concluded wars for pruning",
		"candidate_wars", len(candidates),
		"retention", sp.retention)

	pruned := 0
	for _, warID := range candidates {
		lastActivity, err := sp.lastActivity(ctx, warID)
		if err != nil {
			slog.Warn("Could not determine war activity - skipping",
				"war_id", warID, "err", err)
			continue
		}

		if !wardomain.ShouldPruneWar(lastActivity, now, sp.retention) {
			continue
		}

		sp.deleteWarSheets(ctx, warID, lastActivity)
		pruned++
	}

	slog.Info("Completed war sheet pruning",
		"candidate_wars", len(candidates),
		"pruned_wars", pruned)
	return nil
}

// lastActivity returns the timestamp of a war's most recent recorded attack,
// used as a proxy for when the war concluded. A zero time means the war's age
// could not be established (empty or unparseable records), in which case the
// pruning policy keeps the sheets.
func (sp *SheetPruner) lastActivity(ctx context.Context, warID int) (time.Time, error) {
	info, err := sp.sheets.ReadExistingRecords(ctx, sp.spreadsheetID, wardomain.RecordsSheetTitle(warID))
	if err != nil {
		return time.Time{}, err
	}
	if info == nil || info.LatestTimestamp <= 0 {
		return time.Time{}, nil
	}
	return time.Unix(info.LatestTimestamp, 0).UTC(), nil
}

// deleteWarSheets removes every sheet owned by a war. Individual failures are
// logged and do not abort the remaining deletions.
func (sp *SheetPruner) deleteWarSheets(ctx context.Context, warID int, lastActivity time.Time) {
	for _, title := range wardomain.WarSheetTitles(warID) {
		if err := sp.sheets.DeleteSheet(ctx, sp.spreadsheetID, title); err != nil {
			slog.Error("Failed to delete concluded war sheet",
				"war_id", warID, "sheet_name", title, "err", err)
			continue
		}
		slog.Info("Pruned concluded war sheet",
			"war_id", warID,
			"sheet_name", title,
			"last_activity", lastActivity.Format("2006-01-02 15:04:05"))
	}
}
