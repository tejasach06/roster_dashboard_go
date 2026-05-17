package web

import (
	"fmt"
	"net/http"
	"strings"

	"roster_dashboard_go/internal/app"
)

func (s *Server) teamAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	id := formInt(r, "id")
	action := r.FormValue("action")
	var err error
	switch action {
	case "delete":
		err = s.store.DeleteTeam(id)
	case "save":
		if id > 0 {
			err = s.store.UpdateTeam(id, r.FormValue("name"), r.FormValue("description"))
		} else {
			_, err = s.store.CreateTeam(r.FormValue("name"), r.FormValue("description"))
		}
	default:
		err = fmt.Errorf("Unknown action")
	}
	if err != nil {
		redirectWithError(w, r, "/admin", err.Error())
		return
	}
	redirectBack(w, r, "/admin")
}

func (s *Server) employeeAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	action := r.FormValue("action")
	var err error
	switch action {
	case "delete":
		err = s.store.DeleteEmployee(formInt(r, "id"))
	case "bulk":
		ids := parseIDs(r.Form["ids"])
		_, err = s.store.BulkEditEmployees(ids, r.FormValue("field"), r.FormValue("value"))
	default:
		_, err = s.store.SaveEmployee(formInt(r, "id"), stringMapFromForm(r))
	}
	if err != nil {
		redirectWithError(w, r, "/admin", err.Error())
		return
	}
	redirectBack(w, r, "/admin")
}

func (s *Server) userAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	id := formInt(r, "id")
	action := r.FormValue("action")
	var err error
	switch action {
	case "delete":
		err = s.store.DeleteUser(user.ID, id)
	case "save":
		if id > 0 {
			err = s.store.UpdateUser(id, stringMapFromForm(r))
		} else {
			_, err = s.store.CreateUser(stringMapFromForm(r))
		}
	default:
		err = fmt.Errorf("Unknown action")
	}
	if err != nil {
		redirectWithError(w, r, "/admin", err.Error())
		return
	}
	redirectBack(w, r, "/admin")
}

func (s *Server) rosterCellAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	teamID := formInt(r, "team_id")
	if !user.CanAccessTeam(teamID) {
		redirectWithError(w, r, "/", "Access denied")
		return
	}
	entryID := formInt(r, "entry_id")
	shiftCode := r.FormValue("shift_code")
	var err error
	if shiftCode == "" {
		if entryID > 0 {
			err = s.store.DeleteRosterEntry(entryID)
		}
	} else if entryID > 0 {
		_, err = s.store.UpdateRosterEntry(entryID, shiftCode, r.FormValue("notes"))
	} else {
		_, err = s.store.UpsertRosterEntry(formInt(r, "employee_id"), teamID, shiftCode, r.FormValue("date"), "", false)
	}
	target := fmt.Sprintf("/team/%d?month=%s", teamID, r.FormValue("month"))
	if err != nil {
		redirectWithError(w, r, target, err.Error())
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) rosterBulkAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	teamID := formInt(r, "team_id")
	if !user.CanAccessTeam(teamID) {
		redirectWithError(w, r, "/", "Access denied")
		return
	}
	month := r.FormValue("month")
	headers, _ := app.BuildDayHeaders(month)
	dates := make([]string, 0, len(headers))
	for _, h := range headers {
		dates = append(dates, h.Date)
	}
	_, err := s.store.BulkAssign(formInt(r, "employee_id"), teamID, r.FormValue("shift_code"), dates)
	target := fmt.Sprintf("/team/%d?month=%s", teamID, month)
	if err != nil {
		redirectWithError(w, r, target, err.Error())
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) rosterCopyAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	teamID := formInt(r, "team_id")
	if !user.CanAccessTeam(teamID) {
		redirectWithError(w, r, "/", "Access denied")
		return
	}
	fromMonth, toMonth := r.FormValue("from_month"), r.FormValue("to_month")
	result, err := s.store.CopyRoster(teamID, fromMonth, toMonth)
	target := fmt.Sprintf("/team/%d?month=%s", teamID, toMonth)
	if err != nil {
		redirectWithError(w, r, target, err.Error())
		return
	}
	if result.Copied == 0 {
		redirectWithError(w, r, target, "No entries copied")
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) rosterRemoveEmployeeAction(w http.ResponseWriter, r *http.Request, user app.User) {
	_ = r.ParseForm()
	teamID := formInt(r, "team_id")
	month := r.FormValue("month")
	if !user.CanAccessTeam(teamID) {
		redirectWithError(w, r, "/", "Access denied")
		return
	}
	_, err := s.store.DeleteEmployeeRoster(formInt(r, "employee_id"), teamID, month)
	target := fmt.Sprintf("/team/%d?month=%s", teamID, month)
	if err != nil {
		redirectWithError(w, r, target, err.Error())
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func splitCSVRows(raw string) []map[string]any {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) < 2 {
		return nil
	}
	headers := strings.Split(lines[0], ",")
	for i := range headers {
		headers[i] = strings.ToLower(strings.Trim(strings.TrimSpace(headers[i]), `"`))
	}
	var rows []map[string]any
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		values := strings.Split(line, ",")
		row := map[string]any{}
		for i, header := range headers {
			if i < len(values) {
				row[header] = strings.Trim(strings.TrimSpace(values[i]), `"`)
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows
}
