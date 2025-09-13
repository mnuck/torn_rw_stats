package processing

import (
	"reflect"
	"strings"
	"testing"

	"torn_rw_stats/internal/app"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestStateChangeDetectionServiceProperties uses property-based testing
func TestStateChangeDetectionServiceProperties(t *testing.T) {
	service := NewStateChangeDetectionService(nil)

	properties := gopter.NewProperties(nil)

	// Property: Hospital normalization should be idempotent
	properties.Property("hospital normalization idempotent", prop.ForAll(
		func(description string) bool {
			normalized1 := service.NormalizeHospitalDescription(description)
			normalized2 := service.NormalizeHospitalDescription(normalized1)
			return normalized1 == normalized2
		},
		gen.AlphaString(),
	))

	// Property: Hospital patterns should always normalize to "In hospital"
	properties.Property("hospital patterns normalize correctly", prop.ForAll(
		func(countryName string, timeStr string) bool {
			// Generate various hospital description patterns
			patterns := []string{
				"In hospital for " + timeStr,
				"In a " + countryName + " hospital for " + timeStr,
				"IN HOSPITAL FOR " + strings.ToUpper(timeStr),
				"in a " + strings.ToLower(countryName) + " hospital for " + timeStr,
			}

			for _, pattern := range patterns {
				normalized := service.NormalizeHospitalDescription(pattern)
				if normalized != "In hospital" {
					return false
				}
			}
			return true
		},
		gen.OneConstOf("British", "Mexican", "South African", "Swiss", "Chinese", "Japanese"),
		gen.OneConstOf("30mins", "2hrs", "1hr 30mins", "45mins", "3hrs 15mins"),
	))

	// Property: Non-hospital descriptions should remain unchanged
	properties.Property("non-hospital descriptions unchanged", prop.ForAll(
		func(description string) bool {
			// Only test strings that don't match hospital patterns
			lowerDesc := strings.ToLower(description)
			if strings.Contains(lowerDesc, "hospital") {
				return true // Skip hospital descriptions
			}

			normalized := service.NormalizeHospitalDescription(description)
			return normalized == description
		},
		gen.OneConstOf("Okay", "Traveling to Mexico", "At the gym", "Flying", "Online", "Offline"),
	))

	// Property: Status change detection should be symmetric for identical members
	properties.Property("identical members have no status change", prop.ForAll(
		func(member app.FactionMember) bool {
			hasChange := service.HasStatusChanged(member, member)
			return !hasChange
		},
		genFactionMember(),
	))

	// Property: Any change in LastAction.Status should trigger status change
	properties.Property("last action status change detected", prop.ForAll(
		func(member app.FactionMember, newStatus string) bool {
			// Create a copy with different LastAction.Status
			modifiedMember := member
			modifiedMember.LastAction.Status = newStatus

			// Only test when the status actually changes
			if member.LastAction.Status == newStatus {
				return true // Skip if status is the same
			}

			hasChange := service.HasStatusChanged(member, modifiedMember)
			return hasChange
		},
		genFactionMember(),
		gen.OneConstOf("Online", "Offline", "Idle", "Away"),
	))

	// Property: Hospital countdown changes should NOT trigger status change
	properties.Property("hospital countdown ignored", prop.ForAll(
		func(baseTime, newTime string) bool {
			oldMember := app.FactionMember{
				Status: app.MemberStatus{
					Description: "In hospital for " + baseTime,
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{Status: "Idle"},
			}

			newMember := app.FactionMember{
				Status: app.MemberStatus{
					Description: "In hospital for " + newTime,
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{Status: "Idle"},
			}

			hasChange := service.HasStatusChanged(oldMember, newMember)
			return !hasChange // Should NOT have changed
		},
		gen.OneConstOf("30mins", "2hrs", "1hr 30mins"),
		gen.OneConstOf("25mins", "1hr 55mins", "1hr 25mins"),
	))

	// Property: Country-specific hospital countdown changes should NOT trigger status change
	properties.Property("country hospital countdown ignored", prop.ForAll(
		func(country, baseTime, newTime string) bool {
			oldMember := app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a " + country + " hospital for " + baseTime,
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{Status: "Idle"},
			}

			newMember := app.FactionMember{
				Status: app.MemberStatus{
					Description: "In a " + country + " hospital for " + newTime,
					State:       "Hospital",
					Color:       "red",
				},
				LastAction: app.LastAction{Status: "Idle"},
			}

			hasChange := service.HasStatusChanged(oldMember, newMember)
			return !hasChange // Should NOT have changed
		},
		gen.OneConstOf("British", "Mexican", "South African", "Swiss"),
		gen.OneConstOf("33mins", "2hrs 15mins", "45mins"),
		gen.OneConstOf("28mins", "2hrs 10mins", "40mins"),
	))

	properties.TestingRun(t)
}

// genFactionMember generates a faction member with various states
func genFactionMember() gopter.Gen {
	return gen.Struct(reflect.TypeOf(app.FactionMember{}), map[string]gopter.Gen{
		"Name":  gen.AlphaString(),
		"Level": gen.IntRange(1, 100),
		"Status": gen.Struct(reflect.TypeOf(app.MemberStatus{}), map[string]gopter.Gen{
			"Description": gen.OneConstOf(
				"Okay",
				"Traveling to Mexico",
				"In hospital for 30mins",
				"In a British hospital for 2hrs",
				"At the gym",
			),
			"State": gen.OneConstOf("Okay", "Hospital", "Traveling", "Gym"),
			"Color": gen.OneConstOf("green", "red", "blue", "orange"),
		}),
		"LastAction": gen.Struct(reflect.TypeOf(app.LastAction{}), map[string]gopter.Gen{
			"Status": gen.OneConstOf("Online", "Offline", "Idle", "Away"),
		}),
	})
}
