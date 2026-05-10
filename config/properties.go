package config

import "os"

// Props holds application configuration derived from environment variables.
type Props struct {
	Env                string
	EnforceSignupLimit bool
	EnforceAdminAuth   bool
	DatabaseURL        string
	Schema             string
}

// Load reads configuration from environment variables.
func Load() Props {
	return Props{
		Env:                os.Getenv("ENV"),
		EnforceSignupLimit: os.Getenv("ENFORCE_SIGNUP_LIMIT") != "",
		EnforceAdminAuth:   os.Getenv("ENFORCE_ADMIN_AUTH") != "",
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Schema:             os.Getenv("DB_SCHEMA"),
	}
}
