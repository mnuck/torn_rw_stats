// setup-bigquery creates the BigQuery dataset and table needed by torn_rw_stats.
// Run this once before deploying the application.
//
// Usage:
//
//	go run ./cmd/setup-bigquery/
//
// Required env vars (same as the main app):
//
//	BIGQUERY_PROJECT_ID   GCP project that owns the dataset
//	BIGQUERY_DATASET_ID   Dataset to create (if it doesn't exist)
//
// Optional env vars:
//
//	BIGQUERY_TABLE_ID         Table name (default: state_changes)
//	BIGQUERY_LOCATION         Dataset location (default: US)
//	GOOGLE_CREDENTIALS_FILE   Path to service-account key (default: credentials.json)
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	bq "cloud.google.com/go/bigquery"
	"github.com/joho/godotenv"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func main() {
	_ = godotenv.Load()

	projectID := os.Getenv("BIGQUERY_PROJECT_ID")
	if projectID == "" {
		fatalf("BIGQUERY_PROJECT_ID is required")
	}

	datasetID := os.Getenv("BIGQUERY_DATASET_ID")
	if datasetID == "" {
		fatalf("BIGQUERY_DATASET_ID is required")
	}

	tableID := os.Getenv("BIGQUERY_TABLE_ID")
	if tableID == "" {
		tableID = "state_changes"
	}

	location := os.Getenv("BIGQUERY_LOCATION")
	if location == "" {
		location = "US"
	}

	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if credentialsFile == "" {
		credentialsFile = "credentials.json"
	}

	ctx := context.Background()

	client, err := bq.NewClient(ctx, projectID, option.WithCredentialsFile(credentialsFile)) //nolint:staticcheck
	if err != nil {
		fatalf("failed to create BigQuery client: %v", err)
	}
	defer client.Close()

	// --- Create dataset if it doesn't exist ---
	dataset := client.Dataset(datasetID)
	if err := dataset.Create(ctx, &bq.DatasetMetadata{Location: location}); err != nil {
		if !isAlreadyExists(err) {
			fatalf("failed to create dataset %q: %v", datasetID, err)
		}
		fmt.Printf("Dataset %q already exists — skipping creation.\n", datasetID)
	} else {
		fmt.Printf("Created dataset %q in location %q.\n", datasetID, location)
	}

	// --- Create table if it doesn't exist ---
	schema := bq.Schema{
		{Name: "timestamp", Type: bq.TimestampFieldType, Required: true},
		{Name: "member_id", Type: bq.StringFieldType},
		{Name: "member_name", Type: bq.StringFieldType},
		{Name: "faction_id", Type: bq.StringFieldType},
		{Name: "faction_name", Type: bq.StringFieldType},
		{Name: "last_action_status", Type: bq.StringFieldType},
		{Name: "status_description", Type: bq.StringFieldType},
		{Name: "status_state", Type: bq.StringFieldType},
		{Name: "status_until", Type: bq.TimestampFieldType},
		{Name: "status_travel_type", Type: bq.StringFieldType},
	}

	table := dataset.Table(tableID)
	metadata := &bq.TableMetadata{
		Schema: schema,
		TimePartitioning: &bq.TimePartitioning{
			Type:  bq.DayPartitioningType,
			Field: "timestamp",
		},
		Clustering: &bq.Clustering{
			Fields: []string{"faction_id", "member_id"},
		},
	}

	if err := table.Create(ctx, metadata); err != nil {
		if !isAlreadyExists(err) {
			fatalf("failed to create table %q: %v", tableID, err)
		}
		fmt.Printf("Table %q already exists — skipping creation.\n", tableID)
	} else {
		fmt.Printf("Created table %q with DAY partitioning on timestamp.\n", tableID)
	}

	fmt.Println()
	fmt.Printf("Setup complete: %s.%s.%s is ready.\n", projectID, datasetID, tableID)
}

func isAlreadyExists(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 409
	}
	return false
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
