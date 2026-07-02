package domain

// The types below define the application's own published JSON export format.
// The JSON tags here are owned by this application (consumed by the tactical
// dashboard), not by any external API.

// JSONMember represents a member in the JSON export format
type JSONMember struct {
	Name            string `json:"Name"`
	MemberID        string `json:"MemberID"`
	Level           int    `json:"Level"`
	State           string `json:"State"`
	Status          string `json:"Status,omitempty"`
	Countdown       string `json:"Countdown,omitempty"`
	Until           string `json:"Until,omitempty"`
	Arrival         string `json:"Arrival,omitempty"`
	BusinessArrival string `json:"BusinessArrival,omitempty"`
}

// LocationData represents the traveling and located members for a location
type LocationData struct {
	Traveling []JSONMember `json:"Traveling"`
	LocatedIn []JSONMember `json:"Located In"`
}

// StatusV2JSON represents the complete JSON export structure
type StatusV2JSON struct {
	Faction   string                  `json:"Faction"`
	Updated   string                  `json:"Updated"`
	Interval  int                     `json:"Interval"` // Update interval in seconds
	Locations map[string]LocationData `json:"Locations"`
}
