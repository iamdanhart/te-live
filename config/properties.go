package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// Props holds application configuration.
type Props struct {
	Env                string   `json:"-"`
	EnforceSignupLimit bool     `json:"enforce_signup_limit"`
	EnforceAdminAuth   bool     `json:"enforce_admin_auth"`
	DatabaseURL        string   `json:"-"`
	AllowedHosts       []string `json:"allowed_hosts"`
}

// Load reads configuration based on the ENV environment variable, falling back
// to dev. Secrets (DATABASE_URL) are always read from environment variables.
func Load() Props {
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	slog.Info("loading config", "env", env)

	f, err := openConfig(env)
	if err != nil {
		panic(fmt.Sprintf("config: cannot open %s config: %v", env, err))
	}
	defer f.Close()

	var props Props
	if err := json.NewDecoder(f).Decode(&props); err != nil {
		panic(fmt.Sprintf("config: cannot parse %s config: %v", env, err))
	}

	props.Env = env
	props.DatabaseURL = os.Getenv("DATABASE_URL")
	return props
}