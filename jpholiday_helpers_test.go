package jpholiday

import (
	"testing"
	"time"
)

func TestIsBusinessDay(t *testing.T) {
	// 2026: Jan 1 = Thu, Jun 1 = Mon
	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"Wednesday non-holiday", d(2026, time.June, 10), true},
		{"Thursday non-holiday", d(2026, time.June, 11), true},
		{"Friday non-holiday", d(2026, time.June, 12), true},
		{"Saturday", d(2026, time.June, 6), false},
		{"Sunday", d(2026, time.June, 7), false},
		{"New Years Day (Thursday)", d(2026, time.January, 1), false},
		{"Substitute holiday", d(2026, time.May, 6), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBusinessDay(tt.date); got != tt.want {
				t.Errorf("IsBusinessDay(%s) = %v, want %v",
					tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestIsBusinessDay_JSTNormalization(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{
			// 2026-01-02 (Fri) 20:00 UTC = 2026-01-03 (Sat) 05:00 JST → not business day
			"UTC Friday evening — Saturday in JST",
			time.Date(2026, time.January, 2, 20, 0, 0, 0, time.UTC),
			false,
		},
		{
			// 2026-01-02 (Fri) 14:59 UTC = 2026-01-02 (Fri) 23:59 JST → business day
			"UTC Friday afternoon — still Friday in JST",
			time.Date(2026, time.January, 2, 14, 59, 0, 0, time.UTC),
			true,
		},
		{
			// 2026-01-11 (Sun) 15:00 UTC = 2026-01-12 (Mon 成人の日) 00:00 JST → not business day
			"UTC Sunday 15:00 — Monday holiday in JST",
			time.Date(2026, time.January, 11, 15, 0, 0, 0, time.UTC),
			false,
		},
		{
			// 2026-01-11 (Sun) 14:59 UTC = 2026-01-11 (Sun) 23:59 JST → weekend
			"UTC Sunday 14:59 — still Sunday in JST",
			time.Date(2026, time.January, 11, 14, 59, 0, 0, time.UTC),
			false,
		},
		{
			// 2026-01-12 (Mon 成人の日) 15:00 UTC = 2026-01-13 (Tue) 00:00 JST → business day
			"UTC Monday holiday 15:00 — Tuesday in JST",
			time.Date(2026, time.January, 12, 15, 0, 0, 0, time.UTC),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBusinessDay(tt.time); got != tt.want {
				jst := time.FixedZone("JST", 9*60*60)
				t.Errorf("IsBusinessDay(%v) = %v, want %v (JST: %v %v)",
					tt.time.Format(time.RFC3339),
					got, tt.want,
					tt.time.In(jst).Format("2006-01-02 15:04"),
					tt.time.In(jst).Weekday())
			}
		})
	}
}

func TestIsBusinessDay_CustomHoliday(t *testing.T) {
	cal := New()
	day := d(2026, time.June, 10) // Wednesday
	if !cal.IsBusinessDay(day) {
		t.Fatal("should be a business day by default")
	}

	cal.AddCustomHoliday(day, "会社記念日")
	if cal.IsBusinessDay(day) {
		t.Fatal("should not be a business day with custom holiday")
	}
}

func TestNextHoliday(t *testing.T) {
	h, ok := NextHoliday(d(2026, time.January, 1))
	if !ok {
		t.Fatal("expected a next holiday")
	}
	// Next holiday after 2026-01-01 is 2026-01-12 (成人の日).
	if h.Date != d(2026, time.January, 12) {
		t.Errorf("NextHoliday after 2026-01-01 = %s, want 2026-01-12",
			h.Date.Format("2006-01-02"))
	}
	if h.Name != "成人の日" {
		t.Errorf("NextHoliday name = %q, want 成人の日", h.Name)
	}
}

func TestNextHoliday_EndOfDataset(t *testing.T) {
	_, ok := NextHoliday(d(2100, time.January, 1))
	if ok {
		t.Error("should return false after end of dataset")
	}
}

func TestPreviousHoliday(t *testing.T) {
	h, ok := PreviousHoliday(d(2026, time.January, 12))
	if !ok {
		t.Fatal("expected a previous holiday")
	}
	if h.Date != d(2026, time.January, 1) {
		t.Errorf("PreviousHoliday before 2026-01-12 = %s, want 2026-01-01",
			h.Date.Format("2006-01-02"))
	}
}

func TestPreviousHoliday_StartOfDataset(t *testing.T) {
	_, ok := PreviousHoliday(d(1950, time.January, 1))
	if ok {
		t.Error("should return false before start of dataset")
	}
}

func TestNextBusinessDay(t *testing.T) {
	// 2026: Jan 1 = Thu, Jun 1 = Mon
	tests := []struct {
		name string
		date time.Time
		want time.Time
	}{
		{"Already business day (Friday)", d(2026, time.June, 5), d(2026, time.June, 5)},
		{"Saturday -> Monday", d(2026, time.June, 6), d(2026, time.June, 8)},
		{"Sunday -> Monday", d(2026, time.June, 7), d(2026, time.June, 8)},
		{"Holiday -> next weekday", d(2026, time.January, 1), d(2026, time.January, 2)},
		// 2026-05-03 Sun, 05-04 Mon (holiday), 05-05 Tue (holiday), 05-06 Wed (holiday) -> 05-07 Thu
		{"GW Sunday -> Thursday", d(2026, time.May, 3), d(2026, time.May, 7)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextBusinessDay(tt.date)
			if got != tt.want {
				t.Errorf("NextBusinessDay(%s) = %s, want %s",
					tt.date.Format("2006-01-02"),
					got.Format("2006-01-02"),
					tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestNextBusinessDay_ZeroOnExhaustion(t *testing.T) {
	cal := New()
	// Add custom holidays for 366 consecutive days to exhaust the loop.
	start := d(2026, time.January, 1)
	for i := 0; i < 366; i++ {
		day := start.AddDate(0, 0, i)
		cal.AddCustomHoliday(day, "blocked")
	}
	got := cal.NextBusinessDay(start)
	if !got.IsZero() {
		t.Errorf("expected zero time on exhaustion, got %s", got.Format("2006-01-02"))
	}
}

func TestPreviousBusinessDay_ZeroOnExhaustion(t *testing.T) {
	cal := New()
	start := d(2026, time.December, 31)
	for i := 0; i < 366; i++ {
		day := start.AddDate(0, 0, -i)
		cal.AddCustomHoliday(day, "blocked")
	}
	got := cal.PreviousBusinessDay(start)
	if !got.IsZero() {
		t.Errorf("expected zero time on exhaustion, got %s", got.Format("2006-01-02"))
	}
}

func TestPreviousBusinessDay(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want time.Time
	}{
		{"Already business day (Friday)", d(2026, time.June, 5), d(2026, time.June, 5)},
		{"Saturday -> Friday", d(2026, time.June, 6), d(2026, time.June, 5)},
		{"Sunday -> Friday", d(2026, time.June, 7), d(2026, time.June, 5)},
		{"Monday holiday -> previous Friday", d(2026, time.January, 12), d(2026, time.January, 9)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreviousBusinessDay(tt.date)
			if got != tt.want {
				t.Errorf("PreviousBusinessDay(%s) = %s, want %s",
					tt.date.Format("2006-01-02"),
					got.Format("2006-01-02"),
					tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestBusinessDaysBetween(t *testing.T) {
	// 2026: Jun 1 = Mon, Jun 8 = Mon
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{"Mon-Fri no holidays", d(2026, time.June, 8), d(2026, time.June, 12), 5},
		{"Full week with weekend", d(2026, time.June, 8), d(2026, time.June, 14), 5},
		{"Same day business day", d(2026, time.June, 8), d(2026, time.June, 8), 1},
		{"Same day weekend", d(2026, time.June, 6), d(2026, time.June, 6), 0},
		{"Reversed range", d(2026, time.June, 12), d(2026, time.June, 8), 0},
		// GW 2026: 04/29(Wed holiday), 04/30(Thu), 05/01(Fri),
		// 05/02(Sat), 05/03(Sun holiday), 05/04(Mon holiday), 05/05(Tue holiday), 05/06(Wed holiday)
		{"Golden Week", d(2026, time.April, 29), d(2026, time.May, 6), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BusinessDaysBetween(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("BusinessDaysBetween(%s, %s) = %d, want %d",
					tt.from.Format("2006-01-02"),
					tt.to.Format("2006-01-02"),
					got, tt.want)
			}
		})
	}
}
