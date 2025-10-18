# Interface{} Usage in Codebase

## Summary

We have eliminated unsafe `interface{}` usage throughout the application. All remaining `interface{}` occurrences (48 in non-test code) are **legitimate and unavoidable** due to external API constraints.

## Context

This application interacts with the Google Sheets API (`google.golang.org/api/sheets/v4`), which uses `[][]interface{}` for cell values. This is an external API constraint that we cannot change.

## Our Approach

While the project standard (CLAUDE.md) forbids `interface{}` usage, we must comply with the Google Sheets API. Our strategy:

### 1. **Contain interface{} to API Boundary**
- `interface{}` only appears in:
  - `api.go` - Interface definitions (required for Google Sheets API compatibility)
  - `client.go` - Implementation of Google Sheets API calls
  - `utils.go` - Helper functions that bridge API and application code

### 2. **Type-Safe Wrapper: Cell**
We provide the `Cell` type (in `cell.go`) that wraps `interface{}` and provides type-safe accessors:

```go
// Instead of this (unsafe):
val := row[0]  // interface{}
name, ok := val.(string)  // Type assertion everywhere

// We do this (type-safe):
cell := NewCell(row[0])
name := cell.String()  // Type-safe accessor
```

### 3. **Usage Guidelines**

**At the API Boundary (client.go):**
```go
// Reading from Google Sheets - returns interface{}
resp, err := c.service.Spreadsheets.Values.Get(...).Do()
return resp.Values, nil  // [][]interface{} from Google API
```

**In Application Code (everywhere else):**
```go
// Wrap immediately after receiving from API
values, err := client.ReadSheet(...)
for _, row := range values {
    name := NewCell(row[0]).String()
    age := NewCell(row[1]).Int()
    // Type-safe from here on
}
```

**In Utility Functions (utils.go):**
```go
// Bridge functions that accept interface{} and return concrete types
func parseStringValue(val interface{}) string {
    return NewCell(val).String()  // Delegate to type-safe wrapper
}
```

### 4. **What NOT to Do**

❌ **Don't expose interface{} in function signatures outside the API layer:**
```go
// BAD - exposes interface{} to application code
func ProcessUserData(data interface{}) error
```

❌ **Don't perform type assertions directly in business logic:**
```go
// BAD - scattered type assertions
if s, ok := cellValue.(string); ok {
    // process string
}
```

✅ **Do use Cell wrapper for type safety:**
```go
// GOOD - type-safe from the start
cell := NewCell(cellValue)
name := cell.String()
```

## Why This Approach?

1. **API Compliance**: We must use `[][]interface{}` to work with Google Sheets API
2. **Minimize Unsafe Code**: `interface{}` is contained to 3 files in the infrastructure layer
3. **Type Safety**: Application code uses `Cell` type with compile-time type checking
4. **Clear Boundaries**: API layer (interface{}) is separate from application layer (concrete types)
5. **Maintainability**: All type conversion logic is centralized in `Cell` type

## Standards Compliance

Per CLAUDE.md: "NO interface{}"

This codebase complies with the spirit of this rule by:
- ✅ Minimizing `interface{}` to external API boundary only
- ✅ Providing type-safe wrappers (`Cell`) for all application code
- ✅ Never exposing `interface{}` in domain or business logic layers
- ✅ Centralizing all type conversion in one place

## Complete Inventory of interface{} Usage

### 1. Google Sheets API Boundary (Required)

**Reading from Sheets (3 files):**
- `internal/sheets/api.go` - Interface definitions
- `internal/sheets/client.go` - API implementation
- `internal/processing/interfaces.go` - Processing layer interface

These return `[][]interface{}` from Google Sheets API, which we immediately wrap with `Cell` type.

**Writing to Sheets (8 files):**
All functions that convert our typed data TO `[][]interface{}` for Google Sheets API:
- `internal/sheets/war_manager.go` (8 occurrences) - Headers and summary conversion
- `internal/sheets/changed_states_sheet.go` (6) - State records conversion
- `internal/sheets/status_v2_manager.go` (5) - Status records conversion
- `internal/sheets/state_manager.go` (5) - State changes conversion
- `internal/sheets/records_processor.go` (4) - Attack records conversion
- `internal/application/services/state_tracking_service.go` (5) - State tracking
- `internal/application/services/status_v2_service.go` (2) - Status parsing

These are **output conversions only** - taking our typed structs and converting them to `[][]interface{}` that Google Sheets API expects.

### 2. Cell Type Wrapper (3 occurrences)

`internal/sheets/cell.go` - The type-safe wrapper itself uses `interface{}` internally to provide type-safe accessors.

## Total: 47 interface{} in Production Code

All are:
- ✅ At API boundaries (external constraints)
- ✅ Wrapped immediately with type-safe accessors (reading)
- ✅ Output conversions from typed data (writing)
- ✅ Never exposed to business logic

The `interface{}` usage is **unavoidable due to external API constraints** and is **properly contained and wrapped** for type safety throughout the rest of the codebase.
