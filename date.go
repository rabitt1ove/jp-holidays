package jpholiday

import "time"

// jstZone is the Asia/Tokyo timezone (UTC+9) used to normalize all input
// times to the Japanese calendar date before holiday lookups.
var jstZone = time.FixedZone("Asia/Tokyo", 9*60*60)

// date is an internal comparable key for map lookups.
// Users work with time.Time; this type is not exported.
type date struct {
	year  int
	month time.Month
	day   int
}

// dateFromTime converts a time.Time to a date by first normalizing to JST.
// This ensures that a moment in time always maps to the correct Japanese
// calendar date regardless of the input timezone.
func dateFromTime(t time.Time) date {
	jt := t.In(jstZone)
	y, m, d := jt.Date()
	return date{year: y, month: m, day: d}
}

func (d date) toTime() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

func (d date) before(other date) bool {
	if d.year != other.year {
		return d.year < other.year
	}
	if d.month != other.month {
		return d.month < other.month
	}
	return d.day < other.day
}

func (d date) after(other date) bool {
	return other.before(d)
}

func (d date) inRange(from, to date) bool {
	return !d.before(from) && !to.before(d)
}
