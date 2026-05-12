package feature_flag

import (
	"database/sql"
	"log/slog"
)

type PgFeatureFlag struct {
	db *sql.DB
}

func New(db *sql.DB) *PgFeatureFlag {
	return &PgFeatureFlag{db: db}
}

func (f *PgFeatureFlag) Enabled(flagKey string) bool {
	var enabled bool
	err := f.db.QueryRow(
		`SELECT enabled FROM telive.feature_flags WHERE key = $1`,
		flagKey,
	).Scan(&enabled)
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error("feature flag query", "key", flagKey, "err", err)
		}
		return false
	}
	return enabled
}