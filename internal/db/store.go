package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"roster_dashboard_go/internal/app"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: database}
	if err := s.Init(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) Init() error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			team_id INTEGER REFERENCES teams(id) ON DELETE SET NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS employees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			emp_code TEXT DEFAULT '',
			job_title TEXT DEFAULT '',
			email TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	_ = s.addColumn("employees", "emp_code", "TEXT DEFAULT ''")
	_ = s.addColumn("employees", "team_id", "INTEGER REFERENCES teams(id) ON DELETE SET NULL")

	var oldCols int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('roster_entries') WHERE name = 'month'`).Scan(&oldCols); err != nil {
		return err
	}
	if oldCols > 0 {
		if _, err := s.DB.Exec(`DROP TABLE IF EXISTS roster_entries`); err != nil {
			return err
		}
	}
	if _, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS roster_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		employee_id INTEGER NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
		team_id    INTEGER NOT NULL REFERENCES teams(id)    ON DELETE CASCADE,
		shift_code TEXT NOT NULL,
		date       TEXT NOT NULL,
		notes      TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(employee_id, date)
	)`); err != nil {
		return err
	}
	return s.seedAdmin()
}

func (s *Store) addColumn(table, name, definition string) error {
	_, err := s.DB.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, name, definition))
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

func (s *Store) seedAdmin() error {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM users WHERE username = 'admin'`).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	username := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_USERNAME"))
	if username == "" {
		username = "admin"
	}
	name := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_NAME"))
	if name == "" {
		name = "Admin"
	}
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if password == "" {
		if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
			return fmt.Errorf("INITIAL_ADMIN_PASSWORD must be set when seeding the first admin in production")
		}
		password = "admin123"
	}
	if len(password) < 8 {
		return fmt.Errorf("INITIAL_ADMIN_PASSWORD must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO users (name, username, password_hash, role) VALUES (?, ?, ?, 'admin')`, name, username, string(hash))
	return err
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func ptrInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}

