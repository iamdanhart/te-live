package config

import (
	"os"
	"strings"
)

// Props holds application configuration derived from environment variables.
type Props struct {
	Env                string
	EnforceSignupLimit bool
	EnforceAdminAuth   bool
	DatabaseURL        string
}

// Load reads configuration from environment variables.
func Load() Props {
	dbURL := os.Getenv("DATABASE_URL")
	if schema := os.Getenv("DB_SCHEMA"); schema != "" {
		if strings.Contains(dbURL, "?") {
			dbURL += "&search_path=" + schema
		} else {
			dbURL += "?search_path=" + schema
		}
	}
	return Props{
		Env:                os.Getenv("ENV"),
		EnforceSignupLimit: os.Getenv("ENFORCE_SIGNUP_LIMIT") != "",
		EnforceAdminAuth:   os.Getenv("ENFORCE_ADMIN_AUTH") != "",
		DatabaseURL:        dbURL,
	}
}
