package web

import (
	"database/sql"
	"net/http"
	"strconv"

	"roster_dashboard_go/internal/app"
)

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Username and password required")
		return
	}
	user, err := s.store.Authenticate(body.Username, body.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	token, err := s.setSession(w, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (s *Server) apiMe(w http.ResponseWriter, r *http.Request, user app.User) {
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) apiTeams(w http.ResponseWriter, r *http.Request, user app.User) {
	teams, err := s.store.ListTeams(user)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func (s *Server) apiCreateTeam(w http.ResponseWriter, r *http.Request, user app.User) {
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	team, err := s.store.CreateTeam(asString(body["name"]), asString(body["description"]))
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (s *Server) apiUpdateTeam(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := s.store.UpdateTeam(id, asString(body["name"]), asString(body["description"])); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiDeleteTeam(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	if err := s.store.DeleteTeam(id); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiEmployees(w http.ResponseWriter, r *http.Request, user app.User) {
	employees, err := s.store.ListEmployees()
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employees)
}

func (s *Server) apiSaveEmployee(w http.ResponseWriter, r *http.Request, user app.User) {
	id := int64(0)
	if r.Method == http.MethodPut {
		var err error
		id, err = idParam(r, "id")
		if err != nil {
			apiErr(w, err)
			return
		}
	}
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	employee, err := s.store.SaveEmployee(id, body)
	if err != nil {
		apiErr(w, err)
		return
	}
	if id > 0 {
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": employee.ID, "name": employee.Name})
}

func (s *Server) apiDeleteEmployee(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	if err := s.store.DeleteEmployee(id); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiBulkEditEmployees(w http.ResponseWriter, r *http.Request, user app.User) {
	var body struct {
		IDs   []int64 `json:"ids"`
		Field string  `json:"field"`
		Value any     `json:"value"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	count, err := s.store.BulkEditEmployees(body.IDs, body.Field, body.Value)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": count})
}

func (s *Server) apiImportEmployees(w http.ResponseWriter, r *http.Request, user app.User) {
	rows, ok := decodeRows(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ImportEmployees(rows))
}

func (s *Server) apiUsers(w http.ResponseWriter, r *http.Request, user app.User) {
	users, err := s.store.ListUsers()
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) apiCreateUser(w http.ResponseWriter, r *http.Request, user app.User) {
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	created, err := s.store.CreateUser(body)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) apiUpdateUser(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := s.store.UpdateUser(id, body); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiDeleteUser(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	if err := s.store.DeleteUser(user.ID, id); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiImportUsers(w http.ResponseWriter, r *http.Request, user app.User) {
	rows, ok := decodeRows(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ImportUsers(rows))
}

func (s *Server) apiRosterStats(w http.ResponseWriter, r *http.Request, user app.User) {
	stats, err := s.store.RosterStats(user, queryMonth(r))
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) apiTeamRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	teamID, err := idParam(r, "teamId")
	if err != nil {
		apiErr(w, err)
		return
	}
	if !user.CanAccessTeam(teamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	entries, err := s.store.TeamRoster(teamID, queryMonth(r))
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) apiCreateRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	teamID := int64FromAny(body["team_id"])
	if !user.IsAdmin() && user.TeamID != nil {
		teamID = *user.TeamID
	}
	if !user.CanAccessTeam(teamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	entry, err := s.store.UpsertRosterEntry(int64FromAny(body["employee_id"]), teamID, asString(body["shift_code"]), asString(body["date"]), asString(body["notes"]), false)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) apiUpdateRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	teamID, err := s.store.RosterEntryTeam(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Entry not found")
			return
		}
		apiErr(w, err)
		return
	}
	if !user.CanAccessTeam(teamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	entry, err := s.store.UpdateRosterEntry(id, asString(body["shift_code"]), asString(body["notes"]))
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) apiDeleteRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	id, err := idParam(r, "id")
	if err != nil {
		apiErr(w, err)
		return
	}
	teamID, err := s.store.RosterEntryTeam(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Entry not found")
		return
	}
	if !user.CanAccessTeam(teamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	if err := s.store.DeleteRosterEntry(id); err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) apiDeleteEmployeeRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	employeeID, err := idParam(r, "employeeId")
	if err != nil {
		apiErr(w, err)
		return
	}
	teamID, _ := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	month := r.URL.Query().Get("month")
	if teamID == 0 || month == "" {
		writeError(w, http.StatusBadRequest, "team_id and month are required")
		return
	}
	if !user.CanAccessTeam(teamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	deleted, err := s.store.DeleteEmployeeRoster(employeeID, teamID, month)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}

func (s *Server) apiBulkRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	var body struct {
		EmployeeID int64    `json:"employee_id"`
		TeamID     int64    `json:"team_id"`
		ShiftCode  string   `json:"shift_code"`
		Dates      []string `json:"dates"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !user.IsAdmin() && user.TeamID != nil {
		body.TeamID = *user.TeamID
	}
	if !user.CanAccessTeam(body.TeamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	updated, err := s.store.BulkAssign(body.EmployeeID, body.TeamID, body.ShiftCode, body.Dates)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": updated})
}

func (s *Server) apiCopyRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	var body struct {
		TeamID    int64  `json:"team_id"`
		FromMonth string `json:"from_month"`
		ToMonth   string `json:"to_month"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !user.CanAccessTeam(body.TeamID) {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}
	result, err := s.store.CopyRoster(body.TeamID, body.FromMonth, body.ToMonth)
	if err != nil {
		apiErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) apiImportRoster(w http.ResponseWriter, r *http.Request, user app.User) {
	rows, ok := decodeRows(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ImportRoster(rows))
}

func decodeRows(w http.ResponseWriter, r *http.Request) ([]map[string]any, bool) {
	var body struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.Rows) == 0 {
		writeError(w, http.StatusBadRequest, "rows array required")
		return nil, false
	}
	return body.Rows, true
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func int64FromAny(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		id, _ := strconv.ParseInt(t, 10, 64)
		return id
	default:
		return 0
	}
}
