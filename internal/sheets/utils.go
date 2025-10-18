package sheets

// Helper functions for parsing sheet values
// These wrap interface{} values from Google Sheets API in type-safe Cell wrappers

// parseStringValue converts a raw Google Sheets cell value to string
func parseStringValue(val interface{}) string {
	return NewCell(val).String()
}

// parseIntValue converts a raw Google Sheets cell value to int
func parseIntValue(val interface{}) int {
	return NewCell(val).Int()
}

// parseInt64Value converts a raw Google Sheets cell value to int64
func parseInt64Value(val interface{}) int64 {
	return NewCell(val).Int64()
}

// parseInt64PointerValue converts a raw Google Sheets cell value to *int64
func parseInt64PointerValue(val interface{}) *int64 {
	return NewCell(val).Int64Ptr()
}
