package jpholiday

import (
	"sync"
	"testing"
	"time"
)

// d is a test helper to construct dates.
func d(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestIsHoliday(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"New Years Day", d(2026, time.January, 1), true},
		{"Coming of Age Day", d(2026, time.January, 12), true},
		{"National Foundation Day", d(2026, time.February, 11), true},
		{"Emperors Birthday", d(2026, time.February, 23), true},
		{"Vernal Equinox", d(2026, time.March, 20), true},
		{"Showa Day", d(2026, time.April, 29), true},
		{"Constitution Memorial Day", d(2026, time.May, 3), true},
		{"Greenery Day", d(2026, time.May, 4), true},
		{"Childrens Day", d(2026, time.May, 5), true},
		{"Substitute holiday 05-06", d(2026, time.May, 6), true},
		{"Marine Day", d(2026, time.July, 20), true},
		{"Mountain Day", d(2026, time.August, 11), true},
		{"Respect for Aged Day", d(2026, time.September, 21), true},
		{"Bridge holiday 09-22", d(2026, time.September, 22), true},
		{"Autumnal Equinox", d(2026, time.September, 23), true},
		{"Sports Day", d(2026, time.October, 12), true},
		{"Culture Day", d(2026, time.November, 3), true},
		{"Labor Thanksgiving Day", d(2026, time.November, 23), true},

		{"Regular weekday", d(2026, time.June, 10), false},
		{"Saturday non-holiday", d(2026, time.June, 6), false},
		{"Sunday non-holiday", d(2026, time.June, 7), false},
		{"Day before New Years", d(2026, time.December, 31), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsHoliday(tt.date); got != tt.want {
				t.Errorf("IsHoliday(%v) = %v, want %v", tt.date.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

func TestIsHoliday_TimeOfDayIgnored(t *testing.T) {
	t.Parallel()

	jst := time.FixedZone("JST", 9*60*60)
	late := time.Date(2026, time.January, 1, 23, 59, 59, 0, jst)
	if !IsHoliday(late) {
		t.Error("IsHoliday should ignore time-of-day")
	}
}

func TestIsHoliday_JSTNormalization(t *testing.T) {
	t.Parallel()

	jst := time.FixedZone("JST", 9*60*60)

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		// 2026-01-01 is 元日 (New Year's Day).
		{
			"JST noon on holiday",
			time.Date(2026, time.January, 1, 12, 0, 0, 0, jst),
			true,
		},
		{
			"JST 23:59 on holiday — still Jan 1 in JST",
			time.Date(2026, time.January, 1, 23, 59, 0, 0, jst),
			true,
		},
		{
			// 2025-12-31 20:00 UTC = 2026-01-01 05:00 JST → 元日
			"UTC Dec 31 evening — already Jan 1 in JST",
			time.Date(2025, time.December, 31, 20, 0, 0, 0, time.UTC),
			true,
		},
		{
			// 2026-01-01 14:59 UTC = 2026-01-01 23:59 JST → still 元日
			"UTC Jan 1 14:59 — still Jan 1 in JST",
			time.Date(2026, time.January, 1, 14, 59, 0, 0, time.UTC),
			true,
		},
		{
			// 2026-01-01 15:00 UTC = 2026-01-02 00:00 JST → not a holiday
			"UTC Jan 1 15:00 — already Jan 2 in JST",
			time.Date(2026, time.January, 1, 15, 0, 0, 0, time.UTC),
			false,
		},
		{
			// 2025-12-31 14:59 UTC = 2025-12-31 23:59 JST → not a holiday
			"UTC Dec 31 14:59 — still Dec 31 in JST",
			time.Date(2025, time.December, 31, 14, 59, 0, 0, time.UTC),
			false,
		},
		{
			// US Pacific (UTC-8): 2025-12-31 11:00 PST = 2025-12-31 19:00 UTC = 2026-01-01 04:00 JST → 元日
			"US Pacific Dec 31 morning — already Jan 1 in JST",
			time.Date(2025, time.December, 31, 11, 0, 0, 0, time.FixedZone("PST", -8*60*60)),
			true,
		},
		{
			// India (UTC+5:30): 2026-01-01 03:29 IST = 2025-12-31 21:59 UTC = 2026-01-01 06:59 JST → 元日
			"India Jan 1 early morning — already Jan 1 in JST",
			time.Date(2026, time.January, 1, 3, 29, 0, 0, time.FixedZone("IST", 5*60*60+30*60)),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsHoliday(tt.time); got != tt.want {
				t.Errorf("IsHoliday(%v) = %v, want %v (JST: %v)",
					tt.time.Format(time.RFC3339),
					got, tt.want,
					tt.time.In(time.FixedZone("JST", 9*60*60)).Format("2006-01-02 15:04"))
			}
		})
	}
}

func TestIsHoliday_BeforeDataset(t *testing.T) {
	t.Parallel()
	if IsHoliday(d(1950, time.January, 1)) {
		t.Error("dates before 1955 should not be holidays (no data)")
	}
}

func TestIsHoliday_AfterDataset(t *testing.T) {
	t.Parallel()

	if IsHoliday(d(2100, time.January, 1)) {
		t.Error("dates after dataset should not be holidays")
	}
}

func TestHolidayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date time.Time
		want string
	}{
		{d(2026, time.January, 1), "元日"},
		{d(2026, time.January, 12), "成人の日"},
		{d(2026, time.May, 3), "憲法記念日"},
		{d(2026, time.May, 4), "みどりの日"},
		{d(2026, time.May, 5), "こどもの日"},
		{d(2026, time.November, 3), "文化の日"},
		{d(2026, time.November, 23), "勤労感謝の日"},
		{d(2026, time.June, 10), ""},
	}
	for _, tt := range tests {
		name := tt.date.Format("2006-01-02")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := HolidayName(tt.date); got != tt.want {
				t.Errorf("HolidayName(%s) = %q, want %q", name, got, tt.want)
			}
		})
	}
}

