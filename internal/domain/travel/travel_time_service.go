package travel

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// TravelTimeService handles travel time calculations and formatting
type TravelTimeService struct {
	regularTimes  map[string]int
	airstripTimes map[string]int
	businessTimes map[string]int
}

// NewTravelTimeService creates a new travel time service with predefined travel times
func NewTravelTimeService() *TravelTimeService {
	return &TravelTimeService{
		regularTimes: map[string]int{
			"Mexico":         26,
			"Cayman Islands": 35,
			"Canada":         41,
			"Hawaii":         134, // 2h 14m
			"United Kingdom": 159, // 2h 39m
			"Argentina":      167, // 2h 47m
			"Switzerland":    175, // 2h 55m
			"Japan":          225, // 3h 45m
			"China":          242, // 4h 2m
			"UAE":            271, // 4h 31m
			"South Africa":   297, // 4h 57m
		},
		airstripTimes: map[string]int{
			"Mexico":         18,
			"Cayman Islands": 25,
			"Canada":         29,
			"Hawaii":         94,  // 1h 34m
			"United Kingdom": 111, // 1h 51m
			"Argentina":      117, // 1h 57m
			"Switzerland":    123, // 2h 3m
			"Japan":          158, // 2h 38m
			"China":          169, // 2h 49m
			"UAE":            190, // 3h 10m
			"South Africa":   208, // 3h 28m
		},
		businessTimes: map[string]int{
			// Business class travel times (fastest option)
			// Note: API detection for business class is not yet implemented
			"Mexico":         8,
			"Cayman Islands": 11,
			"Canada":         12,
			"Hawaii":         40,
			"United Kingdom": 48,
			"Argentina":      50,
			"Switzerland":    53,
			"Japan":          68, // 1h 8m
			"China":          72, // 1h 12m
			"UAE":            81, // 1h 21m
			"South Africa":   89, // 1h 29m
		},
	}
}

// TravelTimeData holds calculated travel timing information
type TravelTimeData struct {
	Departure       string
	Arrival         string
	BusinessArrival string // Alternative arrival time assuming business class for "standard" travel
	Countdown       string
}

// GetTravelTime returns travel duration based on destination and travel type
// Travel types: "regular" (default), "airstrip" (private jet), "business" (business class)
// Note: Business class detection from API is not currently implemented - this prepares the infrastructure
func (tts *TravelTimeService) GetTravelTime(destination string, travelType string) time.Duration {
	var minutes int
	switch travelType {
	case "airstrip":
		minutes = tts.airstripTimes[destination]
	case "business":
		minutes = tts.businessTimes[destination]
	default:
		minutes = tts.regularTimes[destination]
	}

	if minutes == 0 {
		log.Warn().
			Str("destination", destination).
			Str("travel_type", travelType).
			Msg("Unknown travel destination, using default time")
		return 30 * time.Minute // Default fallback
	}

	return time.Duration(minutes) * time.Minute
}

