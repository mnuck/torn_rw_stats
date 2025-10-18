package sheets

import (
	"fmt"
	"strconv"
)

// Cell provides type-safe access to Google Sheets cell values.
// The Google Sheets API returns [][]interface{}, which we cannot change.
// This type wraps interface{} to provide type-safe accessors throughout our codebase.
type Cell struct {
	raw interface{}
}

// NewCell creates a Cell from a raw interface{} value from Google Sheets API
func NewCell(raw interface{}) Cell {
	return Cell{raw: raw}
}

// String returns the cell value as a string
func (c Cell) String() string {
	if c.raw == nil {
		return ""
	}
	if s, ok := c.raw.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", c.raw)
}

// Int returns the cell value as an int
func (c Cell) Int() int {
	if c.raw == nil {
		return 0
	}
	switch v := c.raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// Int64 returns the cell value as an int64
func (c Cell) Int64() int64 {
	if c.raw == nil {
		return 0
	}
	switch v := c.raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// Int64Ptr returns the cell value as *int64, or nil if empty
func (c Cell) Int64Ptr() *int64 {
	if c.raw == nil || c.raw == "" {
		return nil
	}
	i := c.Int64()
	if i == 0 {
		return nil
	}
	return &i
}

// IsEmpty returns true if the cell contains nil or empty string
func (c Cell) IsEmpty() bool {
	return c.raw == nil || c.raw == ""
}

// Raw returns the underlying interface{} value for Google Sheets API calls.
// This should only be used at the API boundary.
func (c Cell) Raw() interface{} {
	return c.raw
}
