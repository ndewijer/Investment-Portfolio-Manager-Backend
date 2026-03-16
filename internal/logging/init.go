package logging

import (
	"database/sql"
	"log/slog"
)

// Init creates a DBHandler, reads logging config from system_setting, and
// sets slog.SetDefault. Returns the handler for runtime config wiring.
//
// Chicken-and-egg: If called before migrations (table doesn't exist yet),
// config queries fail gracefully and defaults apply. DB writes will fail
// until the log table is created, falling back to stderr.
func Init(db *sql.DB) *DBHandler {
	h := NewDBHandler(db)

	// Read LOGGING_ENABLED (direct query — no repo import).
	var enabledStr string
	err := db.QueryRow("SELECT value FROM system_setting WHERE key = 'LOGGING_ENABLED'").Scan(&enabledStr)
	if err == nil {
		if enabledStr == "false" || enabledStr == "0" {
			h.SetEnabled(false)
		}
	}
	// On error (first run, no table yet): default enabled=true holds.

	// Read LOGGING_LEVEL.
	var levelStr string
	err = db.QueryRow("SELECT value FROM system_setting WHERE key = 'LOGGING_LEVEL'").Scan(&levelStr)
	if err == nil {
		h.SetLevel(DBStringToSlogLevel(levelStr))
	}
	// On error: default level=INFO holds.

	slog.SetDefault(slog.New(h))
	return h
}
