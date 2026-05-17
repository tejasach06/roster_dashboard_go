package app

import "testing"

func TestMonthBoundsLeapYear(t *testing.T) {
	start, end, last, err := MonthBounds("2024-02")
	if err != nil {
		t.Fatal(err)
	}
	if start != "2024-02-01" || end != "2024-02-29" || last != 29 {
		t.Fatalf("unexpected bounds: %s %s %d", start, end, last)
	}
}

func TestValidation(t *testing.T) {
	if !ValidMonth("2026-05") || ValidMonth("2026-13") {
		t.Fatal("month validation mismatch")
	}
	if !ValidDate("2026-05-17") || ValidDate("2026-02-31") {
		t.Fatal("date validation mismatch")
	}
	if !ValidShift("MS") || ValidShift("BAD") {
		t.Fatal("shift validation mismatch")
	}
}
