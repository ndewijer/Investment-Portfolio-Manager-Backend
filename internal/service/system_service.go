package service

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/version"
)

// SystemService handles system-related operations
type SystemService struct {
	db *sql.DB
}

// NewSystemService creates a new SystemService
func NewSystemService(db *sql.DB) *SystemService {
	return &SystemService{
		db: db,
	}
}

// CheckHealth checks the health of the system
func (s *SystemService) CheckHealth() error {
	return database.HealthCheck(s.db)
}

// CheckVersion retrieves version information including app version, database version,
// feature availability, and pending migration status.
func (s *SystemService) CheckVersion() (model.VersionInfo, error) {
	appVersion := version.Version
	dbVersion, err := s.getDbVersion()
	if err != nil {
		dbVersion = "unknown"
	}

	features := s.checkFeatureAvailability(dbVersion)
	migrationNeeded, migrationMsg := s.checkPendingMigrations(dbVersion, appVersion)

	var msgPtr *string
	if migrationMsg != "" {
		msgPtr = &migrationMsg
	}

	return model.VersionInfo{
		AppVersion:       appVersion,
		DbVersion:        dbVersion,
		Features:         features,
		MigrationNeeded:  migrationNeeded,
		MigrationMessage: msgPtr,
	}, nil
}

// getDbVersion retrieves the current database schema version.
// It queries goose_db_version first; for legacy Python-migrated DBs it falls back to alembic_version.
func (s *SystemService) getDbVersion() (string, error) {
	var versionID int64
	err := s.db.QueryRow(
		"SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1",
	).Scan(&versionID)
	if err == nil {
		return fmt.Sprintf("%d", versionID), nil
	}
	// Fallback for legacy Python DBs
	var versionNum string
	err = s.db.QueryRow("SELECT version_num FROM alembic_version").Scan(&versionNum)
	if err != nil {
		return "", err
	}
	return versionNum, nil
}

// checkFeatureAvailability determines which features are available based on the database version.
func (s *SystemService) checkFeatureAvailability(dbVersion string) map[string]bool {
	features := map[string]bool{
		"basic_portfolio_management":    true, // Introduced 1.1.1
		"realized_gain_loss":            true, // Introduced 1.1.1
		"ibkr_integration":              true, // Introduced 1.3.0
		"materialized_view_performance": true, // Introduced 1.4.0
		"fund_level_materialized_view":  true, // Introduced 1.5.0
	}

	_ = dbVersion

	return features
}

// checkPendingMigrations checks if the database schema is behind the application version.
// App version "1.6.3" maps to schema version 163 (dots removed). If the app schema version
// is higher than the applied DB schema version, a migration is needed.
func (s *SystemService) checkPendingMigrations(dbVersion, appVersion string) (bool, string) {
	appSchemaVersion, err := strconv.Atoi(strings.ReplaceAll(appVersion, ".", ""))
	if err != nil {
		// Non-numeric version (e.g. "dev") — cannot compare
		return false, ""
	}

	dbSchemaVersion, err := strconv.Atoi(dbVersion)
	if err != nil {
		// DB version is unknown or a legacy alembic hex string — cannot compare
		return false, ""
	}

	if appSchemaVersion > dbSchemaVersion {
		return true, fmt.Sprintf("Database schema version %s is behind app version %s; run migrations to upgrade", dbVersion, appVersion)
	}

	return false, ""
}