func TestHolidaysInYear(t *testing.T) {
	t.Parallel()

	holidays := HolidaysInYear(2026)
	if len(holidays) == 0 {
		t.Fatal("expected holidays in 2026")
	}

	// First holiday should be New Year's Day.
	if holidays[0].Name != "元日" {
		t.Errorf("first holiday = %q, want 元日", holidays[0].Name)
	}

	// Verify sorted order.
	for i := 1; i < len(holidays); i++ {
		if !holidays[i].Date.After(holidays[i-1].Date) {
			t.Errorf("holidays not sorted: [%d]%v >= [%d]%v",
				i-1, holidays[i-1].Date.Format("2006-01-02"),
				i, holidays[i].Date.Format("2006-01-02"))
		}
	}
}

func TestHolidaysInYear_Empty(t *testing.T) {
	t.Parallel()

	holidays := HolidaysInYear(1900)
	if len(holidays) != 0 {
		t.Errorf("expected 0 holidays for 1900, got %d", len(holidays))
	}
}

func TestHolidaysInMonth(t *testing.T) {
	t.Parallel()

	holidays := HolidaysInMonth(2026, time.May)
	// May 2026: 5/3 憲法記念日, 5/4 みどりの日, 5/5 こどもの日, 5/6 休日
	if len(holidays) != 4 {
		t.Errorf("expected 4 holidays in May 2026, got %d", len(holidays))
	}

	for _, h := range holidays {
		if h.Date.Month() != time.May {
			t.Errorf("unexpected month: %v", h.Date)
		}
	}
}

func TestHolidaysInMonth_Empty(t *testing.T) {
	t.Parallel()

	holidays := HolidaysInMonth(2026, time.June)
	if len(holidays) != 0 {
		t.Errorf("expected 0 holidays in June 2026, got %d", len(holidays))
	}
}

