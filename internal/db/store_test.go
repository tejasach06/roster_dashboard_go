package db

import (
	"path/filepath"
	"testing"

	"roster_dashboard_go/internal/app"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "roster.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestProductionSeedAdminRequiresConfiguredPassword(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")

	store, err := Open(filepath.Join(t.TempDir(), "roster.db"))
	if err == nil {
		_ = store.Close()
		t.Fatal("expected production seed without password to fail")
	}
}

func TestSeedAdminCanUseConfiguredCredentials(t *testing.T) {
	t.Setenv("INITIAL_ADMIN_USERNAME", "owner")
	t.Setenv("INITIAL_ADMIN_NAME", "Owner")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "configured123")

	store := newTestStore(t)
	user, err := store.Authenticate("owner", "configured123")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Owner" || user.Role != "admin" {
		t.Fatalf("seeded user mismatch: %#v", user)
	}
}

func TestSeedAdminAndTeamScopedLists(t *testing.T) {
	store := newTestStore(t)
	admin, err := store.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeam("Support Alpha", "")
	if err != nil {
		t.Fatal(err)
	}
	member, err := store.CreateUser(map[string]any{"name": "Mina", "username": "mina", "password": "password1", "role": "member", "team_id": team.ID})
	if err != nil {
		t.Fatal(err)
	}
	adminTeams, err := store.ListTeams(admin)
	if err != nil || len(adminTeams) != 1 {
		t.Fatalf("admin teams = %d, err = %v", len(adminTeams), err)
	}
	memberTeams, err := store.ListTeams(member)
	if err != nil || len(memberTeams) != 1 || memberTeams[0].ID != team.ID {
		t.Fatalf("member teams = %#v, err = %v", memberTeams, err)
	}
}

func TestRosterBulkCopyAndLastAdminProtection(t *testing.T) {
	store := newTestStore(t)
	admin, _ := store.Authenticate("admin", "admin123")
	team, _ := store.CreateTeam("Support Alpha", "")
	emp, err := store.SaveEmployee(0, map[string]any{"name": "Alice", "emp_code": "A1", "team_id": team.ID})
	if err != nil {
		t.Fatal(err)
	}
	dates := []string{"2026-05-01", "2026-05-02"}
	if updated, err := store.BulkAssign(emp.ID, team.ID, "GS", dates); err != nil || updated != 2 {
		t.Fatalf("bulk assign updated=%d err=%v", updated, err)
	}
	result, err := store.CopyRoster(team.ID, "2026-05", "2026-06")
	if err != nil || result.Copied != 2 {
		t.Fatalf("copy result=%#v err=%v", result, err)
	}
	if err := store.DeleteUser(admin.ID, admin.ID); err == nil {
		t.Fatal("expected self-delete to fail")
	}
	if err := store.UpdateUser(admin.ID, map[string]any{"name": "Admin", "username": "admin", "role": "member"}); err == nil {
		t.Fatal("expected last admin demotion to fail")
	}
}

func TestImports(t *testing.T) {
	store := newTestStore(t)
	team, _ := store.CreateTeam("Support Alpha", "")
	employees := store.ImportEmployees([]map[string]any{{"name": "Alice", "emp_code": "A1"}})
	if employees.Created != 1 {
		t.Fatalf("employees import: %#v", employees)
	}
	employees = store.ImportEmployees([]map[string]any{{"name": "Alice", "emp_code": "A1"}})
	if employees.Created != 0 || employees.Skipped != 1 {
		t.Fatalf("duplicate employees import: %#v", employees)
	}
	roster := store.ImportRoster([]map[string]any{{"emp_code": "A1", "date": "2026-05-01", "shift_code": "MS", "team_name": team.Name}})
	if roster.Imported != 1 {
		t.Fatalf("roster import: %#v", roster)
	}
	stats, err := store.RosterStats(app.User{ID: 1, Role: "admin"}, "2026-05")
	if err != nil || len(stats) == 0 {
		t.Fatalf("stats len=%d err=%v", len(stats), err)
	}
}
