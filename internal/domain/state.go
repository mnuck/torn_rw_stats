package domain

import "time"

// TravelRecord represents a member's travel status record
type TravelRecord struct {
	Name            string
	Level           int
	Location        string
	State           string
	Departure       string
	Countdown       string
	Arrival         string
	BusinessArrival string // Alternative arrival time assuming business class
}

// StateChangeRecord represents a member's state change record
type StateChangeRecord struct {
	Timestamp            time.Time
	MemberID             int
	MemberName           string
	FactionName          string
	FactionID            int
	LastActionStatus     string
	StatusDescription    string
	StatusState          string
	StatusColor          string
	StatusDetails        string
	StatusUntil          string
	StatusTravelType     string
	StatusPlaneImageType string
	PreviousState        string
	CurrentState         string
	PreviousLastAction   string
	CurrentLastAction    string
}

// StateRecord represents a point-in-time snapshot of a member's state
type StateRecord struct {
	Timestamp         time.Time
	MemberName        string
	MemberID          string
	FactionName       string
	FactionID         string
	LastActionStatus  string
	StatusDescription string
	StatusState       string
	StatusUntil       time.Time
	StatusTravelType  string
}

// StatusV2Record represents a member's data for Status v2 sheets
type StatusV2Record struct {
	Name            string
	MemberID        string
	Level           int
	State           string    // LastActionStatus from StateRecord
	Status          string    // StatusDescription from StateRecord
	Location        string    // Destination for traveling, otherwise current location
	Countdown       string    // Calculated from StatusUntil field
	Departure       string    // Manual adjustment preserved
	Arrival         string    // Manual adjustment preserved
	BusinessArrival string    // Alternative arrival time assuming business class
	Until           time.Time // StatusUntil timestamp from StateRecord
}
