package service_test

import (
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

// TestPortfolioService_GetAllPortfolios tests the GetAllPortfolios method.
//
// WHY: Portfolio retrieval is a fundamental operation. This ensures the service
// correctly returns all portfolios from the database, including edge cases like
// empty databases and multiple portfolios.
func TestPortfolioService_GetAllPortfolios(t *testing.T) {
	t.Run("returns empty slice when no portfolios exist", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 0 {
			t.Errorf("Expected empty slice, got %d portfolios", len(portfolios))
		}
	})

	t.Run("returns all portfolios including archived", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Create test data
		p1 := testutil.CreatePortfolio(t, db, "Active Portfolio")
		p2 := testutil.CreateArchivedPortfolio(t, db, "Archived Portfolio")

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(portfolios))
		}

		// Verify both portfolios are present
		foundActive := false
		foundArchived := false
		for _, p := range portfolios {
			if p.ID == p1.ID && p.Name == "Active Portfolio" {
				foundActive = true
			}
			if p.ID == p2.ID && p.Name == "Archived Portfolio" && p.IsArchived {
				foundArchived = true
			}
		}

		if !foundActive {
			t.Error("Active portfolio not found in results")
		}
		if !foundArchived {
			t.Error("Archived portfolio not found in results")
		}
	})

	t.Run("returns portfolios in correct order", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Create multiple portfolios
		testutil.CreatePortfolios(t, db, 5)

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err != nil {
			t.Fatalf("GetAllPortfolios() returned unexpected error: %v", err)
		}

		if len(portfolios) != 5 {
			t.Errorf("Expected 5 portfolios, got %d", len(portfolios))
		}
	})
}

// TestPortfolioService_GetAllPortfolios_DatabaseErrors tests error handling.
//
// WHY: The service must gracefully handle database errors without panicking,
// ensuring the application remains stable when the database is unavailable.
func TestPortfolioService_GetAllPortfolios_DatabaseErrors(t *testing.T) {
	t.Run("handles closed database connection", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)

		// Close database to force error
		db.Close()

		// Execute
		portfolios, err := svc.GetAllPortfolios()

		// Assert
		if err == nil {
			t.Error("Expected error when database is closed, got nil")
		}

		if portfolios != nil {
			t.Errorf("Expected nil portfolios on error, got %v", portfolios)
		}
	})
}

// Example of table-driven tests (another Go pattern)
func TestPortfolioService_GetAllPortfolios_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		setupData     func(*testing.T, *testutil.PortfolioBuilder) *testutil.PortfolioBuilder
		expectedCount int
	}{
		{
			name:          "no portfolios",
			setupData:     nil,
			expectedCount: 0,
		},
		{
			name: "single active portfolio",
			setupData: func(t *testing.T, b *testutil.PortfolioBuilder) *testutil.PortfolioBuilder {
				return b.WithName("Single Portfolio")
			},
			expectedCount: 1,
		},
		{
			name: "single archived portfolio",
			setupData: func(t *testing.T, b *testutil.PortfolioBuilder) *testutil.PortfolioBuilder {
				return b.WithName("Archived").Archived()
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := testutil.SetupTestDB(t)
			svc := testutil.NewTestPortfolioService(t, db)

			if tt.setupData != nil {
				builder := testutil.NewPortfolio()
				tt.setupData(t, builder).Build(t, db)
			}

			// Execute
			portfolios, err := svc.GetAllPortfolios()

			// Assert
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(portfolios) != tt.expectedCount {
				t.Errorf("Expected %d portfolios, got %d", tt.expectedCount, len(portfolios))
			}
		})
	}
}

// TODO: Add more tests as you implement more service methods:
// - TestPortfolioService_GetPortfolio
// - TestPortfolioService_CreatePortfolio
// - TestPortfolioService_UpdatePortfolio
// - TestPortfolioService_DeletePortfolio
// - TestPortfolioService_ArchivePortfolio
// etc.