// FormatTravelTime formats duration as HH:MM:SS
// Prefixed with apostrophe to force Google Sheets to treat as text (prevents fraction conversion)
func (tts *TravelTimeService) FormatTravelTime(d time.Duration) string {
	if d <= 0 {
		return "'00:00:00"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("'%02d:%02d:%02d", hours, minutes, seconds)
}

// CalculateTravelTimes calculates travel departure, arrival and countdown for a user
func (tts *TravelTimeService) CalculateTravelTimes(ctx context.Context, userID int, destination string, travelType string, currentTime time.Time, updateInterval time.Duration) *TravelTimeData {
	// Get travel duration based on destination and travel type
	travelDuration := tts.GetTravelTime(destination, travelType)

	// Assume they departed 50% through the last cycle interval
	cycleInterval := updateInterval
	estimatedDepartureTime := currentTime.Add(-cycleInterval / 2)
	arrivalTime := estimatedDepartureTime.Add(travelDuration)

	// Calculate business class arrival time for "standard" travel type
	var businessArrival string
	if travelType == "standard" || travelType == "" {
		businessDuration := tts.GetTravelTime(destination, "business")
		businessArrivalTime := estimatedDepartureTime.Add(businessDuration)
		businessArrival = businessArrivalTime.UTC().Format("2006-01-02 15:04:05")
	}

	// Calculate countdown
	timeRemaining := arrivalTime.Sub(currentTime)
	countdown := tts.FormatTravelTime(timeRemaining)

	// If they've already arrived, show as completed
	if timeRemaining <= 0 {
		countdown = "00:00:00"
	}

	log.Debug().
		Int("user_id", userID).
		Str("destination", destination).
		Str("travel_type", travelType).
		Dur("travel_duration", travelDuration).
		Str("business_arrival", businessArrival).
		Str("countdown", countdown).
		Msg("Calculated travel times")

	return &TravelTimeData{
		Departure:       estimatedDepartureTime.UTC().Format("2006-01-02 15:04:05"),
		Arrival:         arrivalTime.UTC().Format("2006-01-02 15:04:05"),
		BusinessArrival: businessArrival,
		Countdown:       countdown,
	}
}

// CalculateTravelTimesFromDeparture calculates arrival and countdown based on existing departure time
func (tts *TravelTimeService) CalculateTravelTimesFromDeparture(ctx context.Context, userID int, destination, departureStr, existingArrivalStr string, travelType string, currentTime time.Time, locationService *LocationService, statusDescription string) *TravelTimeData {
	// Parse existing departure time as UTC to match how times are stored
	departureTime, err := time.ParseInLocation("2006-01-02 15:04:05", departureStr, time.UTC)
	if err != nil {
		log.Warn().
			Err(err).
			Str("departure_str", departureStr).
			Int("user_id", userID).
			Msg("Failed to parse existing departure time")
		return nil
	}

	var arrivalTime time.Time
	var travelDuration time.Duration

	// If we have an existing arrival time, use it instead of recalculating
	if existingArrivalStr != "" {
		if parsedArrival, err := time.ParseInLocation("2006-01-02 15:04:05", existingArrivalStr, time.UTC); err == nil {
			arrivalTime = parsedArrival
			travelDuration = arrivalTime.Sub(departureTime)
		} else {
			log.Warn().
				Err(err).
				Str("existing_arrival_str", existingArrivalStr).
				Msg("Failed to parse existing arrival time, falling back to calculation")
		}
	}

	// If no existing arrival time or parsing failed, calculate from travel duration
	if arrivalTime.IsZero() {
		travelDestination := locationService.GetTravelDestinationForCalculation(statusDescription, destination)
		travelDuration = tts.GetTravelTime(travelDestination, travelType)
		arrivalTime = departureTime.Add(travelDuration)
	}

	// Calculate business class arrival time for "standard" travel type
	var businessArrivalTime time.Time
	var businessArrival string
	if travelType == "standard" || travelType == "" {
		travelDestination := locationService.GetTravelDestinationForCalculation(statusDescription, destination)
		businessDuration := tts.GetTravelTime(travelDestination, "business")
		businessArrivalTime = departureTime.Add(businessDuration)
		businessArrival = businessArrivalTime.UTC().Format("2006-01-02 15:04:05")
	}

	// Calculate countdown
	timeRemaining := arrivalTime.Sub(currentTime)
	countdown := tts.FormatTravelTime(timeRemaining)

	// If they've already arrived, show as completed
	if timeRemaining <= 0 {
		countdown = "00:00:00"
	}

	log.Debug().
		Int("user_id", userID).
		Str("destination", destination).
		Str("travel_type", travelType).
		Dur("travel_duration", travelDuration).
		Str("departure", departureStr).
		Str("arrival", arrivalTime.UTC().Format("2006-01-02 15:04:05")).
		Str("business_arrival", businessArrival).
		Str("countdown", countdown).
		Msg("Recalculated travel times from existing departure")

	return &TravelTimeData{
		Departure:       departureStr, // Keep original departure time
		Arrival:         arrivalTime.UTC().Format("2006-01-02 15:04:05"),
		BusinessArrival: businessArrival,
		Countdown:       countdown,
	}
}
