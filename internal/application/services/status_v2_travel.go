package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"torn_rw_stats/internal/app"
)

// TravelInfo holds travel-related data for a member including departure time,
// arrival times (standard and business class), and countdown to arrival.
type TravelInfo struct {
	Departure       string
	Arrival         string
	BusinessArrival string
	Countdown       string
}

// calculateTravelInfo handles all travel-related calculations and preserves manual adjustments
func (s *StatusV2Service) calculateTravelInfo(ctx context.Context, stateRecord app.StateRecord, existing *app.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time, location string) TravelInfo {
	if stateRecord.StatusState != "Traveling" {
		return TravelInfo{} // Clear travel data for non-traveling members
	}

	memberKey := fmt.Sprintf("%s_%s", stateRecord.FactionID, stateRecord.MemberID)
	departure := s.calculateDeparture(memberKey, existing, departureMap, currentTime)

	// Calculate arrival times using TravelTimeService
	arrival, businessArrival, countdown := s.calculateArrivalTimes(ctx, stateRecord, existing, departure, location, currentTime)

	// Preserve manual adjustments
	return s.applyManualAdjustments(existing, TravelInfo{
		Departure:       departure,
		Arrival:         arrival,
		BusinessArrival: businessArrival,
		Countdown:       countdown,
	})
}

// calculateDeparture determines the departure time for a traveling member
func (s *StatusV2Service) calculateDeparture(memberKey string, existing *app.StatusV2Record, departureMap map[string]time.Time, currentTime time.Time) string {
	if departureTime, hasDeparture := departureMap[memberKey]; hasDeparture {
		return departureTime.Format("2006-01-02 15:04:05")
	}
	if existing != nil && existing.Departure != "" {
		return existing.Departure
	}
	return currentTime.Format("2006-01-02 15:04:05")
}

// calculateArrivalTimes uses TravelTimeService to calculate arrival times and countdown
func (s *StatusV2Service) calculateArrivalTimes(ctx context.Context, stateRecord app.StateRecord, existing *app.StatusV2Record, departure, location string, currentTime time.Time) (string, string, string) {
	if departure == "" {
		return "", "", ""
	}

	memberID := 0
	if id, err := strconv.Atoi(stateRecord.MemberID); err == nil {
		memberID = id
	}

	existingArrival := ""
	if existing != nil {
		existingArrival = existing.Arrival
	}

	travelData := s.travelTimeService.CalculateTravelTimesFromDeparture(
		ctx,
		memberID,
		location,
		departure,
		existingArrival,
		stateRecord.StatusTravelType,
		currentTime,
		s.locationService,
		stateRecord.StatusDescription,
	)

	if travelData != nil {
		return travelData.Arrival, travelData.BusinessArrival, travelData.Countdown
	}
	return "", "", ""
}

// applyManualAdjustments preserves any manual adjustments from existing data
func (s *StatusV2Service) applyManualAdjustments(existing *app.StatusV2Record, calculated TravelInfo) TravelInfo {
	if existing == nil {
		return calculated
	}

	result := calculated
	if existing.Departure != "" {
		result.Departure = existing.Departure
	}
	if existing.Arrival != "" {
		result.Arrival = existing.Arrival
	}
	if existing.BusinessArrival != "" {
		result.BusinessArrival = existing.BusinessArrival
	}
	return result
}
