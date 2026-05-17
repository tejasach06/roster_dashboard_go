package main

import (
	"log"
	"net/http"
	"os"

	"roster_dashboard_go/internal/auth"
	"roster_dashboard_go/internal/db"
	"roster_dashboard_go/internal/web"
)

func main() {
	port := getenv("PORT", "3001")
	dbPath := getenv("DB_PATH", "./roster.db")
	appEnv := getenv("APP_ENV", "development")

	secret, err := auth.SecretFromEnv(os.Getenv("SESSION_SECRET"), os.Getenv("JWT_SECRET"), appEnv)
	if err != nil {
		log.Fatal(err)
	}

	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	server := web.New(store, secret)
	log.Printf("Roster Go app running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
