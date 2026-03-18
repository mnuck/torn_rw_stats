package mocks

import (
	"context"

	"torn_rw_stats/internal/app"
)

// MockBigQueryClient is a test double for bigquery.Client
type MockBigQueryClient struct {
	// Error to return
	InsertStateRecordsError error

	// Call tracking
	InsertStateRecordsCalled     bool
	InsertStateRecordsCalledWith []app.StateRecord
}

// NewMockBigQueryClient creates a new mock BigQuery client
func NewMockBigQueryClient() *MockBigQueryClient {
	return &MockBigQueryClient{}
}

func (m *MockBigQueryClient) InsertStateRecords(_ context.Context, records []app.StateRecord) error {
	m.InsertStateRecordsCalled = true
	m.InsertStateRecordsCalledWith = records
	return m.InsertStateRecordsError
}

// Reset clears all call tracking and errors
func (m *MockBigQueryClient) Reset() {
	m.InsertStateRecordsError = nil
	m.InsertStateRecordsCalled = false
	m.InsertStateRecordsCalledWith = nil
}
