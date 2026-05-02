package mocks

import (
	"context"

	"torn_rw_stats/internal/app"
)

// MockBigQueryClient is a test double for bigquery.Client
type MockBigQueryClient struct {
	// Errors to return
	InsertStateRecordsError        error
	QueryLatestStatePerMemberError error

	// Responses to return
	QueryLatestStatePerMemberResponse []app.StateRecord

	// Call tracking
	InsertStateRecordsCalled            bool
	InsertStateRecordsCalledWith        []app.StateRecord
	QueryLatestStatePerMemberCalled     bool
	QueryLatestStatePerMemberCalledWith []string
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

func (m *MockBigQueryClient) QueryLatestStatePerMember(_ context.Context, memberIDs []string) ([]app.StateRecord, error) {
	m.QueryLatestStatePerMemberCalled = true
	m.QueryLatestStatePerMemberCalledWith = memberIDs
	return m.QueryLatestStatePerMemberResponse, m.QueryLatestStatePerMemberError
}

// Reset clears all call tracking and errors
func (m *MockBigQueryClient) Reset() {
	m.InsertStateRecordsError = nil
	m.InsertStateRecordsCalled = false
	m.InsertStateRecordsCalledWith = nil
	m.QueryLatestStatePerMemberError = nil
	m.QueryLatestStatePerMemberResponse = nil
	m.QueryLatestStatePerMemberCalled = false
	m.QueryLatestStatePerMemberCalledWith = nil
}
