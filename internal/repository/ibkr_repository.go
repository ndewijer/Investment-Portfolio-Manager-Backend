package repository

import (
	"database/sql"

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

func (r *IbkrRepository) GetIbkrConfig() (model.IbkrConfig, error) {

	return model.IbkrConfig{}, nil
}
