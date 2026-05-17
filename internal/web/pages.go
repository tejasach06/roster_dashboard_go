package web

import (
	"fmt"
	"net/http"
	"sort"

	"roster_dashboard_go/internal/app"
)

type PageData struct {
	Title      string
	User       app.User
	IsAdmin    bool
	Error      string
	Month      string
	PrevMonth  string
	NextMonth  string
	Teams      []app.Team
	Users      []app.User
	Employees  []app.Employee
	Stats      []TeamSummary
	Team       app.Team
	Headers    []app.DayHeader
	Rows       []app.RosterRow
	Unassigned []app.Employee
	ShiftCodes []string
}

type TeamSummary struct {
	TeamID      int64
	TeamName    string
	ShiftCounts map[string]int64
	Total       int64
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "login", PageData{Title: "Sign in", Error: r.URL.Query().Get("error")})
}

func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithError(w, r, "/login", "Invalid form")
		return
	}
	user, err := s.store.Authenticate(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		redirectWithError(w, r, "/login", "Invalid credentials")
		return
	}
	if _, err := s.setSession(w, user); err != nil {
		redirectWithError(w, r, "/login", "Unable to create session")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) dashboardPage(w http.ResponseWriter, r *http.Request, user app.User) {
	month := queryMonth(r)
	if !app.ValidMonth(month) {
		month = app.CurrentMonth()
	}
	prev, _ := app.ShiftMonth(month, -1)
	next, _ := app.ShiftMonth(month, 1)
	stats, err := s.store.RosterStats(user, month)
	if err != nil {
		s.render(w, "dashboard", PageData{Title: "Dashboard", User: user, IsAdmin: user.IsAdmin(), Error: err.Error(), Month: month})
		return
	}
	summaries := map[int64]*TeamSummary{}
	for _, row := range stats {
		if _, ok := summaries[row.TeamID]; !ok {
			counts := map[string]int64{}
			for _, code := range app.ShiftCodes {
				counts[code] = 0
			}
			summaries[row.TeamID] = &TeamSummary{TeamID: row.TeamID, TeamName: row.TeamName, ShiftCounts: counts}
		}
		if row.ShiftCode != nil {
			summaries[row.TeamID].ShiftCounts[*row.ShiftCode] += row.Count
			summaries[row.TeamID].Total += row.Count
		}
	}
	out := make([]TeamSummary, 0, len(summaries))
	for _, summary := range summaries {
		out = append(out, *summary)
	}
	s.render(w, "dashboard", PageData{
		Title: "Dashboard", User: user, IsAdmin: user.IsAdmin(), Error: r.URL.Query().Get("error"),
		Month: month, PrevMonth: prev, NextMonth: next, Stats: out, ShiftCodes: app.ShiftCodes,
	})
}

func (s *Server) teamPage(w http.ResponseWriter, r *http.Request, user app.User) {
	teamID, err := idParam(r, "teamId")
	if err != nil || !user.CanAccessTeam(teamID) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	month := queryMonth(r)
	if !app.ValidMonth(month) {
		month = app.CurrentMonth()
	}
	prev, _ := app.ShiftMonth(month, -1)
	next, _ := app.ShiftMonth(month, 1)
	teams, _ := s.store.ListTeams(user)
	employees, _ := s.store.ListEmployees()
	entries, err := s.store.TeamRoster(teamID, month)
	if err != nil {
		redirectWithError(w, r, "/", err.Error())
		return
	}
	headers, _ := app.BuildDayHeaders(month)
	var team app.Team
	for _, t := range teams {
		if t.ID == teamID {
			team = t
			break
		}
	}
	rowsByEmployee := map[int64]*app.RosterRow{}
	for _, entry := range entries {
		if _, ok := rowsByEmployee[entry.EmployeeID]; !ok {
			rowsByEmployee[entry.EmployeeID] = &app.RosterRow{
				Employee: app.Employee{ID: entry.EmployeeID, Name: entry.Name, EmpCode: entry.EmpCode, JobTitle: entry.JobTitle},
				Days:     map[int]app.RosterEntry{},
				Totals:   map[string]int{},
			}
		}
		day := 0
		_, _ = fmtSscanf(entry.Date[8:10], "%d", &day)
		rowsByEmployee[entry.EmployeeID].Days[day] = entry
		rowsByEmployee[entry.EmployeeID].Totals[entry.ShiftCode]++
	}
	rows := make([]app.RosterRow, 0, len(rowsByEmployee))
	assigned := map[int64]bool{}
	for _, row := range rowsByEmployee {
		rows = append(rows, *row)
		assigned[row.Employee.ID] = true
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Employee.Name < rows[j].Employee.Name })
	unassigned := []app.Employee{}
	for _, employee := range employees {
		if employee.TeamID != nil && *employee.TeamID == teamID && !assigned[employee.ID] {
			unassigned = append(unassigned, employee)
		}
	}
	s.render(w, "team", PageData{
		Title: "Team Roster", User: user, IsAdmin: user.IsAdmin(), Error: r.URL.Query().Get("error"),
		Month: month, PrevMonth: prev, NextMonth: next, Team: team, Headers: headers, Rows: rows,
		Employees: employees, Unassigned: unassigned, ShiftCodes: app.ShiftCodes,
	})
}

func (s *Server) adminPage(w http.ResponseWriter, r *http.Request, user app.User) {
	teams, _ := s.store.ListTeams(user)
	employees, _ := s.store.ListEmployees()
	users, _ := s.store.ListUsers()
	s.render(w, "admin", PageData{
		Title: "Admin", User: user, IsAdmin: true, Error: r.URL.Query().Get("error"),
		Teams: teams, Employees: employees, Users: users, ShiftCodes: app.ShiftCodes,
	})
}

func (s *Server) settingsPage(w http.ResponseWriter, r *http.Request, user app.User) {
	s.render(w, "settings", PageData{Title: "Settings", User: user, IsAdmin: true, Error: r.URL.Query().Get("error"), ShiftCodes: app.ShiftCodes})
}

// fmtSscanf keeps pages.go dependency-light in template-adjacent code.
func fmtSscanf(str, format string, a ...any) (int, error) {
	return fmt.Sscanf(str, format, a...)
}