func TestHolidaysBetween(t *testing.T) {
	t.Parallel()

	// Golden Week 2026: 4/29 昭和の日, 5/3 憲法記念日, 5/4 みどりの日, 5/5 こどもの日, 5/6 休日
	holidays := HolidaysBetween(d(2026, time.April, 28), d(2026, time.May, 7))
	if len(holidays) != 5 {
		t.Errorf("expected 5 holidays in Golden Week 2026, got %d", len(holidays))
	}

	// Verify sorted order.
	for i := 1; i < len(holidays); i++ {
		if !holidays[i].Date.After(holidays[i-1].Date) {
			t.Errorf("not sorted at index %d", i)
		}
	}
}

func TestHolidaysBetween_Reversed(t *testing.T) {
	t.Parallel()

	holidays := HolidaysBetween(d(2026, time.December, 31), d(2026, time.January, 1))
	if len(holidays) != 0 {
		t.Errorf("expected 0 holidays for reversed range, got %d", len(holidays))
	}
}

func TestHolidaysBetween_SameDay_Holiday(t *testing.T) {
	t.Parallel()

	holidays := HolidaysBetween(d(2026, time.January, 1), d(2026, time.January, 1))
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday, got %d", len(holidays))
	}
}

func TestHolidaysBetween_SameDay_NonHoliday(t *testing.T) {
	t.Parallel()

	holidays := HolidaysBetween(d(2026, time.June, 10), d(2026, time.June, 10))
	if len(holidays) != 0 {
		t.Errorf("expected 0 holidays, got %d", len(holidays))
	}
}

func TestHolidays(t *testing.T) {
	t.Parallel()

	all := Holidays()
	if len(all) < 1000 {
		t.Errorf("expected at least 1000 holidays, got %d", len(all))
	}

	// Verify sorted.
	for i := 1; i < len(all); i++ {
		if !all[i].Date.After(all[i-1].Date) {
			t.Errorf("not sorted at index %d: %v >= %v",
				i, all[i-1].Date.Format("2006-01-02"), all[i].Date.Format("2006-01-02"))
		}
	}
}

// --- Custom holiday tests ---

func TestCustomHoliday_AddAndRemove(t *testing.T) {
	t.Parallel()

	cal := New()
	day := d(2026, time.June, 15)

	if cal.IsHoliday(day) {
		t.Fatal("June 15 should not be a holiday by default")
	}

	cal.AddCustomHoliday(day, "会社記念日")
	if !cal.IsHoliday(day) {
		t.Fatal("June 15 should be a holiday after adding")
	}
	if got := cal.HolidayName(day); got != "会社記念日" {
		t.Errorf("HolidayName = %q, want 会社記念日", got)
	}

	cal.RemoveCustomHoliday(day)
	if cal.IsHoliday(day) {
		t.Fatal("June 15 should not be a holiday after removal")
	}
}

func TestCustomHoliday_Overwrite(t *testing.T) {
	t.Parallel()

	cal := New()
	day := d(2026, time.June, 15)

	cal.AddCustomHoliday(day, "記念日A")
	cal.AddCustomHoliday(day, "記念日B")
	if got := cal.HolidayName(day); got != "記念日B" {
		t.Errorf("HolidayName = %q, want 記念日B", got)
	}
}

func TestCustomHoliday_AppearsInRange(t *testing.T) {
	t.Parallel()

	cal := New()
	day := d(2026, time.June, 15)
	cal.AddCustomHoliday(day, "会社記念日")

	holidays := cal.HolidaysInMonth(2026, time.June)
	if len(holidays) != 1 {
		t.Fatalf("expected 1 holiday in June, got %d", len(holidays))
	}
	if holidays[0].Name != "会社記念日" {
		t.Errorf("expected 会社記念日, got %q", holidays[0].Name)
	}
}

func TestCustomHoliday_TakesPrecedence(t *testing.T) {
	t.Parallel()

	cal := New()
	newYears := d(2026, time.January, 1)
	cal.AddCustomHoliday(newYears, "カスタム元日")

	if got := cal.HolidayName(newYears); got != "カスタム元日" {
		t.Errorf("custom should take precedence, got %q", got)
	}
}

