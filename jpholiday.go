// Package jpholiday provides Japanese national holiday lookups and business day utilities.
//
// Holiday data is sourced from the Cabinet Office of Japan (内閣府) and compiled
// into this package at build time. The data requires no runtime parsing.
//
// All time.Time inputs are normalized to JST (Asia/Tokyo, UTC+9) before
// extracting the calendar date, so the correct Japanese holiday is returned
// regardless of the input timezone.
//
// Basic usage with package-level functions:
//
//	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
//	t := time.Date(2024, 1, 1, 0, 0, 0, 0, jst)
//	jpholiday.IsHoliday(t)    // true
//	jpholiday.HolidayName(t)  // "元日"
//
// For isolated custom holiday management, create a Calendar instance:
//
//	cal := jpholiday.New()
//	cal.AddCustomHoliday(t, "会社記念日")
package jpholiday

import (
	"sort"
	"sync"
	"time"
)

// Holiday represents a single holiday entry.
type Holiday struct {
	Date time.Time // The date of the holiday (midnight UTC).
	Name string    // The Japanese name of the holiday (e.g., "元日").
}

// Calendar holds holiday data and supports custom holidays.
// Create one with [New]. All methods are safe for concurrent use.
type Calendar struct {
	mu      sync.RWMutex
	custom  map[date]string
	removed map[date]bool
}

// New creates a new Calendar backed by the built-in holiday dataset.
func New() *Calendar {
	return &Calendar{
		custom:  make(map[date]string),
		removed: make(map[date]bool),
	}
}

// defaultCal is the package-level calendar used by top-level functions.
var defaultCal = New()

// lookup returns the holiday name for a date, checking custom holidays first,
// then built-in holidays (unless removed).
func (c *Calendar) lookup(d date) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, ok := c.custom[d]; ok {
		return name, true
	}
	if c.removed[d] {
		return "", false
	}
	if name, ok := builtinHolidays[d]; ok {
		return name, true
	}
	return "", false
}

// IsHoliday reports whether the given date is a holiday (built-in or custom).
// The input time is converted to JST (Asia/Tokyo, UTC+9) before extracting
// the calendar date, so the result is always correct for the Japanese calendar
// regardless of the input timezone.
func (c *Calendar) IsHoliday(t time.Time) bool {
	_, ok := c.lookup(dateFromTime(t))
	return ok
}

// HolidayName returns the holiday name for the given date, or an empty string
// if it is not a holiday.
func (c *Calendar) HolidayName(t time.Time) string {
	name, _ := c.lookup(dateFromTime(t))
	return name
}

// HolidaysInYear returns all holidays in the given year, sorted by date.
func (c *Calendar) HolidaysInYear(year int) []Holiday {
	from := date{year: year, month: time.January, day: 1}
	to := date{year: year, month: time.December, day: 31}
	return c.holidaysInRange(from, to)
}

// HolidaysInMonth returns all holidays in the given year and month, sorted by date.
func (c *Calendar) HolidaysInMonth(year int, month time.Month) []Holiday {
	from := date{year: year, month: month, day: 1}
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	to := date{year: year, month: month, day: lastDay}
	return c.holidaysInRange(from, to)
}

// HolidaysBetween returns all holidays in the range [from, to] inclusive,
// sorted by date. If from is after to, returns nil.
func (c *Calendar) HolidaysBetween(from, to time.Time) []Holiday {
	fromD := dateFromTime(from)
	toD := dateFromTime(to)
	if toD.before(fromD) {
		return nil
	}
	return c.holidaysInRange(fromD, toD)
}

// Holidays returns all holidays (built-in + custom, minus removed), sorted by date.
// If a built-in and a custom holiday exist on the same date, only the custom
// holiday is returned.
func (c *Calendar) Holidays() []Holiday {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Holiday
	for d, name := range builtinHolidays {
		if c.removed[d] {
			continue
		}
		if _, ok := c.custom[d]; ok {
			continue
		}
		result = append(result, Holiday{Date: d.toTime(), Name: name})
	}
	for d, name := range c.custom {
		result = append(result, Holiday{Date: d.toTime(), Name: name})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result
}

// holidaysInRange collects holidays within the given date range (inclusive).
func (c *Calendar) holidaysInRange(from, to date) []Holiday {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Holiday
	for d, name := range builtinHolidays {
		if c.removed[d] {
			continue
		}
		if _, ok := c.custom[d]; ok {
			continue
		}
		if d.inRange(from, to) {
			result = append(result, Holiday{Date: d.toTime(), Name: name})
		}
	}
	for d, name := range c.custom {
		if d.inRange(from, to) {
			result = append(result, Holiday{Date: d.toTime(), Name: name})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result
}

// AddCustomHoliday registers a custom holiday on the given date.
// If a custom holiday already exists on that date, it is overwritten.
// If a built-in holiday exists on the same date, this custom holiday takes
// precedence in lookups and list APIs.
func (c *Calendar) AddCustomHoliday(t time.Time, name string) {
	d := dateFromTime(t)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.custom[d] = name
}

// RemoveCustomHoliday removes a previously added custom holiday.
// Has no effect if no custom holiday exists on that date.
func (c *Calendar) RemoveCustomHoliday(t time.Time) {
	d := dateFromTime(t)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.custom, d)
}

// RemoveHoliday suppresses a built-in holiday so it no longer appears in queries.
// Has no effect on custom holidays. Use [Calendar.RestoreHoliday] to undo.
func (c *Calendar) RemoveHoliday(t time.Time) {
	d := dateFromTime(t)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removed[d] = true
}

// RestoreHoliday restores a previously removed built-in holiday.
func (c *Calendar) RestoreHoliday(t time.Time) {
	d := dateFromTime(t)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.removed, d)
}

// --- Package-level convenience functions ---

// IsHoliday reports whether the given date is a holiday.
func IsHoliday(t time.Time) bool { return defaultCal.IsHoliday(t) }

// HolidayName returns the holiday name for the given date, or "".
func HolidayName(t time.Time) string { return defaultCal.HolidayName(t) }

// HolidaysInYear returns all holidays in the given year, sorted by date.
func HolidaysInYear(year int) []Holiday { return defaultCal.HolidaysInYear(year) }

// HolidaysInMonth returns all holidays in the given year and month, sorted by date.
func HolidaysInMonth(year int, month time.Month) []Holiday {
	return defaultCal.HolidaysInMonth(year, month)
}

// HolidaysBetween returns all holidays in the range [from, to] inclusive.
func HolidaysBetween(from, to time.Time) []Holiday {
	return defaultCal.HolidaysBetween(from, to)
}

// Holidays returns all holidays sorted by date.
func Holidays() []Holiday { return defaultCal.Holidays() }

// AddCustomHoliday registers a custom holiday on the default calendar.
func AddCustomHoliday(t time.Time, name string) { defaultCal.AddCustomHoliday(t, name) }

// RemoveCustomHoliday removes a custom holiday from the default calendar.
func RemoveCustomHoliday(t time.Time) { defaultCal.RemoveCustomHoliday(t) }

// RemoveHoliday suppresses a built-in holiday on the default calendar.
func RemoveHoliday(t time.Time) { defaultCal.RemoveHoliday(t) }

// RestoreHoliday restores a suppressed built-in holiday on the default calendar.
func RestoreHoliday(t time.Time) { defaultCal.RestoreHoliday(t) }
