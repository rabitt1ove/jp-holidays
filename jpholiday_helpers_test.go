package jpholiday

import (
	"testing"
	"time"
)

func TestIsBusinessDay(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"Monday non-holiday", d(2024, time.June, 10), true},
		{"Tuesday non-holiday", d(2024, time.June, 11), true},
		{"Friday non-holiday", d(2024, time.June, 14), true},
		{"Saturday", d(2024, time.June, 8), false},
		{"Sunday", d(2024, time.June, 9), false},
		{"New Years Day (Monday)", d(2024, time.January, 1), false},
		{"Substitute holiday", d(2024, time.May, 6), false},
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
			// 2024-01-05 (Fri) 20:00 UTC = 2024-01-06 (Sat) 05:00 JST → not business day
			"UTC Friday evening — Saturday in JST",
			time.Date(2024, time.January, 5, 20, 0, 0, 0, time.UTC),
			false,
		},
		{
			// 2024-01-05 (Fri) 14:59 UTC = 2024-01-05 (Fri) 23:59 JST → business day
			"UTC Friday afternoon — still Friday in JST",
			time.Date(2024, time.January, 5, 14, 59, 0, 0, time.UTC),
			true,
		},
		{
			// 2024-01-07 (Sun) 15:00 UTC = 2024-01-08 (Mon) 00:00 JST → 成人の日 → not business day
			"UTC Sunday 15:00 — Monday holiday in JST",
			time.Date(2024, time.January, 7, 15, 0, 0, 0, time.UTC),
			false,
		},
		{
			// 2024-01-07 (Sun) 14:59 UTC = 2024-01-07 (Sun) 23:59 JST → weekend
			"UTC Sunday 14:59 — still Sunday in JST",
			time.Date(2024, time.January, 7, 14, 59, 0, 0, time.UTC),
			false,
		},
		{
			// 2024-01-08 (Mon 成人の日) 15:00 UTC = 2024-01-09 (Tue) 00:00 JST → business day
			"UTC Monday holiday 15:00 — Tuesday in JST",
			time.Date(2024, time.January, 8, 15, 0, 0, 0, time.UTC),
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
	day := d(2024, time.June, 10) // Monday
	if !cal.IsBusinessDay(day) {
		t.Fatal("should be a business day by default")
	}

	cal.AddCustomHoliday(day, "会社記念日")
	if cal.IsBusinessDay(day) {
		t.Fatal("should not be a business day with custom holiday")
	}
}

func TestNextHoliday(t *testing.T) {
	h, ok := NextHoliday(d(2024, time.January, 1))
	if !ok {
		t.Fatal("expected a next holiday")
	}
	// Next holiday after 2024-01-01 is 2024-01-08 (成人の日).
	if h.Date != d(2024, time.January, 8) {
		t.Errorf("NextHoliday after 2024-01-01 = %s, want 2024-01-08",
			h.Date.Format("2006-01-02"))
	}
	if h.Name != "成人の日" {
		t.Errorf("NextHoliday name = %q, want 成人の日", h.Name)
	}
}

func TestNextHoliday_EndOfDataset(t *testing.T) {
	_, ok := NextHoliday(d(2028, time.January, 1))
	if ok {
		t.Error("should return false after end of dataset")
	}
}

func TestPreviousHoliday(t *testing.T) {
	h, ok := PreviousHoliday(d(2024, time.January, 8))
	if !ok {
		t.Fatal("expected a previous holiday")
	}
	if h.Date != d(2024, time.January, 1) {
		t.Errorf("PreviousHoliday before 2024-01-08 = %s, want 2024-01-01",
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
	tests := []struct {
		name string
		date time.Time
		want time.Time
	}{
		{"Already business day (Friday)", d(2024, time.June, 7), d(2024, time.June, 7)},
		{"Saturday -> Monday", d(2024, time.June, 8), d(2024, time.June, 10)},
		{"Sunday -> Monday", d(2024, time.June, 9), d(2024, time.June, 10)},
		{"Holiday -> next weekday", d(2024, time.January, 1), d(2024, time.January, 2)},
		// 2024-05-03 Fri, 05-04 Sat, 05-05 Sun, 05-06 Mon (sub holiday) -> 05-07 Tue
		{"GW Friday -> Tuesday", d(2024, time.May, 3), d(2024, time.May, 7)},
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

func TestPreviousBusinessDay(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want time.Time
	}{
		{"Already business day (Friday)", d(2024, time.June, 7), d(2024, time.June, 7)},
		{"Saturday -> Friday", d(2024, time.June, 8), d(2024, time.June, 7)},
		{"Sunday -> Friday", d(2024, time.June, 9), d(2024, time.June, 7)},
		{"Monday holiday -> previous Friday", d(2024, time.January, 8), d(2024, time.January, 5)},
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
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{"Mon-Fri no holidays", d(2024, time.June, 10), d(2024, time.June, 14), 5},
		{"Full week with weekend", d(2024, time.June, 10), d(2024, time.June, 16), 5},
		{"Same day business day", d(2024, time.June, 10), d(2024, time.June, 10), 1},
		{"Same day weekend", d(2024, time.June, 8), d(2024, time.June, 8), 0},
		{"Reversed range", d(2024, time.June, 14), d(2024, time.June, 10), 0},
		// GW 2024: 04/29(Mon holiday), 04/30(Tue), 05/01(Wed), 05/02(Thu), 05/03(Fri holiday),
		// 05/04(Sat), 05/05(Sun), 05/06(Mon holiday)
		{"Golden Week", d(2024, time.April, 29), d(2024, time.May, 6), 3},
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
