package auth

import (
	"testing"
	"time"

	"roster_dashboard_go/internal/app"
)

func TestTokenRoundTrip(t *testing.T) {
	teamID := int64(7)
	user := app.User{ID: 1, Name: "Admin", Username: "admin", Role: "admin", TeamID: &teamID}
	token, err := Sign(FromUser(user, time.Hour), "secret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.ID != user.ID || claims.TeamID == nil || *claims.TeamID != teamID {
		t.Fatalf("claims mismatch: %#v", claims)
	}
	if _, err := Verify(token, "other-secret"); err == nil {
		t.Fatal("expected bad secret to fail")
	}
}

func TestProductionSecretRequired(t *testing.T) {
	if _, err := SecretFromEnv("", "", "production"); err == nil {
		t.Fatal("expected production default secret rejection")
	}
	if _, err := SecretFromEnv("short-secret", "", "production"); err == nil {
		t.Fatal("expected short production secret rejection")
	}
	if _, err := SecretFromEnv("0123456789abcdef0123456789abcdef", "", "production"); err != nil {
		t.Fatal(err)
	}
}
