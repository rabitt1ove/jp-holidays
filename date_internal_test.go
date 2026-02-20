package jpholiday

import (
	"testing"
	"time"
)

func d(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestDateBefore_EqualDates(t *testing.T) {
	t.Parallel()

	d1 := date{year: 2026, month: time.January, day: 1}
	if d1.before(d1) {
		t.Error("equal dates: d.before(d) should be false")
	}
	if d1.after(d1) {
		t.Error("equal dates: d.after(d) should be false")
	}
}

func TestDateBefore_SameYearSameMonth(t *testing.T) {
	t.Parallel()

	d1 := date{year: 2026, month: time.January, day: 1}
	d2 := date{year: 2026, month: time.January, day: 15}
	if !d1.before(d2) {
		t.Error("Jan 1 should be before Jan 15")
	}
	if d2.before(d1) {
		t.Error("Jan 15 should not be before Jan 1")
	}
}

func TestDateBefore_SameYearDifferentMonth(t *testing.T) {
	t.Parallel()

	d1 := date{year: 2026, month: time.January, day: 31}
	d2 := date{year: 2026, month: time.February, day: 1}
	if !d1.before(d2) {
		t.Error("Jan 31 should be before Feb 1")
	}
}

func TestDateBefore_DifferentYear(t *testing.T) {
	t.Parallel()

	d1 := date{year: 2025, month: time.December, day: 31}
	d2 := date{year: 2026, month: time.January, day: 1}
	if !d1.before(d2) {
		t.Error("2025-12-31 should be before 2026-01-01")
	}
}

func TestDateInRange_Boundaries(t *testing.T) {
	t.Parallel()

	from := date{year: 2026, month: time.January, day: 1}
	to := date{year: 2026, month: time.January, day: 31}

	if !from.inRange(from, to) {
		t.Error("from date should be in range (inclusive)")
	}
	if !to.inRange(from, to) {
		t.Error("to date should be in range (inclusive)")
	}

	beforeFrom := date{year: 2025, month: time.December, day: 31}
	afterTo := date{year: 2026, month: time.February, day: 1}
	if beforeFrom.inRange(from, to) {
		t.Error("day before from should not be in range")
	}
	if afterTo.inRange(from, to) {
		t.Error("day after to should not be in range")
	}
}