func TestCustomHoliday_NoDuplicateInRange(t *testing.T) {
	t.Parallel()

	cal := New()
	newYears := d(2026, time.January, 1)
	cal.AddCustomHoliday(newYears, "カスタム元日")

	holidays := cal.HolidaysBetween(newYears, newYears)
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday (no duplicate), got %d", len(holidays))
	}
	if len(holidays) > 0 && holidays[0].Name != "カスタム元日" {
		t.Errorf("expected custom name, got %q", holidays[0].Name)
	}
}

func TestRemoveBuiltinHoliday(t *testing.T) {
	t.Parallel()

	cal := New()
	newYears := d(2026, time.January, 1)

	if !cal.IsHoliday(newYears) {
		t.Fatal("New Years should be a holiday")
	}

	cal.RemoveHoliday(newYears)
	if cal.IsHoliday(newYears) {
		t.Fatal("New Years should not be a holiday after removal")
	}
	if got := cal.HolidayName(newYears); got != "" {
		t.Errorf("HolidayName should be empty, got %q", got)
	}

	cal.RestoreHoliday(newYears)
	if !cal.IsHoliday(newYears) {
		t.Fatal("New Years should be restored")
	}
}

func TestRemoveBuiltinHoliday_InRange(t *testing.T) {
	t.Parallel()

	cal := New()
	cal.RemoveHoliday(d(2026, time.January, 1))

	holidays := cal.HolidaysInMonth(2026, time.January)
	for _, h := range holidays {
		if h.Name == "元日" {
			t.Error("removed holiday should not appear in range queries")
		}
	}
}

func TestCustomHoliday_DoesNotAffectDefault(t *testing.T) {
	t.Parallel()

	cal := New()
	day := d(2026, time.August, 15)
	cal.AddCustomHoliday(day, "お盆")

	if IsHoliday(day) {
		t.Fatal("package-level should not see cal's custom holiday")
	}
}

func TestRemoveCustomHoliday_NoEffect(t *testing.T) {
	t.Parallel()

	cal := New()
	// Removing a non-existent custom holiday should not panic or error.
	cal.RemoveCustomHoliday(d(2026, time.June, 15))
}

// --- Holidays() with custom/removed overlaps ---

