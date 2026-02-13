package jpholiday

import (
	"testing"
	"time"
)

func BenchmarkIsHoliday_Hit(b *testing.B) {
	t := d(2024, time.January, 1)
	for b.Loop() {
		IsHoliday(t)
	}
}

func BenchmarkIsHoliday_Miss(b *testing.B) {
	t := d(2024, time.June, 10)
	for b.Loop() {
		IsHoliday(t)
	}
}

func BenchmarkHolidayName(b *testing.B) {
	t := d(2024, time.January, 1)
	for b.Loop() {
		HolidayName(t)
	}
}

func BenchmarkHolidaysInYear(b *testing.B) {
	for b.Loop() {
		HolidaysInYear(2024)
	}
}

func BenchmarkHolidaysInMonth(b *testing.B) {
	for b.Loop() {
		HolidaysInMonth(2024, time.May)
	}
}

func BenchmarkHolidaysBetween(b *testing.B) {
	from := d(2024, time.April, 28)
	to := d(2024, time.May, 7)
	for b.Loop() {
		HolidaysBetween(from, to)
	}
}

func BenchmarkNextHoliday(b *testing.B) {
	t := d(2024, time.June, 1)
	for b.Loop() {
		NextHoliday(t)
	}
}

func BenchmarkPreviousHoliday(b *testing.B) {
	t := d(2024, time.June, 1)
	for b.Loop() {
		PreviousHoliday(t)
	}
}

func BenchmarkIsBusinessDay(b *testing.B) {
	t := d(2024, time.June, 10)
	for b.Loop() {
		IsBusinessDay(t)
	}
}

func BenchmarkNextBusinessDay(b *testing.B) {
	t := d(2024, time.May, 3) // Golden Week start
	for b.Loop() {
		NextBusinessDay(t)
	}
}

func BenchmarkBusinessDaysBetween_Month(b *testing.B) {
	from := d(2024, time.June, 1)
	to := d(2024, time.June, 30)
	for b.Loop() {
		BusinessDaysBetween(from, to)
	}
}

func BenchmarkBusinessDaysBetween_Year(b *testing.B) {
	from := d(2024, time.January, 1)
	to := d(2024, time.December, 31)
	for b.Loop() {
		BusinessDaysBetween(from, to)
	}
}
