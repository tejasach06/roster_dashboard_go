package app

import "time"

var ShiftCodes = []string{"MS", "GS", "AS", "NS", "WO", "EL"}

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TeamID   *int64 `json:"team_id"`
	TeamName string `json:"team_name,omitempty"`
}

func (u User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u User) CanAccessTeam(teamID int64) bool {
	return u.IsAdmin() || (u.TeamID != nil && *u.TeamID == teamID)
}

type Team struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int64  `json:"member_count"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type Employee struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	EmpCode   string `json:"emp_code"`
	JobTitle  string `json:"job_title"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	TeamID    *int64 `json:"team_id"`
	TeamName  string `json:"team_name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type RosterEntry struct {
	ID         int64  `json:"id"`
	ShiftCode  string `json:"shift_code"`
	Date       string `json:"date"`
	Notes      string `json:"notes"`
	EmployeeID int64  `json:"employee_id"`
	Name       string `json:"name"`
	EmpCode    string `json:"emp_code"`
	JobTitle   string `json:"job_title"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
}

type StatRow struct {
	TeamID    int64   `json:"team_id"`
	TeamName  string  `json:"team_name"`
	ShiftCode *string `json:"shift_code"`
	Count     int64   `json:"count"`
}

type DayHeader struct {
	Num       int
	DOW       string
	IsWeekend bool
	IsToday   bool
	Date      string
}

type RosterRow struct {
	Employee Employee
	Days     map[int]RosterEntry
	Totals   map[string]int
}

type ImportResult struct {
	Created  int      `json:"created,omitempty"`
	Imported int      `json:"imported,omitempty"`
	Skipped  int      `json:"skipped,omitempty"`
	Updated  int      `json:"updated,omitempty"`
	Copied   int      `json:"copied,omitempty"`
	Deleted  int64    `json:"deleted,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func ValidShift(code string) bool {
	for _, valid := range ShiftCodes {
		if code == valid {
			return true
		}
	}
	return false
}

func Today() time.Time {
	return time.Now()
}
