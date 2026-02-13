package jpholiday

import "time"

// IsBusinessDay reports whether the given date is a business day
// (neither a weekend nor a holiday). The date is interpreted in JST.
func (c *Calendar) IsBusinessDay(t time.Time) bool {
	wd := t.In(jstZone).Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !c.IsHoliday(t)
}

// NextHoliday returns the next holiday strictly after the given date.
// Returns false if no future holiday exists in the dataset.
func (c *Calendar) NextHoliday(t time.Time) (Holiday, bool) {
	d := dateFromTime(t)
	var best date
	var bestName string
	found := false

	c.mu.RLock()
	defer c.mu.RUnlock()

	for hd, name := range builtinHolidays {
		if c.removed[hd] {
			continue
		}
		if hd.after(d) && (!found || hd.before(best)) {
			best = hd
			bestName = name
			found = true
		}
	}
	for hd, name := range c.custom {
		if hd.after(d) && (!found || hd.before(best)) {
			best = hd
			bestName = name
			found = true
		}
	}

	if !found {
		return Holiday{}, false
	}
	return Holiday{Date: best.toTime(), Name: bestName}, true
}

// PreviousHoliday returns the most recent holiday strictly before the given date.
// Returns false if no past holiday exists in the dataset.
func (c *Calendar) PreviousHoliday(t time.Time) (Holiday, bool) {
	d := dateFromTime(t)
	var best date
	var bestName string
	found := false

	c.mu.RLock()
	defer c.mu.RUnlock()

	for hd, name := range builtinHolidays {
		if c.removed[hd] {
			continue
		}
		if hd.before(d) && (!found || hd.after(best)) {
			best = hd
			bestName = name
			found = true
		}
	}
	for hd, name := range c.custom {
		if hd.before(d) && (!found || hd.after(best)) {
			best = hd
			bestName = name
			found = true
		}
	}

	if !found {
		return Holiday{}, false
	}
	return Holiday{Date: best.toTime(), Name: bestName}, true
}

// NextBusinessDay returns the next business day on or after the given date.
// If t itself is a business day, it returns t (normalized to midnight UTC).
// Returns the zero time if no business day is found within 366 days.
func (c *Calendar) NextBusinessDay(t time.Time) time.Time {
	d := dateFromTime(t)
	cur := d.toTime()
	for i := 0; i < 366; i++ {
		if c.IsBusinessDay(cur) {
			return cur
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return time.Time{}
}

// PreviousBusinessDay returns the most recent business day on or before the given date.
// If t itself is a business day, it returns t (normalized to midnight UTC).
// Returns the zero time if no business day is found within 366 days.
func (c *Calendar) PreviousBusinessDay(t time.Time) time.Time {
	d := dateFromTime(t)
	cur := d.toTime()
	for i := 0; i < 366; i++ {
		if c.IsBusinessDay(cur) {
			return cur
		}
		cur = cur.AddDate(0, 0, -1)
	}
	return time.Time{}
}

// BusinessDaysBetween returns the count of business days in the range [from, to] inclusive.
// If from is after to, returns 0.
func (c *Calendar) BusinessDaysBetween(from, to time.Time) int {
	fromD := dateFromTime(from)
	toD := dateFromTime(to)
	if toD.before(fromD) {
		return 0
	}

	count := 0
	cur := fromD.toTime()
	end := toD.toTime()
	for !cur.After(end) {
		if c.IsBusinessDay(cur) {
			count++
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return count
}

// --- Package-level convenience functions ---

// IsBusinessDay reports whether the given date is a business day.
func IsBusinessDay(t time.Time) bool { return defaultCal.IsBusinessDay(t) }

// NextHoliday returns the next holiday strictly after the given date.
func NextHoliday(t time.Time) (Holiday, bool) { return defaultCal.NextHoliday(t) }

// PreviousHoliday returns the most recent holiday strictly before the given date.
func PreviousHoliday(t time.Time) (Holiday, bool) { return defaultCal.PreviousHoliday(t) }

// NextBusinessDay returns the next business day on or after the given date.
func NextBusinessDay(t time.Time) time.Time { return defaultCal.NextBusinessDay(t) }

// PreviousBusinessDay returns the most recent business day on or before the given date.
func PreviousBusinessDay(t time.Time) time.Time { return defaultCal.PreviousBusinessDay(t) }

// BusinessDaysBetween returns the count of business days in the range [from, to].
func BusinessDaysBetween(from, to time.Time) int { return defaultCal.BusinessDaysBetween(from, to) }
