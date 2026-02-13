package jpholiday_test

import (
	"fmt"
	"time"

	jpholiday "github.com/rabitt1ove/jp-holidays"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func ExampleIsHoliday() {
	t := time.Date(2024, time.January, 1, 0, 0, 0, 0, jst)
	fmt.Println(jpholiday.IsHoliday(t))
	// Output: true
}

func ExampleHolidayName() {
	t := time.Date(2024, time.January, 1, 0, 0, 0, 0, jst)
	fmt.Println(jpholiday.HolidayName(t))
	// Output: 元日
}

func ExampleHolidaysInYear() {
	holidays := jpholiday.HolidaysInYear(2024)
	for _, h := range holidays[:3] {
		fmt.Printf("%s: %s\n", h.Date.Format("2006-01-02"), h.Name)
	}
	// Output:
	// 2024-01-01: 元日
	// 2024-01-08: 成人の日
	// 2024-02-11: 建国記念の日
}

func ExampleHolidaysInMonth() {
	holidays := jpholiday.HolidaysInMonth(2024, time.May)
	for _, h := range holidays {
		fmt.Printf("%s: %s\n", h.Date.Format("01-02"), h.Name)
	}
	// Output:
	// 05-03: 憲法記念日
	// 05-04: みどりの日
	// 05-05: こどもの日
	// 05-06: 休日
}

func ExampleIsBusinessDay() {
	fmt.Println(jpholiday.IsBusinessDay(time.Date(2024, time.June, 10, 0, 0, 0, 0, jst)))   // Monday
	fmt.Println(jpholiday.IsBusinessDay(time.Date(2024, time.June, 8, 0, 0, 0, 0, jst)))    // Saturday
	fmt.Println(jpholiday.IsBusinessDay(time.Date(2024, time.January, 1, 0, 0, 0, 0, jst))) // Holiday
	// Output:
	// true
	// false
	// false
}

func ExampleNew() {
	cal := jpholiday.New()
	cal.AddCustomHoliday(
		time.Date(2024, time.June, 15, 0, 0, 0, 0, jst),
		"会社記念日",
	)
	fmt.Println(cal.IsHoliday(time.Date(2024, time.June, 15, 0, 0, 0, 0, jst)))
	fmt.Println(cal.HolidayName(time.Date(2024, time.June, 15, 0, 0, 0, 0, jst)))
	// Output:
	// true
	// 会社記念日
}

func ExampleNextBusinessDay() {
	// 土曜日 → 次の月曜日
	sat := time.Date(2024, time.June, 8, 0, 0, 0, 0, jst)
	next := jpholiday.NextBusinessDay(sat)
	fmt.Println(next.Format("2006-01-02"))
	// Output: 2024-06-10
}
