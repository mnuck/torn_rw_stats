package travel

import (
	"fmt"
	"testing"
	"testing/quick"
	"time"
)

func TestTravelTimeServiceProperties(t *testing.T) {
	service := NewTravelTimeService()

	destinations := []string{"Mexico", "United Kingdom", "Switzerland", "Hawaii", "Canada"}
	travelTypes := []string{"regular", "airstrip", "business"}

	t.Run("travel time always positive", func(t *testing.T) {
		for _, dest := range destinations {
			for _, tt := range travelTypes {
				if d := service.GetTravelTime(dest, tt); d <= 0 {
					t.Errorf("GetTravelTime(%q, %q) = %v, want > 0", dest, tt, d)
				}
			}
		}
	})

	t.Run("airstrip faster than regular", func(t *testing.T) {
		for _, dest := range destinations {
			regular := service.GetTravelTime(dest, "regular")
			airstrip := service.GetTravelTime(dest, "airstrip")
			if airstrip > regular {
				t.Errorf("airstrip (%v) > regular (%v) for %q", airstrip, regular, dest)
			}
		}
	})

	t.Run("business class fastest", func(t *testing.T) {
		for _, dest := range destinations {
			regular := service.GetTravelTime(dest, "regular")
			airstrip := service.GetTravelTime(dest, "airstrip")
			business := service.GetTravelTime(dest, "business")
			if business > airstrip || business > regular {
				t.Errorf("business (%v) not fastest for %q: regular=%v airstrip=%v", business, dest, regular, airstrip)
			}
		}
	})

	t.Run("known destinations return consistent times", func(t *testing.T) {
		for _, dest := range destinations {
			for _, tt := range travelTypes {
				t1 := service.GetTravelTime(dest, tt)
				t2 := service.GetTravelTime(dest, tt)
				if t1 != t2 {
					t.Errorf("GetTravelTime(%q, %q) not consistent: %v vs %v", dest, tt, t1, t2)
				}
			}
		}
	})

	t.Run("format travel time valid format", func(t *testing.T) {
		// int8 keeps minutes in -128..127 (max ~2h7m), ensuring 2-digit hours and fixed 9-char output.
		if err := quick.Check(func(minutes int8) bool {
			d := time.Duration(minutes) * time.Minute
			f := service.FormatTravelTime(d)
			if len(f) != 9 || f[0] != '\'' || f[3] != ':' || f[6] != ':' {
				return false
			}
			if minutes < 0 && f != "'00:00:00" {
				return false
			}
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("format travel time components consistent", func(t *testing.T) {
		if err := quick.Check(func(hours, minutes uint8) bool {
			h := int(hours % 24)
			m := int(minutes % 60)
			d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
			f := service.FormatTravelTime(d)
			var ph, pm, ps int
			if n, err := fmt.Sscanf(f[1:], "%02d:%02d:%02d", &ph, &pm, &ps); err != nil || n != 3 {
				return false
			}
			return ph == h && pm == m && ps == 0
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("zero and negative durations format to zero", func(t *testing.T) {
		if err := quick.Check(func(minutes uint16) bool {
			d := -time.Duration(minutes) * time.Minute
			return service.FormatTravelTime(d) == "'00:00:00"
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("large durations format correctly", func(t *testing.T) {
		if err := quick.Check(func(days uint8) bool {
			d := time.Duration(days) * 24 * time.Hour
			f := service.FormatTravelTime(d)
			colons := 0
			for _, c := range f {
				if c == ':' {
					colons++
				}
			}
			return colons == 2 && len(f) >= 8
		}, nil); err != nil {
			t.Error(err)
		}
	})
}
