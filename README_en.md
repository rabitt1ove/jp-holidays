# jp-holidays

[![CI](https://github.com/rabitt1ove/jp-holidays/actions/workflows/ci.yml/badge.svg)](https://github.com/rabitt1ove/jp-holidays/actions/workflows/ci.yml)

Japanese national holiday library for Go. Zero dependencies.

[日本語版 README](README.md)

## Features

- Lookup holidays by date, month, year, or date range
- Business day utilities (next/previous business day, counting)
- Custom holiday support (in-memory, per-Calendar instance)
- Thread-safe for concurrent use
- Zero external dependencies — holiday data is compiled into the binary
- Data sourced from the [Cabinet Office of Japan](https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv) (1955–2027)

## Installation

```bash
go get github.com/rabitt1ove/jp-holidays
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    jpholiday "github.com/rabitt1ove/jp-holidays"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func main() {
    t := time.Date(2024, time.January, 1, 0, 0, 0, 0, jst)

    fmt.Println(jpholiday.IsHoliday(t))    // true
    fmt.Println(jpholiday.HolidayName(t))  // 元日
    fmt.Println(jpholiday.IsBusinessDay(t)) // false
}
```

## API

### Holiday Lookup

| Function | Description |
|---|---|
| `IsHoliday(t time.Time) bool` | Check if a date is a holiday |
| `HolidayName(t time.Time) string` | Get the holiday name (empty string if not a holiday) |
| `HolidaysInYear(year int) []Holiday` | Get all holidays in a year |
| `HolidaysInMonth(year int, month time.Month) []Holiday` | Get all holidays in a month |
| `HolidaysBetween(from, to time.Time) []Holiday` | Get all holidays in a date range (inclusive) |
| `Holidays() []Holiday` | Get all holidays in the dataset |

### Business Day Utilities

| Function | Description |
|---|---|
| `IsBusinessDay(t time.Time) bool` | Check if a date is a business day (not weekend, not holiday) |
| `NextBusinessDay(t time.Time) time.Time` | Next business day on or after the date |
| `PreviousBusinessDay(t time.Time) time.Time` | Previous business day on or before the date |
| `BusinessDaysBetween(from, to time.Time) int` | Count business days in range (inclusive) |
| `NextHoliday(t time.Time) (Holiday, bool)` | Next holiday strictly after the date |
| `PreviousHoliday(t time.Time) (Holiday, bool)` | Previous holiday strictly before the date |

### Custom Holidays

| Function | Description |
|---|---|
| `AddCustomHoliday(t time.Time, name string)` | Add a custom holiday |
| `RemoveCustomHoliday(t time.Time)` | Remove a custom holiday |
| `RemoveHoliday(t time.Time)` | Suppress a built-in holiday |
| `RestoreHoliday(t time.Time)` | Restore a suppressed built-in holiday |

If a built-in holiday and a custom holiday exist on the same date, the custom holiday takes precedence.  
In list APIs (`Holidays`, `HolidaysInYear`, `HolidaysInMonth`, `HolidaysBetween`), that date is returned only once (no duplicates).

### Calendar Instance

All functions above are also available as methods on `*Calendar`. Use `New()` to create an isolated instance with its own custom holiday set:

```go
cal := jpholiday.New()
cal.AddCustomHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst), "Company Anniversary")

cal.IsHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst)) // true
jpholiday.IsHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst)) // false (default calendar)
```

## Types

```go
type Holiday struct {
    Date time.Time // Midnight UTC
    Name string    // Japanese name (e.g., "元日")
}
```

## Timezone Handling

All `time.Time` inputs are **converted to JST (Asia/Tokyo, UTC+9)** before extracting the calendar date. This ensures correct results based on the Japanese calendar regardless of the input timezone.

```go
// 2023-12-31 20:00 UTC = 2024-01-01 05:00 JST → recognized as New Year's Day
utcTime := time.Date(2023, 12, 31, 20, 0, 0, 0, time.UTC)
jpholiday.IsHoliday(utcTime)   // true (it's January 1 in JST)
jpholiday.HolidayName(utcTime) // "元日"
```

Business day checks (`IsBusinessDay`, etc.) also determine the day of the week in JST.

## Benchmarks

Measured on Apple M2 Pro (`go test -bench=. -benchmem`).
ns/op = nanoseconds per operation (billionths of a second).

| Function | Time | Allocations |
|---|---|---|
| `IsHoliday` | ~20 ns/op | 0 allocs |
| `HolidayName` | ~20 ns/op | 0 allocs |
| `IsBusinessDay` | ~21 ns/op | 0 allocs |
| `NextBusinessDay` | ~200 ns/op | 0 allocs |
| `BusinessDaysBetween` (1 month) | ~1,300 ns/op | 0 allocs |
| `BusinessDaysBetween` (1 year) | ~16,000 ns/op | 0 allocs |
| `HolidaysInYear` | ~12,000 ns/op | 9 allocs |
| `NextHoliday` / `PreviousHoliday` | ~11,000 ns/op | 0 allocs |

Run benchmarks yourself:

```bash
go test -bench=. -benchmem ./...
```

## Data Source

Holiday data is sourced from the Cabinet Office of Japan (内閣府):
https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv

- **Data range**: 1955 (Showa 30) to 2027 (Reiwa 9) — updated as the Cabinet Office publishes new data
- **Update frequency**: Checked weekly (every Sunday) via GitHub Actions
- **Update method**: The CSV URL is resolved dynamically using the [e-Gov Data Portal CKAN API](https://data.e-gov.go.jp/data/api_guide) (recommended by the [Digital Agency of Japan](https://www.digital.go.jp/en/resources/open_data)). If the API is unavailable, direct URLs are used as fallback.
- **Update flow**: When new data is detected, a pull request is automatically created for human review before merging.

### Data Attribution

This library uses data published by the Cabinet Office of Japan (内閣府) on the [e-Gov Data Portal](https://data.e-gov.go.jp/).

> Source: Cabinet Office of Japan, "National Holidays CSV" (内閣府「国民の祝日」について)
> https://www8.cao.go.jp/chosei/shukujitsu/gaiyou.html

The data is provided under terms compatible with [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). Users of this library should be aware that the underlying holiday data originates from this government source.

## License

MIT