func nullableString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func trim(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func intFromAny(v any) *int64 {
	switch t := v.(type) {
	case nil:
		return nil
	case float64:
		x := int64(t)
		if x == 0 {
			return nil
		}
		return &x
	case int:
		x := int64(t)
		if x == 0 {
			return nil
		}
		return &x
	case int64:
		if t == 0 {
			return nil
		}
		return &t
	case *int64:
		if t == nil || *t == 0 {
			return nil
		}
		return t
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		var x int64
		if _, err := fmt.Sscanf(t, "%d", &x); err == nil && x != 0 {
			return &x
		}
	}
	return nil
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func (s *Store) Authenticate(username, password string) (app.User, error) {
	var user app.User
	var teamID sql.NullInt64
	var hash string
	err := s.DB.QueryRow(`SELECT id, name, username, password_hash, role, team_id FROM users WHERE username = ?`, strings.TrimSpace(username)).
		Scan(&user.ID, &user.Name, &user.Username, &hash, &user.Role, &teamID)
	if err != nil {
		return user, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return user, sql.ErrNoRows
	}
	user.TeamID = ptrInt64(teamID)
	return user, nil
}

func (s *Store) UserByID(id int64) (app.User, error) {
	var user app.User
	var teamID sql.NullInt64
	err := s.DB.QueryRow(`SELECT id, name, username, role, team_id FROM users WHERE id = ?`, id).
		Scan(&user.ID, &user.Name, &user.Username, &user.Role, &teamID)
	user.TeamID = ptrInt64(teamID)
	return user, err
}

func (s *Store) ListUsers() ([]app.User, error) {
	rows, err := s.DB.Query(`SELECT u.id, u.name, u.username, u.role, u.team_id, COALESCE(t.name, '')
		FROM users u LEFT JOIN teams t ON t.id = u.team_id ORDER BY u.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []app.User
	for rows.Next() {
		var user app.User
		var teamID sql.NullInt64
		if err := rows.Scan(&user.ID, &user.Name, &user.Username, &user.Role, &teamID, &user.TeamName); err != nil {
			return nil, err
		}
		user.TeamID = ptrInt64(teamID)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateUser(fields map[string]any) (app.User, error) {
	name, username, password := trim(fields["name"]), trim(fields["username"]), trim(fields["password"])
	if name == "" || username == "" || password == "" {
		return app.User{}, fmt.Errorf("Name, username, and password are required")
	}
	if len(password) < 8 {
		return app.User{}, fmt.Errorf("Password must be at least 8 characters")
	}
	role := "member"
	if trim(fields["role"]) == "admin" {
		role = "admin"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return app.User{}, err
	}
	teamID := intFromAny(fields["team_id"])
	res, err := s.DB.Exec(`INSERT INTO users (name, username, password_hash, role, team_id) VALUES (?, ?, ?, ?, ?)`,
		name, username, string(hash), role, nullInt64(teamID))
	if err != nil {
		if isUnique(err) {
			return app.User{}, fmt.Errorf("Username already exists")
		}
		return app.User{}, err
	}
	id, _ := res.LastInsertId()
	return app.User{ID: id, Name: name, Username: username, Role: role, TeamID: teamID}, nil
}

func (s *Store) UpdateUser(id int64, fields map[string]any) error {
	name, username := trim(fields["name"]), trim(fields["username"])
	if name == "" || username == "" {
		return fmt.Errorf("Name and username are required")
	}
	password := trim(fields["password"])
	if password != "" && len(password) < 8 {
		return fmt.Errorf("Password must be at least 8 characters")
	}
	role := "member"
	if trim(fields["role"]) == "admin" {
		role = "admin"
	}
	if role != "admin" {
		target, err := s.UserByID(id)
		if err == nil && target.Role == "admin" {
			count, err := s.AdminCount()
			if err != nil {
				return err
			}
			if count <= 1 {
				return fmt.Errorf("Cannot demote the last admin account")
			}
		}
	}
	teamID := intFromAny(fields["team_id"])
	var err error
	if password != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), 10)
		if hashErr != nil {
			return hashErr
		}
		_, err = s.DB.Exec(`UPDATE users SET name = ?, username = ?, password_hash = ?, role = ?, team_id = ? WHERE id = ?`,
			name, username, string(hash), role, nullInt64(teamID), id)
	} else {
		_, err = s.DB.Exec(`UPDATE users SET name = ?, username = ?, role = ?, team_id = ? WHERE id = ?`,
			name, username, role, nullInt64(teamID), id)
	}
	if isUnique(err) {
		return fmt.Errorf("Username already exists")
	}
	return err
}

func (s *Store) DeleteUser(actorID, targetID int64) error {
	if actorID == targetID {
		return fmt.Errorf("You can't delete your own account")
	}
	target, err := s.UserByID(targetID)
	if err != nil {
		return err
	}
	if target.Role == "admin" {
		count, err := s.AdminCount()
		if err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("Cannot delete the last admin account")
		}
	}
	_, err = s.DB.Exec(`DELETE FROM users WHERE id = ?`, targetID)
	return err
}

func (s *Store) AdminCount() (int64, error) {
	var count int64
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count)
	return count, err
}

func (s *Store) ListTeams(user app.User) ([]app.Team, error) {
	query := `SELECT t.id, t.name, t.description, COUNT(re.id), t.created_at
		FROM teams t LEFT JOIN roster_entries re ON re.team_id = t.id`
	args := []any{}
	if !user.IsAdmin() {
		if user.TeamID == nil {
			return []app.Team{}, nil
		}
		query += ` WHERE t.id = ?`
		args = append(args, *user.TeamID)
	}
	query += ` GROUP BY t.id ORDER BY t.name`
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []app.Team
	for rows.Next() {
		var t app.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.MemberCount, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (s *Store) CreateTeam(name, description string) (app.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return app.Team{}, fmt.Errorf("Team name is required")
	}
	res, err := s.DB.Exec(`INSERT INTO teams (name, description) VALUES (?, ?)`, name, strings.TrimSpace(description))
	if isUnique(err) {
		return app.Team{}, fmt.Errorf("Team name already exists")
	}
	if err != nil {
		return app.Team{}, err
	}
	id, _ := res.LastInsertId()
	return app.Team{ID: id, Name: name, Description: strings.TrimSpace(description)}, nil
}

func (s *Store) UpdateTeam(id int64, name, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("Team name is required")
	}
	_, err := s.DB.Exec(`UPDATE teams SET name = ?, description = ? WHERE id = ?`, name, strings.TrimSpace(description), id)
	if isUnique(err) {
		return fmt.Errorf("Team name already exists")
	}
	return err
}

func (s *Store) DeleteTeam(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM teams WHERE id = ?`, id)
	return err
}

func (s *Store) ListEmployees() ([]app.Employee, error) {
	rows, err := s.DB.Query(`SELECT e.id, e.name, e.emp_code, e.job_title, e.email, e.phone, e.team_id, COALESCE(t.name, ''), e.created_at
		FROM employees e LEFT JOIN teams t ON e.team_id = t.id ORDER BY e.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var employees []app.Employee
	for rows.Next() {
		var e app.Employee
		var teamID sql.NullInt64
		if err := rows.Scan(&e.ID, &e.Name, &e.EmpCode, &e.JobTitle, &e.Email, &e.Phone, &teamID, &e.TeamName, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.TeamID = ptrInt64(teamID)
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

func (s *Store) SaveEmployee(id int64, fields map[string]any) (app.Employee, error) {
	name := trim(fields["name"])
	if name == "" {
		return app.Employee{}, fmt.Errorf("Name is required")
	}
	employee := app.Employee{
		ID:       id,
		Name:     name,
		EmpCode:  trim(fields["emp_code"]),
		JobTitle: trim(fields["job_title"]),
		Email:    trim(fields["email"]),
		Phone:    trim(fields["phone"]),
		TeamID:   intFromAny(fields["team_id"]),
	}
	if id == 0 {
		res, err := s.DB.Exec(`INSERT INTO employees (name, emp_code, job_title, email, phone, team_id) VALUES (?, ?, ?, ?, ?, ?)`,
			employee.Name, employee.EmpCode, employee.JobTitle, employee.Email, employee.Phone, nullInt64(employee.TeamID))
		if err != nil {
			return employee, err
		}
		employee.ID, _ = res.LastInsertId()
		return employee, nil
	}
	_, err := s.DB.Exec(`UPDATE employees SET name = ?, emp_code = ?, job_title = ?, email = ?, phone = ?, team_id = ? WHERE id = ?`,
		employee.Name, employee.EmpCode, employee.JobTitle, employee.Email, employee.Phone, nullInt64(employee.TeamID), id)
	return employee, err
}

func (s *Store) DeleteEmployee(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM employees WHERE id = ?`, id)
	return err
}

func (s *Store) BulkEditEmployees(ids []int64, field string, value any) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("ids array required")
	}
	if field != "job_title" && field != "team_id" {
		return 0, fmt.Errorf("Invalid field")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(fmt.Sprintf(`UPDATE employees SET %s = ? WHERE id = ?`, field))
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for _, id := range ids {
		v := value
		if field == "team_id" {
			v = nullInt64(intFromAny(value))
		}
		if _, err := stmt.Exec(v, id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (s *Store) RosterStats(user app.User, month string) ([]app.StatRow, error) {
	start, end, _, err := app.MonthBounds(month)
	if err != nil {
		return nil, err
	}
	query := `SELECT t.id, t.name, re.shift_code, COUNT(re.id)
		FROM teams t
		LEFT JOIN roster_entries re ON re.team_id = t.id AND re.date >= ? AND re.date <= ?
		WHERE 1=1`
	args := []any{start, end}
	if !user.IsAdmin() {
		query += ` AND t.id = ?`
		if user.TeamID == nil {
			args = append(args, -1)
		} else {
			args = append(args, *user.TeamID)
		}
	}
	query += ` GROUP BY t.id, re.shift_code ORDER BY t.name`
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []app.StatRow
	for rows.Next() {
		var row app.StatRow
		var code sql.NullString
		if err := rows.Scan(&row.TeamID, &row.TeamName, &code, &row.Count); err != nil {
			return nil, err
		}
		if code.Valid {
			row.ShiftCode = &code.String
		}
		stats = append(stats, row)
	}
	return stats, rows.Err()
}

const entryJoin = `SELECT re.id, re.shift_code, re.date, re.notes,
	e.id AS employee_id, e.name, e.emp_code, e.job_title, e.email, e.phone
	FROM roster_entries re JOIN employees e ON e.id = re.employee_id`

func (s *Store) TeamRoster(teamID int64, month string) ([]app.RosterEntry, error) {
	start, end, _, err := app.MonthBounds(month)
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(entryJoin+` WHERE re.team_id = ? AND re.date >= ? AND re.date <= ? ORDER BY e.name, re.date`, teamID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]app.RosterEntry, error) {
	var entries []app.RosterEntry
	for rows.Next() {
		var e app.RosterEntry
		if err := rows.Scan(&e.ID, &e.ShiftCode, &e.Date, &e.Notes, &e.EmployeeID, &e.Name, &e.EmpCode, &e.JobTitle, &e.Email, &e.Phone); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) UpsertRosterEntry(employeeID, teamID int64, shiftCode, date, notes string, update bool) (app.RosterEntry, error) {
	if employeeID == 0 {
		return app.RosterEntry{}, fmt.Errorf("employee_id is required")
	}
	if !app.ValidDate(date) {
		return app.RosterEntry{}, fmt.Errorf("date must be YYYY-MM-DD")
	}
	if !app.ValidShift(shiftCode) {
		return app.RosterEntry{}, fmt.Errorf("Invalid shift_code")
	}
	if teamID == 0 {
		return app.RosterEntry{}, fmt.Errorf("team_id is required")
	}
	var id int64
	if update {
		_, err := s.DB.Exec(`INSERT INTO roster_entries (employee_id, team_id, shift_code, date, notes)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(employee_id, date) DO UPDATE SET shift_code = excluded.shift_code, notes = excluded.notes, updated_at = CURRENT_TIMESTAMP`,
			employeeID, teamID, shiftCode, date, strings.TrimSpace(notes))
		if err != nil {
			return app.RosterEntry{}, err
		}
		err = s.DB.QueryRow(`SELECT id FROM roster_entries WHERE employee_id = ? AND date = ?`, employeeID, date).Scan(&id)
		if err != nil {
			return app.RosterEntry{}, err
		}
	} else {
		res, err := s.DB.Exec(`INSERT INTO roster_entries (employee_id, team_id, shift_code, date, notes) VALUES (?, ?, ?, ?, ?)`,
			employeeID, teamID, shiftCode, date, strings.TrimSpace(notes))
		if isUnique(err) {
			return app.RosterEntry{}, fmt.Errorf("Entry already exists for this date")
		}
		if err != nil {
			return app.RosterEntry{}, err
		}
		id, _ = res.LastInsertId()
	}
	return s.RosterEntryByID(id)
}

func (s *Store) RosterEntryByID(id int64) (app.RosterEntry, error) {
	var e app.RosterEntry
	err := s.DB.QueryRow(entryJoin+` WHERE re.id = ?`, id).
		Scan(&e.ID, &e.ShiftCode, &e.Date, &e.Notes, &e.EmployeeID, &e.Name, &e.EmpCode, &e.JobTitle, &e.Email, &e.Phone)
	return e, err
}

func (s *Store) RosterEntryTeam(id int64) (int64, error) {
	var teamID int64
	err := s.DB.QueryRow(`SELECT team_id FROM roster_entries WHERE id = ?`, id).Scan(&teamID)
	return teamID, err
}

func (s *Store) UpdateRosterEntry(id int64, shiftCode, notes string) (app.RosterEntry, error) {
	if !app.ValidShift(shiftCode) {
		return app.RosterEntry{}, fmt.Errorf("Invalid shift_code")
	}
	existing, err := s.RosterEntryByID(id)
	if err != nil {
		return app.RosterEntry{}, err
	}
	if strings.TrimSpace(notes) == "" {
		notes = existing.Notes
	}
	_, err = s.DB.Exec(`UPDATE roster_entries SET shift_code = ?, notes = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, shiftCode, strings.TrimSpace(notes), id)
	if err != nil {
		return app.RosterEntry{}, err
	}
	return s.RosterEntryByID(id)
}

func (s *Store) DeleteRosterEntry(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM roster_entries WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteEmployeeRoster(employeeID, teamID int64, month string) (int64, error) {
	start, end, _, err := app.MonthBounds(month)
	if err != nil {
		return 0, err
	}
	res, err := s.DB.Exec(`DELETE FROM roster_entries WHERE employee_id = ? AND team_id = ? AND date >= ? AND date <= ?`, employeeID, teamID, start, end)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) BulkAssign(employeeID, teamID int64, shiftCode string, dates []string) (int, error) {
	if employeeID == 0 || shiftCode == "" || len(dates) == 0 {
		return 0, fmt.Errorf("employee_id, shift_code, and dates[] are required")
	}
	if !app.ValidShift(shiftCode) {
		return 0, fmt.Errorf("Invalid shift_code")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO roster_entries (employee_id, team_id, shift_code, date, notes)
		VALUES (?, ?, ?, ?, '')
		ON CONFLICT(employee_id, date) DO UPDATE SET shift_code = excluded.shift_code, updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	count := 0
	for _, date := range dates {
		if app.ValidDate(date) {
			if _, err := stmt.Exec(employeeID, teamID, shiftCode, date); err != nil {
				return 0, err
			}
			count++
		}
	}
	return count, tx.Commit()
}

func (s *Store) CopyRoster(teamID int64, fromMonth, toMonth string) (app.ImportResult, error) {
	fStart, fEnd, _, err := app.MonthBounds(fromMonth)
	if err != nil {
		return app.ImportResult{}, err
	}
	_, _, toLast, err := app.MonthBounds(toMonth)
	if err != nil {
		return app.ImportResult{}, err
	}
	rows, err := s.DB.Query(`SELECT employee_id, team_id, shift_code, date, notes FROM roster_entries WHERE team_id = ? AND date >= ? AND date <= ?`, teamID, fStart, fEnd)
	if err != nil {
		return app.ImportResult{}, err
	}
	defer rows.Close()
	type copyRow struct {
		employeeID int64
		teamID     int64
		shiftCode  string
		date       string
		notes      string
	}
	var entries []copyRow
	for rows.Next() {
		var row copyRow
		if err := rows.Scan(&row.employeeID, &row.teamID, &row.shiftCode, &row.date, &row.notes); err != nil {
			return app.ImportResult{}, err
		}
		entries = append(entries, row)
	}
	if err := rows.Err(); err != nil {
		return app.ImportResult{}, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return app.ImportResult{}, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO roster_entries (employee_id, team_id, shift_code, date, notes) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return app.ImportResult{}, err
	}
	defer stmt.Close()
	copied := 0
	for _, row := range entries {
		var day int
		fmt.Sscanf(row.date[8:10], "%d", &day)
		if day > toLast {
			continue
		}
		res, err := stmt.Exec(row.employeeID, row.teamID, row.shiftCode, fmt.Sprintf("%s-%02d", toMonth, day), row.notes)
		if err != nil {
			return app.ImportResult{}, err
		}
		changes, _ := res.RowsAffected()
		if changes > 0 {
			copied++
		}
	}
	if err := tx.Commit(); err != nil {
		return app.ImportResult{}, err
	}
	return app.ImportResult{Copied: copied, Skipped: len(entries) - copied}, nil
}

func (s *Store) ImportEmployees(rows []map[string]any) app.ImportResult {
	result := app.ImportResult{}
	for _, row := range rows {
		name := trim(row["name"])
		if name == "" {
			result.Skipped++
			continue
		}
		empCode := trim(row["emp_code"])
		var existing int64
		var err error
		if empCode != "" {
			err = s.DB.QueryRow(`SELECT id FROM employees WHERE emp_code = ? OR name = ? LIMIT 1`, empCode, name).Scan(&existing)
		} else {
			err = s.DB.QueryRow(`SELECT id FROM employees WHERE name = ? LIMIT 1`, name).Scan(&existing)
		}
		if err == nil {
			result.Skipped++
			continue
		}
		if err != sql.ErrNoRows {
			result.Errors = append(result.Errors, "Failed to check employee "+name)
			result.Skipped++
			continue
		}
		_, err = s.DB.Exec(`INSERT INTO employees (name, emp_code, job_title, email, phone) VALUES (?, ?, ?, ?, ?)`,
			name, empCode, trim(row["job_title"]), trim(row["email"]), trim(row["phone"]))
		if err != nil {
			result.Errors = append(result.Errors, "Failed to import employee "+name)
			result.Skipped++
			continue
		}
		result.Created++
	}
	return result
}

func (s *Store) ImportUsers(rows []map[string]any) app.ImportResult {
	result := app.ImportResult{}
	for _, row := range rows {
		name, username, password := trim(row["name"]), trim(row["username"]), trim(row["password"])
		if name == "" || username == "" || password == "" {
			result.Errors = append(result.Errors, "Missing name/username/password for row: "+username)
			result.Skipped++
			continue
		}
		if len(password) < 8 {
			result.Errors = append(result.Errors, fmt.Sprintf(`Password too short for user "%s" (min 8 chars)`, username))
			result.Skipped++
			continue
		}
		var teamID *int64
		teamName := trim(row["team_name"])
		if teamName != "" {
			var id int64
			if err := s.DB.QueryRow(`SELECT id FROM teams WHERE name = ?`, teamName).Scan(&id); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf(`Team not found: "%s" (user: %s)`, teamName, username))
				result.Skipped++
				continue
			}
			teamID = &id
		}
		_, err := s.CreateUser(map[string]any{
			"name": name, "username": username, "password": password, "role": trim(row["role"]), "team_id": teamID,
		})
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Skipped++
		} else {
			result.Created++
		}
	}
	if len(result.Errors) > 20 {
		result.Errors = result.Errors[:20]
	}
	return result
}

func (s *Store) ImportRoster(rows []map[string]any) app.ImportResult {
	result := app.ImportResult{}
	for _, row := range rows {
		if trim(row["date"]) == "" && trim(row["shift_code"]) == "" && trim(row["month"]) != "" {
			for day := 1; day <= 31; day++ {
				shiftCode := strings.ToUpper(trim(row[fmt.Sprintf("%d", day)]))
				if shiftCode == "" {
					continue
				}
				s.importRosterRow(map[string]any{
					"emp_code":   row["emp_code"],
					"date":       fmt.Sprintf("%s-%02d", trim(row["month"]), day),
					"shift_code": shiftCode,
					"team_name":  row["team_name"],
				}, &result)
			}
			continue
		}
		s.importRosterRow(row, &result)
	}
	if len(result.Errors) > 30 {
		result.Errors = result.Errors[:30]
	}
	return result
}

func (s *Store) importRosterRow(row map[string]any, result *app.ImportResult) {
	empCode, date, shiftCode, teamName := trim(row["emp_code"]), trim(row["date"]), strings.ToUpper(trim(row["shift_code"])), trim(row["team_name"])
	if empCode == "" || date == "" || shiftCode == "" || teamName == "" {
		result.Errors = append(result.Errors, "Missing fields in row")
		return
	}
	if !app.ValidShift(shiftCode) {
		result.Errors = append(result.Errors, fmt.Sprintf(`Invalid shift_code "%s" (%s %s)`, shiftCode, empCode, date))
		return
	}
	if !app.ValidDate(date) {
		result.Errors = append(result.Errors, fmt.Sprintf(`Invalid date "%s" for %s`, date, empCode))
		return
	}
	var employeeID, teamID int64
	if err := s.DB.QueryRow(`SELECT id FROM employees WHERE emp_code = ?`, empCode).Scan(&employeeID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf(`Employee not found: emp_code "%s"`, empCode))
		return
	}
	if err := s.DB.QueryRow(`SELECT id FROM teams WHERE name = ?`, teamName).Scan(&teamID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf(`Team not found: "%s"`, teamName))
		return
	}
	if _, err := s.UpsertRosterEntry(employeeID, teamID, shiftCode, date, "", true); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to import row (%s %s)", empCode, date))
	} else {
		result.Imported++
	}
}
