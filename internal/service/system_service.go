package service

import (
	"database/sql"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
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

type VersionInfo struct {
	AppVersion       string          `json:"app_version"`
	DbVersion        string          `json:"db_version"`
	Features         map[string]bool `json:"features"`
	MigrationNeeded  bool            `json:"migration_needed"`
	MigrationMessage *string         `json:"migration_message,omitempty"`
}

func (s *SystemService) CheckVersion() (VersionInfo, error) {
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

	return VersionInfo{
		AppVersion:       appVersion,
		DbVersion:        dbVersion,
		Features:         features,
		MigrationNeeded:  migrationNeeded,
		MigrationMessage: msgPtr,
	}, nil
}

//
// SUPPORTING FUNCTIONS
//

func (s *SystemService) getDbVersion() (string, error) {
	var versionNum string
	err := s.db.QueryRow("SELECT version_num FROM alembic_version").Scan(&versionNum)
	if err != nil {
		return "", err
	}
	return versionNum, nil
}

func (s *SystemService) checkFeatureAvailability(dbVersion string) map[string]bool {
	features := map[string]bool{
		"basic_portfolio_management":    true, // Introduced 1.1.1
		"realized_gain_loss":            true, // Introduced 1.1.1
		"ibkr_integration":              true, // Introduced 1.3.0
		"materialized_view_performance": true, // Introduced 1.4.0
		"fund_level_materialized_view":  true, // Introduced 1.5.0
	}

	_ = dbVersion

	// Parse version and set features
	// (version parsing logic here)
	// Currently not yet needed as we're always going to be on the latest version
	// due to the GO backend still being in development.

	// The original python code:
	// # Parse version and check feature availability
	// try:
	// 	if db_version != "unknown":
	// 		# Remove 'v' prefix if present and split version
	// 		version_clean = db_version.lstrip("v")
	// 		parts = version_clean.split(".")
	// 		major, minor, patch = int(parts[0]), int(parts[1]), int(parts[2])

	// 		# Version 1.1.1+: Realized gains/losses
	// 		if (
	// 			major > 1
	// 			or (major == 1 and minor > 1)
	// 			or (major == 1 and minor == 1 and patch >= 1)
	// 		):
	// 			features["realized_gain_loss"] = True

	// 		# Check for version 1.1.0+ with specific logic
	// 		if minor > 1 or (minor == 1 and patch >= 1):
	// 			features["realized_gain_loss"] = True

	// 		# Version 1.3.0+: IBKR integration
	// 		if major > 1 or (major == 1 and minor >= 3):
	// 			features["ibkr_integration"] = True

	// 		# Version 1.4.0+: Materialized view performance optimization
	// 		if major > 1 or (major == 1 and minor >= 4):
	// 			features["materialized_view_performance"] = True

	return features
}

func (s *SystemService) checkPendingMigrations(dbVersion, appVersion string) (bool, string) {

	_ = dbVersion
	_ = appVersion
	// Function also not yet implemented. This will be required once we have a db schema upgrade method implemented.
	// For Python we use Alembic but that's not going to be a thing yet.

	// The original python code:
	// try:
	// # Get the Alembic configuration
	// migrations_dir = os.path.join(os.path.dirname(__file__), "../../migrations")
	// alembic_cfg = Config(os.path.join(migrations_dir, "alembic.ini"))
	// alembic_cfg.set_main_option("script_location", migrations_dir)
	// # Set path_separator to suppress warning
	// alembic_cfg.set_main_option("path_separator", os.pathsep)

	// # Get script directory
	// script = ScriptDirectory.from_config(alembic_cfg)

	// # Get current revision from database
	// with db.engine.connect() as connection:
	// 	context = MigrationContext.configure(connection)
	// 	current_rev = context.get_current_revision()

	// 	# Get head revision (latest available migration)
	// 	head_rev = script.get_current_head()

	// 	# If current revision is None, database needs to be initialized
	// 	if current_rev is None:
	// 		return True, "Database not initialized"

	// 	# If current revision doesn't match head, there are pending migrations
	// 	if current_rev != head_rev:
	// 		return True, None

	// 	# No pending migrations
	// 	return False, None
	return false, ""
}