func TestHolidays_CustomOverridesBuiltin(t *testing.T) {
	t.Parallel()

	cal := New()
	newYears := d(2026, time.January, 1)
	cal.AddCustomHoliday(newYears, "カスタム元日")

	all := cal.Holidays()
	count := 0
	for _, h := range all {
		if h.Date.Equal(newYears) {
			count++
			if h.Name != "カスタム元日" {
				t.Errorf("expected custom name, got %q", h.Name)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 entry for Jan 1, got %d", count)
	}
}

func TestHolidays_RemovedExcluded(t *testing.T) {
	t.Parallel()

	cal := New()
	cal.RemoveHoliday(d(2026, time.January, 1))

	for _, h := range cal.Holidays() {
		if h.Date.Equal(d(2026, time.January, 1)) {
			t.Error("removed holiday should not appear in Holidays()")
		}
	}
}

func TestHolidays_CustomOnlyDate(t *testing.T) {
	t.Parallel()

	cal := New()
	custom := d(2026, time.June, 15)
	cal.AddCustomHoliday(custom, "会社記念日")

	found := false
	for _, h := range cal.Holidays() {
		if h.Date.Equal(custom) {
			found = true
			if h.Name != "会社記念日" {
				t.Errorf("expected 会社記念日, got %q", h.Name)
			}
		}
	}
	if !found {
		t.Error("custom-only holiday should appear in Holidays()")
	}
}

// --- Package-level custom/remove convenience functions ---

func TestPackageLevelAddRemoveCustomHoliday(t *testing.T) {
	// NOT parallel: mutates package-level defaultCal.
	day := d(2099, time.July, 1) // Far future, won't conflict
	t.Cleanup(func() { RemoveCustomHoliday(day) })

	AddCustomHoliday(day, "パッケージテスト")
	if !IsHoliday(day) {
		t.Error("AddCustomHoliday should make the date a holiday")
	}
	if got := HolidayName(day); got != "パッケージテスト" {
		t.Errorf("HolidayName = %q, want パッケージテスト", got)
	}

	RemoveCustomHoliday(day)
	if IsHoliday(day) {
		t.Error("RemoveCustomHoliday should remove the custom holiday")
	}
}

func TestPackageLevelRemoveRestoreHoliday(t *testing.T) {
	// NOT parallel: mutates package-level defaultCal.
	newYears := d(2026, time.January, 1)
	t.Cleanup(func() { RestoreHoliday(newYears) })

	RemoveHoliday(newYears)
	if IsHoliday(newYears) {
		t.Error("RemoveHoliday should suppress the built-in holiday")
	}

	RestoreHoliday(newYears)
	if !IsHoliday(newYears) {
		t.Error("RestoreHoliday should bring back the built-in holiday")
	}
}

// --- NextHoliday / PreviousHoliday with custom holidays ---

func TestNextHoliday_CustomHoliday(t *testing.T) {
	t.Parallel()

	cal := New()
	// Add a custom holiday closer than the next built-in holiday.
	// Next built-in after 2026-01-01 is 2026-01-12 (成人の日).
	custom := d(2026, time.January, 5)
	cal.AddCustomHoliday(custom, "カスタム祝日")

	h, ok := cal.NextHoliday(d(2026, time.January, 1))
	if !ok {
		t.Fatal("expected a next holiday")
	}
	if !h.Date.Equal(custom) {
		t.Errorf("NextHoliday = %s, want 2026-01-05", h.Date.Format("2006-01-02"))
	}
	if h.Name != "カスタム祝日" {
		t.Errorf("NextHoliday name = %q, want カスタム祝日", h.Name)
	}
}

func TestNextHoliday_CustomOnly(t *testing.T) {
	t.Parallel()

	cal := New()
	// Remove all built-in holidays by going far into the future where none exist,
	// then add a custom one.
	custom := d(2200, time.June, 15)
	cal.AddCustomHoliday(custom, "未来の祝日")

	h, ok := cal.NextHoliday(d(2200, time.June, 1))
	if !ok {
		t.Fatal("expected a next holiday from custom")
	}
	if !h.Date.Equal(custom) {
		t.Errorf("NextHoliday = %s, want 2200-06-15", h.Date.Format("2006-01-02"))
	}
}

func TestNextHoliday_RemovedBuiltin(t *testing.T) {
	t.Parallel()

	cal := New()
	// Remove 2026-01-12 (成人の日). Next after 2026-01-01 should skip it.
	cal.RemoveHoliday(d(2026, time.January, 12))

	h, ok := cal.NextHoliday(d(2026, time.January, 1))
	if !ok {
		t.Fatal("expected a next holiday")
	}
	if h.Date.Equal(d(2026, time.January, 12)) {
		t.Error("removed holiday should be skipped")
	}
}

func TestPreviousHoliday_CustomHoliday(t *testing.T) {
	t.Parallel()

	cal := New()
	// Add a custom holiday closer than the previous built-in.
	// Previous built-in before 2026-01-12 is 2026-01-01 (元日).
	custom := d(2026, time.January, 10)
	cal.AddCustomHoliday(custom, "カスタム祝日")

	h, ok := cal.PreviousHoliday(d(2026, time.January, 12))
	if !ok {
		t.Fatal("expected a previous holiday")
	}
	if !h.Date.Equal(custom) {
		t.Errorf("PreviousHoliday = %s, want 2026-01-10", h.Date.Format("2006-01-02"))
	}
	if h.Name != "カスタム祝日" {
		t.Errorf("PreviousHoliday name = %q, want カスタム祝日", h.Name)
	}
}

func TestPreviousHoliday_CustomOnly(t *testing.T) {
	t.Parallel()

	cal := New()
	custom := d(1900, time.June, 15)
	cal.AddCustomHoliday(custom, "古い祝日")

	h, ok := cal.PreviousHoliday(d(1900, time.July, 1))
	if !ok {
		t.Fatal("expected a previous holiday from custom")
	}
	if !h.Date.Equal(custom) {
		t.Errorf("PreviousHoliday = %s, want 1900-06-15", h.Date.Format("2006-01-02"))
	}
}

func TestPreviousHoliday_RemovedBuiltin(t *testing.T) {
	t.Parallel()

	cal := New()
	cal.RemoveHoliday(d(2026, time.January, 1))

	h, ok := cal.PreviousHoliday(d(2026, time.January, 12))
	if !ok {
		t.Fatal("expected a previous holiday")
	}
	if h.Date.Equal(d(2026, time.January, 1)) {
		t.Error("removed holiday should be skipped")
	}
}

// --- Concurrency tests ---

func TestConcurrentAccess(t *testing.T) {
	cal := New()
	var wg sync.WaitGroup

	// Concurrent reads.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cal.IsHoliday(d(2026, time.January, 1))
			cal.HolidayName(d(2026, time.May, 3))
			cal.HolidaysInYear(2026)
		}()
	}

	// Concurrent writes.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			day := d(2026, time.June, i%28+1)
			cal.AddCustomHoliday(day, "テスト")
			cal.RemoveCustomHoliday(day)
		}(i)
	}

	wg.Wait()
}

