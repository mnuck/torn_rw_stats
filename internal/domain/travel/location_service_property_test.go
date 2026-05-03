package travel

import (
	"strings"
	"testing"
	"testing/quick"
)

func TestLocationServiceProperties(t *testing.T) {
	service := NewLocationService()

	destinations := []string{"Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"}
	hospitalPairs := [][2]string{
		{"British", "United Kingdom"},
		{"Mexican", "Mexico"},
		{"Swiss", "Switzerland"},
	}

	t.Run("parse location idempotent", func(t *testing.T) {
		if err := quick.Check(func(description string) bool {
			loc1 := service.ParseLocation(description)
			loc2 := service.ParseLocation(loc1)
			return loc1 == loc2
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("okay status returns torn", func(t *testing.T) {
		if got := service.ParseLocation("Okay"); got != "Torn" {
			t.Errorf("ParseLocation(%q) = %q, want %q", "Okay", got, "Torn")
		}
	})

	t.Run("empty description returns empty", func(t *testing.T) {
		if got := service.ParseLocation(""); got != "" {
			t.Errorf("ParseLocation(%q) = %q, want empty", "", got)
		}
	})

	t.Run("traveling patterns extract destination", func(t *testing.T) {
		for _, dest := range destinations {
			patterns := []string{
				"Traveling to " + dest,
				"TRAVELING TO " + strings.ToUpper(dest),
				"traveling to " + strings.ToLower(dest),
			}
			for _, p := range patterns {
				if got := service.ParseLocation(p); got != dest {
					t.Errorf("ParseLocation(%q) = %q, want %q", p, got, dest)
				}
			}
		}
	})

	t.Run("in location patterns extract location", func(t *testing.T) {
		for _, dest := range destinations {
			patterns := []string{
				"In " + dest,
				"IN " + strings.ToUpper(dest),
				"in " + strings.ToLower(dest),
			}
			for _, p := range patterns {
				if got := service.ParseLocation(p); got != dest {
					t.Errorf("ParseLocation(%q) = %q, want %q", p, got, dest)
				}
			}
		}
	})

	t.Run("return travel returns torn", func(t *testing.T) {
		for _, origin := range destinations {
			patterns := []string{
				"Returning to Torn from " + origin,
				"RETURNING TO TORN FROM " + strings.ToUpper(origin),
				"returning to torn from " + strings.ToLower(origin),
			}
			for _, p := range patterns {
				if got := service.ParseLocation(p); got != "Torn" {
					t.Errorf("ParseLocation(%q) = %q, want %q", p, got, "Torn")
				}
			}
		}
	})

	t.Run("hospital patterns map to countries", func(t *testing.T) {
		for _, pair := range hospitalPairs {
			adjective, country := pair[0], pair[1]
			patterns := []string{
				"In a " + adjective + " hospital for 30mins",
				"IN A " + strings.ToUpper(adjective) + " HOSPITAL FOR 2HRS",
				"in a " + strings.ToLower(adjective) + " hospital for 45mins",
			}
			for _, p := range patterns {
				if got := service.ParseLocation(p); got != country {
					t.Errorf("ParseLocation(%q) = %q, want %q", p, got, country)
				}
			}
		}
	})

	t.Run("descriptions containing torn return torn", func(t *testing.T) {
		prefixes := []string{"In ", "At ", "Chilling in ", ""}
		suffixes := []string{" city", " location", ""}
		for _, prefix := range prefixes {
			for _, suffix := range suffixes {
				for _, variant := range []string{"torn", "Torn", "TORN"} {
					p := prefix + variant + suffix
					if got := service.ParseLocation(p); got != "Torn" {
						t.Errorf("ParseLocation(%q) = %q, want %q", p, got, "Torn")
					}
				}
			}
		}
	})

	t.Run("unknown descriptions return themselves", func(t *testing.T) {
		unknowns := []string{"At the beach", "Doing crimes", "Random activity", "Custom status"}
		for _, desc := range unknowns {
			if got := service.ParseLocation(desc); got != desc {
				t.Errorf("ParseLocation(%q) = %q, want %q", desc, got, desc)
			}
		}
	})

	t.Run("travel destination calculation for returns", func(t *testing.T) {
		for _, origin := range destinations {
			desc := "Returning to Torn from " + origin
			got := service.GetTravelDestinationForCalculation(desc, "Torn")
			if got != origin {
				t.Errorf("GetTravelDestinationForCalculation(%q, %q) = %q, want %q", desc, "Torn", got, origin)
			}
		}
	})

	t.Run("travel destination calculation for non-returns", func(t *testing.T) {
		for _, dest := range destinations {
			desc := "Traveling to " + dest
			got := service.GetTravelDestinationForCalculation(desc, dest)
			if got != dest {
				t.Errorf("GetTravelDestinationForCalculation(%q, %q) = %q, want %q", desc, dest, got, dest)
			}
		}
	})
}
