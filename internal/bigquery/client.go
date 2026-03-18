package bigquery

import (
	"context"
	"fmt"
	"time"

	bq "cloud.google.com/go/bigquery"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"

	"torn_rw_stats/internal/app"
)

// Client wraps the BigQuery streaming insert API for state record insertion.
type Client struct {
	inserter  *bq.Inserter
	datasetID string
	tableID   string
}

// NewClient creates a Client using a service-account credentials file.
func NewClient(ctx context.Context, credentialsFile, projectID, datasetID, tableID string) (*Client, error) {
	bqClient, err := bq.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsFile)) //nolint:staticcheck
	if err != nil {
		return nil, fmt.Errorf("failed to create bigquery client: %w", err)
	}

	inserter := bqClient.Dataset(datasetID).Table(tableID).Inserter()

	return &Client{
		inserter:  inserter,
		datasetID: datasetID,
		tableID:   tableID,
	}, nil
}

// stateRecordRow is the BigQuery-insertable representation of app.StateRecord.
// It implements bq.ValueSaver.
type stateRecordRow struct {
	record app.StateRecord
}

func (r *stateRecordRow) Save() (map[string]bq.Value, string, error) {
	row := map[string]bq.Value{
		"timestamp":          r.record.Timestamp.UTC(),
		"member_id":          r.record.MemberID,
		"member_name":        r.record.MemberName,
		"faction_id":         r.record.FactionID,
		"faction_name":       r.record.FactionName,
		"last_action_status": r.record.LastActionStatus,
		"status_description": r.record.StatusDescription,
		"status_state":       r.record.StatusState,
		"status_travel_type": r.record.StatusTravelType,
	}
	// Only set status_until when non-zero to avoid storing the zero epoch as a TIMESTAMP.
	if !r.record.StatusUntil.IsZero() {
		row["status_until"] = r.record.StatusUntil.UTC()
	}
	// Empty InsertID: BigQuery assigns its own dedup ID (at-least-once semantics).
	return row, "", nil
}

// InsertStateRecords streams the provided records into BigQuery.
func (c *Client) InsertStateRecords(ctx context.Context, records []app.StateRecord) error {
	if len(records) == 0 {
		return nil
	}

	rows := make([]*stateRecordRow, len(records))
	for i, r := range records {
		r := r // capture loop variable
		rows[i] = &stateRecordRow{record: r}
	}

	start := time.Now()
	if err := c.inserter.Put(ctx, rows); err != nil {
		return fmt.Errorf("bigquery insert failed (dataset=%s table=%s): %w", c.datasetID, c.tableID, err)
	}

	log.Debug().
		Int("rows", len(records)).
		Dur("duration", time.Since(start)).
		Str("dataset", c.datasetID).
		Str("table", c.tableID).
		Msg("BigQuery streaming insert complete")

	return nil
}
