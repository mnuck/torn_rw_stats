package bigquery

import (
	"testing"
	"time"

	bq "cloud.google.com/go/bigquery"

	"torn_rw_stats/internal/app"
)

func TestStateRecordRowSave_FieldMapping(t *testing.T) {
	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	until := time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC)

	record := app.StateRecord{
		Timestamp:         ts,
		MemberID:          "12345",
		MemberName:        "Tester",
		FactionID:         "999",
		FactionName:       "TestFaction",
		LastActionStatus:  "Online",
		StatusDescription: "Okay",
		StatusState:       "okay",
		StatusUntil:       until,
		StatusTravelType:  "",
	}

	row := &stateRecordRow{record: record}
	values, insertID, err := row.Save()

	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if insertID != "" {
		t.Errorf("Save() insertID = %q, want empty string", insertID)
	}

	checks := map[string]bq.Value{
		"member_id":          "12345",
		"member_name":        "Tester",
		"faction_id":         "999",
		"faction_name":       "TestFaction",
		"last_action_status": "Online",
		"status_description": "Okay",
		"status_state":       "okay",
		"status_travel_type": "",
	}
	for field, want := range checks {
		if got := values[field]; got != want {
			t.Errorf("Save()[%q] = %v, want %v", field, got, want)
		}
	}

	// timestamp should be set to UTC
	if gotTS, ok := values["timestamp"].(time.Time); !ok {
		t.Errorf("Save()[timestamp] is not a time.Time, got %T", values["timestamp"])
	} else if !gotTS.Equal(ts.UTC()) {
		t.Errorf("Save()[timestamp] = %v, want %v", gotTS, ts.UTC())
	}

	// status_until should be present because it is non-zero
	if gotUntil, ok := values["status_until"].(time.Time); !ok {
		t.Errorf("Save()[status_until] is not a time.Time, got %T", values["status_until"])
	} else if !gotUntil.Equal(until.UTC()) {
		t.Errorf("Save()[status_until] = %v, want %v", gotUntil, until.UTC())
	}
}

func TestStateRecordRowSave_ZeroStatusUntilOmitted(t *testing.T) {
	record := app.StateRecord{
		Timestamp: time.Now().UTC(),
		MemberID:  "1",
		// StatusUntil is zero value
	}

	row := &stateRecordRow{record: record}
	values, _, err := row.Save()
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	if _, present := values["status_until"]; present {
		t.Error("Save() should not include status_until when StatusUntil is zero, but it was present")
	}
}

func TestStateRecordRowSave_NonZeroStatusUntilIncluded(t *testing.T) {
	until := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	record := app.StateRecord{
		Timestamp:   time.Now().UTC(),
		MemberID:    "2",
		StatusUntil: until,
	}

	row := &stateRecordRow{record: record}
	values, _, err := row.Save()
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	if _, present := values["status_until"]; !present {
		t.Error("Save() should include status_until when StatusUntil is non-zero, but it was absent")
	}
}
