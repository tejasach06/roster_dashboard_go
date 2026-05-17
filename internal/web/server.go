package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"roster_dashboard_go/internal/app"
	"roster_dashboard_go/internal/auth"
	"roster_dashboard_go/internal/db"
)

//go:embed templates/*.html static/*
var embedded embed.FS

type Server struct {
	store  *db.Store
	secret string
	mux    *http.ServeMux
}

func New(store *db.Store, secret string) *Server {
	s := &Server{store: store, secret: secret, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	staticFS, _ := fs.Sub(embedded, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	s.mux.HandleFunc("GET /login", s.loginPage)
	s.mux.HandleFunc("POST /login", s.loginForm)
	s.mux.HandleFunc("POST /logout", s.logout)
	s.mux.HandleFunc("GET /", s.requireUser(s.dashboardPage))
	s.mux.HandleFunc("GET /team/{teamId}", s.requireUser(s.teamPage))
	s.mux.HandleFunc("GET /admin", s.requireAdmin(s.adminPage))
	s.mux.HandleFunc("GET /settings", s.requireAdmin(s.settingsPage))

	s.mux.HandleFunc("POST /actions/teams", s.requireAdmin(s.teamAction))
	s.mux.HandleFunc("POST /actions/employees", s.requireAdmin(s.employeeAction))
	s.mux.HandleFunc("POST /actions/users", s.requireAdmin(s.userAction))
	s.mux.HandleFunc("POST /actions/roster/cell", s.requireUser(s.rosterCellAction))
	s.mux.HandleFunc("POST /actions/roster/bulk", s.requireUser(s.rosterBulkAction))
	s.mux.HandleFunc("POST /actions/roster/copy", s.requireUser(s.rosterCopyAction))
	s.mux.HandleFunc("POST /actions/roster/remove-employee", s.requireUser(s.rosterRemoveEmployeeAction))

	s.mux.HandleFunc("POST /api/auth/login", s.apiLogin)
	s.mux.HandleFunc("GET /api/auth/me", s.requireAPIUser(s.apiMe))
	s.mux.HandleFunc("GET /api/teams", s.requireAPIUser(s.apiTeams))
	s.mux.HandleFunc("POST /api/teams", s.requireAPIAdmin(s.apiCreateTeam))
	s.mux.HandleFunc("PUT /api/teams/{id}", s.requireAPIAdmin(s.apiUpdateTeam))
	s.mux.HandleFunc("DELETE /api/teams/{id}", s.requireAPIAdmin(s.apiDeleteTeam))
	s.mux.HandleFunc("GET /api/employees", s.requireAPIUser(s.apiEmployees))
	s.mux.HandleFunc("POST /api/employees", s.requireAPIAdmin(s.apiSaveEmployee))
	s.mux.HandleFunc("PUT /api/employees/{id}", s.requireAPIAdmin(s.apiSaveEmployee))
	s.mux.HandleFunc("DELETE /api/employees/{id}", s.requireAPIAdmin(s.apiDeleteEmployee))
	s.mux.HandleFunc("POST /api/employees/bulk-import", s.requireAPIAdmin(s.apiImportEmployees))
	s.mux.HandleFunc("PUT /api/employees/bulk-edit", s.requireAPIAdmin(s.apiBulkEditEmployees))
	s.mux.HandleFunc("GET /api/users", s.requireAPIAdmin(s.apiUsers))
	s.mux.HandleFunc("POST /api/users", s.requireAPIAdmin(s.apiCreateUser))
	s.mux.HandleFunc("PUT /api/users/{id}", s.requireAPIAdmin(s.apiUpdateUser))
	s.mux.HandleFunc("DELETE /api/users/{id}", s.requireAPIAdmin(s.apiDeleteUser))
	s.mux.HandleFunc("POST /api/users/bulk-import", s.requireAPIAdmin(s.apiImportUsers))
	s.mux.HandleFunc("GET /api/roster/stats", s.requireAPIUser(s.apiRosterStats))
	s.mux.HandleFunc("GET /api/roster/team/{teamId}", s.requireAPIUser(s.apiTeamRoster))
	s.mux.HandleFunc("POST /api/roster", s.requireAPIUser(s.apiCreateRoster))
	s.mux.HandleFunc("PUT /api/roster/{id}", s.requireAPIUser(s.apiUpdateRoster))
	s.mux.HandleFunc("DELETE /api/roster/{id}", s.requireAPIUser(s.apiDeleteRoster))
	s.mux.HandleFunc("DELETE /api/roster/employee/{employeeId}", s.requireAPIUser(s.apiDeleteEmployeeRoster))
	s.mux.HandleFunc("POST /api/roster/bulk", s.requireAPIUser(s.apiBulkRoster))
	s.mux.HandleFunc("POST /api/roster/copy", s.requireAPIUser(s.apiCopyRoster))
	s.mux.HandleFunc("POST /api/roster/bulk-import", s.requireAPIAdmin(s.apiImportRoster))
}

type handlerWithUser func(http.ResponseWriter, *http.Request, app.User)

func (s *Server) requireUser(next handlerWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r, user)
	}
}