// --- Boundary value tests ---

func TestHolidaysBetween_CrossYearBoundary(t *testing.T) {
	t.Parallel()

	// 2025-11-01 to 2026-01-31 should include holidays from both years.
	// Nov 3 (文化の日), Nov 23 (勤労感謝の日) in 2025; Jan 1 (元日) in 2026.
	holidays := HolidaysBetween(d(2025, time.November, 1), d(2026, time.January, 31))
	if len(holidays) == 0 {
		t.Fatal("expected holidays spanning year boundary")
	}

	hasOldYear := false
	hasNewYear := false
	for _, h := range holidays {
		if h.Date.Year() == 2025 {
			hasOldYear = true
		}
		if h.Date.Year() == 2026 {
			hasNewYear = true
		}
	}
	if !hasOldYear || !hasNewYear {
		t.Error("expected holidays from both 2025 and 2026")
	}

	// Verify sorted across year boundary.
	for i := 1; i < len(holidays); i++ {
		if !holidays[i].Date.After(holidays[i-1].Date) {
			t.Errorf("not sorted at index %d across year boundary", i)
		}
	}
}

func TestHolidaysInMonth_FebruaryLeapYear(t *testing.T) {
	t.Parallel()

	// 2024 is a leap year (Feb has 29 days). Feb 11 = 建国記念の日, Feb 12 = 振替休日, Feb 23 = 天皇誕生日.
	holidays := HolidaysInMonth(2024, time.February)
	if len(holidays) == 0 {
		t.Fatal("expected holidays in February 2024")
	}
	for _, h := range holidays {
		if h.Date.Month() != time.February || h.Date.Year() != 2024 {
			t.Errorf("unexpected date: %v", h.Date)
		}
	}
}

func TestHolidaysInMonth_FebruaryNonLeapYear(t *testing.T) {
	t.Parallel()

	// 2025 is not a leap year (Feb has 28 days).
	holidays := HolidaysInMonth(2025, time.February)
	for _, h := range holidays {
		if h.Date.Day() > 28 {
			t.Errorf("Feb non-leap year should not have day > 28, got %d", h.Date.Day())
		}
	}
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

	// Exactly at boundaries — inclusive.
	if !from.inRange(from, to) {
		t.Error("from date should be in range (inclusive)")
	}
	if !to.inRange(from, to) {
		t.Error("to date should be in range (inclusive)")
	}

	// Just outside boundaries.
	beforeFrom := date{year: 2025, month: time.December, day: 31}
	afterTo := date{year: 2026, month: time.February, day: 1}
	if beforeFrom.inRange(from, to) {
		t.Error("day before from should not be in range")
	}
	if afterTo.inRange(from, to) {
		t.Error("day after to should not be in range")
	}
}
