package app

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var monthRE = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)
var dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func CurrentMonth() string {
	return time.Now().Format("2006-01")
}

func ValidMonth(month string) bool {
	return monthRE.MatchString(month)
}

func ValidDate(date string) bool {
	if !dateRE.MatchString(date) {
		return false
	}
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func ParseMonth(month string) (int, time.Month, error) {
	if !ValidMonth(month) {
		return 0, 0, fmt.Errorf("month must be YYYY-MM")
	}
	year, _ := strconv.Atoi(month[:4])
	mo, _ := strconv.Atoi(month[5:7])
	return year, time.Month(mo), nil
}

func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func MonthBounds(month string) (string, string, int, error) {
	year, mo, err := ParseMonth(month)
	if err != nil {
		return "", "", 0, err
	}
	last := DaysInMonth(year, mo)
	return month + "-01", fmt.Sprintf("%s-%02d", month, last), last, nil
}

func ShiftMonth(month string, delta int) (string, error) {
	year, mo, err := ParseMonth(month)
	if err != nil {
		return "", err
	}
	return time.Date(year, mo, 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0).Format("2006-01"), nil
}

func BuildDayHeaders(month string) ([]DayHeader, error) {
	year, mo, err := ParseMonth(month)
	if err != nil {
		return nil, err
	}
	dows := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	last := DaysInMonth(year, mo)
	now := time.Now()
	headers := make([]DayHeader, 0, last)
	for day := 1; day <= last; day++ {
		t := time.Date(year, mo, day, 0, 0, 0, 0, time.Local)
		wd := t.Weekday()
		headers = append(headers, DayHeader{
			Num:       day,
			DOW:       dows[int(wd)],
			IsWeekend: wd == time.Saturday || wd == time.Sunday,
			IsToday:   now.Year() == year && now.Month() == mo && now.Day() == day,
			Date:      fmt.Sprintf("%s-%02d", month, day),
		})
	}
	return headers, nil
}