func (s *Server) requireAdmin(next handlerWithUser) http.HandlerFunc {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request, user app.User) {
		if !user.IsAdmin() {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r, user)
	})
}

func (s *Server) requireAPIUser(next handlerWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		next(w, r, user)
	}
}

func (s *Server) requireAPIAdmin(next handlerWithUser) http.HandlerFunc {
	return s.requireAPIUser(func(w http.ResponseWriter, r *http.Request, user app.User) {
		if !user.IsAdmin() {
			writeError(w, http.StatusForbidden, "Admin access required")
			return
		}
		next(w, r, user)
	})
}

func (s *Server) currentUser(r *http.Request) (app.User, bool) {
	token := ""
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		token = strings.TrimPrefix(header, "Bearer ")
	} else if cookie, err := r.Cookie("session"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		return app.User{}, false
	}
	claims, err := auth.Verify(token, s.secret)
	if err != nil {
		return app.User{}, false
	}
	user, err := s.store.UserByID(claims.ID)
	if err != nil {
		return app.User{}, false
	}
	return user, true
}

func (s *Server) setSession(w http.ResponseWriter, user app.User) (string, error) {
	token, err := auth.Sign(auth.FromUser(user, 8*time.Hour), s.secret)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((8 * time.Hour).Seconds()),
	})
	return token, nil
}

func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	funcs := template.FuncMap{
		"shiftClass": shiftClass,
		"shiftLabel": shiftLabel,
		"add":        func(a, b int) int { return a + b },
		"sub":        func(a, b int) int { return a - b },
		"monthLabel": monthLabel,
	}
	tmpl, err := template.New("layout.html").Funcs(funcs).ParseFS(embedded, "templates/layout.html", "templates/"+name+".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func idParam(r *http.Request, key string) (int64, error) {
	return auth.Int64Param(r.PathValue(key))
}

func queryMonth(r *http.Request) string {
	month := r.URL.Query().Get("month")
	if month == "" {
		return app.CurrentMonth()
	}
	return month
}

func formInt(r *http.Request, key string) int64 {
	id, _ := strconv.ParseInt(r.FormValue(key), 10, 64)
	return id
}

func redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	target := r.FormValue("redirect")
	if target == "" {
		target = r.Referer()
	}
	if target == "" {
		target = fallback
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func shiftClass(code string) string {
	switch code {
	case "MS":
		return "shift shift-ms"
	case "GS":
		return "shift shift-gs"
	case "AS":
		return "shift shift-as"
	case "NS":
		return "shift shift-ns"
	case "WO":
		return "shift shift-wo"
	case "EL":
		return "shift shift-el"
	default:
		return "shift"
	}
}

func shiftLabel(code string) string {
	switch code {
	case "MS":
		return "Morning Shift"
	case "GS":
		return "General Shift"
	case "AS":
		return "Afternoon Shift"
	case "NS":
		return "Night Shift"
	case "WO":
		return "Week Off"
	case "EL":
		return "Earned Leave"
	default:
		return code
	}
}

func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("January 2006")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func redirectWithError(w http.ResponseWriter, r *http.Request, target, msg string) {
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	http.Redirect(w, r, target+sep+"error="+url.QueryEscape(msg), http.StatusSeeOther)
}

func stringMapFromForm(r *http.Request) map[string]any {
	return map[string]any{
		"name": r.FormValue("name"), "username": r.FormValue("username"), "password": r.FormValue("password"),
		"role": r.FormValue("role"), "team_id": r.FormValue("team_id"),
		"emp_code": r.FormValue("emp_code"), "job_title": r.FormValue("job_title"), "email": r.FormValue("email"), "phone": r.FormValue("phone"),
	}
}

func parseIDs(values []string) []int64 {
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func apiErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "exists") {
		status = http.StatusConflict
	}
	if strings.Contains(strings.ToLower(msg), "no rows") {
		status = http.StatusNotFound
		msg = "Not found"
	}
	writeError(w, status, msg)
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("web setup failed: %v", err))
	}
}
