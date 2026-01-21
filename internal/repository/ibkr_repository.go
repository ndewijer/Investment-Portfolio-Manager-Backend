package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// IbkrRepository provides data access methods for the ibkr table.
// It handles retrieving ibkr information.
type IbkrRepository struct {
	db *sql.DB
}

// NewIbkrRepository creates a new IbkrRepository with the provided database connection.
func NewIbkrRepository(db *sql.DB) *IbkrRepository {
	return &IbkrRepository{db: db}
}

func (r *IbkrRepository) GetIbkrConfig() (*model.IbkrConfig, error) {

	query := `
        SELECT flex_query_id, token_expires_at, last_import_date, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations
		FROM ibkr_config 
      `

	var ic model.IbkrConfig
	// var tokenWarning string
	var tokenExpiresStr, lastImportStr, defaultAllocationStr sql.NullString
	err := r.db.QueryRow(query).Scan(
		&ic.FlexQueryId,
		&tokenExpiresStr,
		&lastImportStr,
		&ic.AutoImportEnabled,
		&ic.CreatedAt,
		&ic.UpdatedAt,
		&ic.Enabled,
		&ic.DefaultAllocationEnabled,
		&defaultAllocationStr,
	)
	if err == sql.ErrNoRows {
		return &model.IbkrConfig{Configured: false}, nil
	}
	if err != nil {
		return nil, err
	}

	// Config exists in database
	ic.Configured = true

	if tokenExpiresStr.Valid {
		ic.TokenExpiresAt, err = ParseTime(tokenExpiresStr.String)
		if err != nil || ic.TokenExpiresAt.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date: %w", err)
		}
	}

	if lastImportStr.Valid {
		ic.LastImportDate, err = ParseTime(lastImportStr.String)
		if err != nil || ic.LastImportDate.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date: %w", err)
		}
	}

	if defaultAllocationStr.Valid {
		var defaultAllocation []model.Allocation
		if err := json.Unmarshal([]byte(defaultAllocationStr.String), &defaultAllocation); err != nil {
			return &ic, fmt.Errorf("failed to parse allocation model: %w", err)
		}
		ic.DefaultAllocations = defaultAllocation
	}

	return &ic, err
}
