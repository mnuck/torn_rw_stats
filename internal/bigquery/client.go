package bigquery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	bq "cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"torn_rw_stats/internal/domain"
)

// Client wraps the BigQuery streaming insert and query APIs for state records.
type Client struct {
	bqClient  *bq.Client
	inserter  *bq.Inserter
	projectID string
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
		bqClient:  bqClient,
		inserter:  inserter,
		projectID: projectID,
		datasetID: datasetID,
		tableID:   tableID,
	}, nil
}

// stateRecordRow is the BigQuery-insertable representation of domain.StateRecord.
// It implements bq.ValueSaver.
type stateRecordRow struct {
	record domain.StateRecord
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

// QueryLatestStatePerMember returns the most recent StateRecord for each of the
// given member IDs. Members with no history are omitted from the result.
func (c *Client) QueryLatestStatePerMember(ctx context.Context, memberIDs []string) ([]domain.StateRecord, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}

	sql := fmt.Sprintf(`
SELECT member_id, member_name, faction_id, faction_name,
       last_action_status, status_description, status_state,
       status_until, status_travel_type, timestamp
FROM (
  SELECT *,
    ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY timestamp DESC) AS rn
  FROM %s.%s.%s
  WHERE member_id IN UNNEST(@member_ids)
)
WHERE rn = 1`,
		c.projectID, c.datasetID, c.tableID)

	q := c.bqClient.Query(sql)
	q.Parameters = []bq.QueryParameter{
		{Name: "member_ids", Value: memberIDs},
	}

	start := time.Now()
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery query failed: %w", err)
	}

	var records []domain.StateRecord
	for {
		var row struct {
			MemberID          string           `bigquery:"member_id"`
			MemberName        string           `bigquery:"member_name"`
			FactionID         string           `bigquery:"faction_id"`
			FactionName       string           `bigquery:"faction_name"`
			LastActionStatus  string           `bigquery:"last_action_status"`
			StatusDescription string           `bigquery:"status_description"`
			StatusState       string           `bigquery:"status_state"`
			StatusUntil       bq.NullTimestamp `bigquery:"status_until"`
			StatusTravelType  string           `bigquery:"status_travel_type"`
			Timestamp         time.Time        `bigquery:"timestamp"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery row iteration failed: %w", err)
		}

		r := domain.StateRecord{
			Timestamp:         row.Timestamp.UTC(),
			MemberID:          row.MemberID,
			MemberName:        row.MemberName,
			FactionID:         row.FactionID,
			FactionName:       row.FactionName,
			LastActionStatus:  row.LastActionStatus,
			StatusDescription: row.StatusDescription,
			StatusState:       row.StatusState,
			StatusTravelType:  row.StatusTravelType,
		}
		if row.StatusUntil.Valid {
			r.StatusUntil = row.StatusUntil.Timestamp.UTC()
		}
		records = append(records, r)
	}

	slog.Debug("BigQuery latest-state-per-member query complete",
		"members_queried", len(memberIDs),
		"records_returned", len(records),
		"duration", time.Since(start))

	return records, nil
}

// QueryLatestStatePerFaction returns the most recent StateRecord for each member
// of the given faction. Members with no history are omitted from the result.
func (c *Client) QueryLatestStatePerFaction(ctx context.Context, factionID string) ([]domain.StateRecord, error) {
	sql := fmt.Sprintf(`
SELECT member_id, member_name, faction_id, faction_name,
       last_action_status, status_description, status_state,
       status_until, status_travel_type, timestamp
FROM (
  SELECT *,
    ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY timestamp DESC) AS rn
  FROM %s.%s.%s
  WHERE faction_id = @faction_id
)
WHERE rn = 1`,
		c.projectID, c.datasetID, c.tableID)

	q := c.bqClient.Query(sql)
	q.Parameters = []bq.QueryParameter{
		{Name: "faction_id", Value: factionID},
	}

	start := time.Now()
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery faction query failed: %w", err)
	}

	var records []domain.StateRecord
	for {
		var row struct {
			MemberID          string           `bigquery:"member_id"`
			MemberName        string           `bigquery:"member_name"`
			FactionID         string           `bigquery:"faction_id"`
			FactionName       string           `bigquery:"faction_name"`
			LastActionStatus  string           `bigquery:"last_action_status"`
			StatusDescription string           `bigquery:"status_description"`
			StatusState       string           `bigquery:"status_state"`
			StatusUntil       bq.NullTimestamp `bigquery:"status_until"`
			StatusTravelType  string           `bigquery:"status_travel_type"`
			Timestamp         time.Time        `bigquery:"timestamp"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery row iteration failed: %w", err)
		}

		r := domain.StateRecord{
			Timestamp:         row.Timestamp.UTC(),
			MemberID:          row.MemberID,
			MemberName:        row.MemberName,
			FactionID:         row.FactionID,
			FactionName:       row.FactionName,
			LastActionStatus:  row.LastActionStatus,
			StatusDescription: row.StatusDescription,
			StatusState:       row.StatusState,
			StatusTravelType:  row.StatusTravelType,
		}
		if row.StatusUntil.Valid {
			r.StatusUntil = row.StatusUntil.Timestamp.UTC()
		}
		records = append(records, r)
	}

	slog.Debug("BigQuery latest-state-per-faction query complete",
		"faction_id", factionID,
		"records_returned", len(records),
		"duration", time.Since(start))

	return records, nil
}

// QueryDepartureTimes returns the most recent transition-to-traveling timestamp for
// each of the given member IDs. Members with no traveling history are omitted.
func (c *Client) QueryDepartureTimes(ctx context.Context, memberIDs []string) (map[string]time.Time, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}

	sql := fmt.Sprintf(`
SELECT member_id, timestamp AS departure_time
FROM (
  SELECT
    member_id,
    timestamp,
    status_state,
    LAG(status_state) OVER (PARTITION BY member_id ORDER BY timestamp) AS prev_status_state
  FROM %s.%s.%s
  WHERE member_id IN UNNEST(@member_ids)
)
WHERE status_state = 'Traveling'
  AND (prev_status_state IS NULL OR prev_status_state != 'Traveling')
QUALIFY ROW_NUMBER() OVER (PARTITION BY member_id ORDER BY timestamp DESC) = 1`,
		c.projectID, c.datasetID, c.tableID)

	q := c.bqClient.Query(sql)
	q.Parameters = []bq.QueryParameter{
		{Name: "member_ids", Value: memberIDs},
	}

	start := time.Now()
	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery departure query failed: %w", err)
	}

	result := make(map[string]time.Time)
	for {
		var row struct {
			MemberID      string    `bigquery:"member_id"`
			DepartureTime time.Time `bigquery:"departure_time"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery row iteration failed: %w", err)
		}
		result[row.MemberID] = row.DepartureTime.UTC()
	}

	slog.Debug("BigQuery departure-times query complete",
		"members_queried", len(memberIDs),
		"departures_found", len(result),
		"duration", time.Since(start))

	return result, nil
}

// InsertStateRecords streams the provided records into BigQuery.
func (c *Client) InsertStateRecords(ctx context.Context, records []domain.StateRecord) error {
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

	slog.Debug("BigQuery streaming insert complete",
		"rows", len(records),
		"duration", time.Since(start),
		"dataset", c.datasetID,
		"table", c.tableID)

	return nil
}
