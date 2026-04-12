package config

import "os"

// Props holds application configuration derived from environment variables.
type Props struct {
	Env                string
	EnforceSignupLimit bool
}

// Load reads configuration from environment variables.
func Load() Props {
	return Props{
		Env:                os.Getenv("ENV"),
		EnforceSignupLimit: os.Getenv("ENFORCE_SIGNUP_LIMIT") != "",
	}
}