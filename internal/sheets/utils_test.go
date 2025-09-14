package sheets

import (
	"testing"
)

// TestParseStringValueComprehensive tests parseStringValue with various inputs
func TestParseStringValueComprehensive(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil input", nil, ""},
		{"string input", "hello", "hello"},
		{"empty string", "", ""},
		{"int input", 42, "42"},
		{"int64 input", int64(123), "123"},
		{"float64 input", 45.67, "45.67"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"complex type", []int{1, 2, 3}, "[1 2 3]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseStringValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// TestParseIntValueComprehensive tests parseIntValue with various inputs
func TestParseIntValueComprehensive(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"nil input", nil, 0},
		{"int input", 42, 42},
		{"int64 input", int64(123), 123},
		{"float64 input", 45.67, 45},
		{"string number", "789", 789},
		{"string non-number", "abc", 0},
		{"empty string", "", 0},
		{"bool input", true, 0},
		{"negative int", -25, -25},
		{"negative float64", -45.99, -45},
		{"string negative", "-100", -100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseIntValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// TestParseInt64ValueComprehensive tests parseInt64Value with various inputs
func TestParseInt64ValueComprehensive(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"nil input", nil, 0},
		{"int64 input", int64(1234567890), 1234567890},
		{"int input", 42, 42},
		{"float64 input", 45.67, 45},
		{"string number", "1234567890", 1234567890},
		{"string non-number", "not_a_number", 0},
		{"empty string", "", 0},
		{"bool input", false, 0},
		{"large int64", int64(9223372036854775807), 9223372036854775807},
		{"negative int64", int64(-1234567890), -1234567890},
		{"string negative", "-987654321", -987654321},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseInt64Value(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// TestParseInt64PointerValueComprehensive tests parseInt64PointerValue with various inputs
func TestParseInt64PointerValueComprehensive(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected *int64
	}{
		{"nil input", nil, nil},
		{"empty string", "", nil},
		{"zero value", 0, nil},
		{"string zero", "0", nil},
		{"valid int64", int64(123), int64Ptr(123)},
		{"valid int", 456, int64Ptr(456)},
		{"valid string", "789", int64Ptr(789)},
		{"valid negative", int64(-123), int64Ptr(-123)},
		{"invalid string", "abc", nil},
		{"bool input", true, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseInt64PointerValue(tc.input)

			if tc.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %d, got nil", *tc.expected)
				} else if *result != *tc.expected {
					t.Errorf("Expected %d, got %d", *tc.expected, *result)
				}
			}
		})
	}
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}

// TestUtilsEdgeCases tests edge cases and boundary conditions
func TestUtilsEdgeCases(t *testing.T) {
	t.Run("parseStringValue with complex types", func(t *testing.T) {
		// Test with more complex types
		testMap := map[string]int{"key": 42}
		result := parseStringValue(testMap)
		expected := "map[key:42]"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("parseIntValue with edge values", func(t *testing.T) {
		// Test with very large float that might overflow
		result := parseIntValue(float64(999999999999999999999))
		// Should handle overflow gracefully
		if result == 0 {
			t.Error("Expected non-zero result for large float, but got 0")
		}
	})

	t.Run("parseInt64Value with boundary values", func(t *testing.T) {
		// Test with string that's too large for int64
		result := parseInt64Value("99999999999999999999999999999")
		if result != 0 {
			t.Errorf("Expected 0 for overflow string, got %d", result)
		}
	})

	t.Run("parseInt64PointerValue with edge cases", func(t *testing.T) {
		// Test with float64 zero
		result := parseInt64PointerValue(float64(0))
		if result != nil {
			t.Error("Expected nil for float64 zero")
		}

		// Test with string that converts to zero
		result = parseInt64PointerValue("0.0")
		if result != nil {
			t.Error("Expected nil for string that converts to zero")
		}
	})
}

// TestTypeAssertions tests the type assertion logic used in the utils
func TestTypeAssertions(t *testing.T) {
	t.Run("string type assertion", func(t *testing.T) {
		var val interface{} = "test_string"
		if s, ok := val.(string); ok {
			if s != "test_string" {
				t.Errorf("Expected 'test_string', got %s", s)
			}
		} else {
			t.Error("Expected successful string assertion")
		}
	})

	t.Run("int type assertion", func(t *testing.T) {
		var val interface{} = 42
		if i, ok := val.(int); ok {
			if i != 42 {
				t.Errorf("Expected 42, got %d", i)
			}
		} else {
			t.Error("Expected successful int assertion")
		}
	})

	t.Run("failed type assertion", func(t *testing.T) {
		var val interface{} = "not_an_int"
		if _, ok := val.(int); ok {
			t.Error("Expected failed int assertion for string")
		}
	})
}

// TestStringConversion tests string conversion patterns used in parseIntValue and parseInt64Value
func TestStringConversion(t *testing.T) {
	t.Run("valid number strings", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected int
		}{
			{"123", 123},
			{"-456", -456},
			{"0", 0},
			{"007", 7}, // Leading zeros
		}

		for _, tc := range testCases {
			result := parseIntValue(tc.input)
			if result != tc.expected {
				t.Errorf("For input %q, expected %d, got %d", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("invalid number strings", func(t *testing.T) {
		invalidStrings := []string{
			"abc",
			"12.34", // parseIntValue uses Atoi, not ParseFloat
			"",
			" 123", // Spaces
			"123 ",
			"1a2b3c",
		}

		for _, input := range invalidStrings {
			result := parseIntValue(input)
			if result != 0 {
				t.Errorf("For invalid input %q, expected 0, got %d", input, result)
			}
		}
	})
}
