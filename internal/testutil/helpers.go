package testutil

import (
	"database/sql"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/yahoo"
)

func NewTestPortfolioService(t *testing.T, db *sql.DB) *service.PortfolioService {
	t.Helper()

	portfolioRepo := repository.NewPortfolioRepository(db)

	return service.NewPortfolioService(
		portfolioRepo,
	)
}

func NewTestRealizedGainLossService(t *testing.T, db *sql.DB) *service.RealizedGainLossService {
	t.Helper()

	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)

	return service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
}

func NewTestTransactionService(t *testing.T, db *sql.DB) *service.TransactionService {
	t.Helper()

	transactionRepo := repository.NewTransactionRepository(db)

	return service.NewTransactionService(
		transactionRepo,
	)
}

func NewTestDividendService(t *testing.T, db *sql.DB) *service.DividendService {
	t.Helper()

	dividendRepo := repository.NewDividendRepository(db)

	return service.NewDividendService(
		dividendRepo,
	)
}

func NewTestDataloaderService(t *testing.T, db *sql.DB) *service.DataLoaderService {
	t.Helper()

	portfolioRepo := repository.NewPortfolioRepository(db)
	fundRepo := repository.NewFundRepository(db)
	transactionService := service.NewTransactionService(repository.NewTransactionRepository(db))
	dividendService := service.NewDividendService(repository.NewDividendRepository(db))
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))

	return service.NewDataLoaderService(
		portfolioRepo,
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService,
	)
}

func NewTestFundService(t *testing.T, db *sql.DB) *service.FundService {
	t.Helper()

	fundRepo := repository.NewFundRepository(db)
	transactionService := service.NewTransactionService(repository.NewTransactionRepository(db))
	dividendService := service.NewDividendService(repository.NewDividendRepository(db))
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))
	dataLoaderService := service.NewDataLoaderService(
		repository.NewPortfolioRepository(db),
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService)
	portfolioService := service.NewPortfolioService(repository.NewPortfolioRepository(db))
	yahooClient := yahoo.NewFinanceClient()

	return service.NewFundService(
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService,
		dataLoaderService,
		portfolioService,
		yahooClient,
	)
}

func NewTestMaterializedService(t *testing.T, db *sql.DB) *service.MaterializedService {
	t.Helper()

	materializedRepo := repository.NewMaterializedRepository(db)
	portfolioRepo := repository.NewPortfolioRepository(db)
	fundRepo := repository.NewFundRepository(db)
	transactionService := service.NewTransactionService(repository.NewTransactionRepository(db))

	dividendService := service.NewDividendService(repository.NewDividendRepository(db))
	realizedGainLossService := service.NewRealizedGainLossService(repository.NewRealizedGainLossRepository(db))
	dataLoaderService := service.NewDataLoaderService(
		repository.NewPortfolioRepository(db),
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService)
	portfolioService := service.NewPortfolioService(portfolioRepo)
	yahooClient := yahoo.NewFinanceClient()
	fundService := service.NewFundService(fundRepo, transactionService, dividendService, realizedGainLossService, dataLoaderService, portfolioService, yahooClient)

	return service.NewMaterializedService(
		materializedRepo,
		portfolioRepo,
		fundRepo,
		transactionService,
		fundService,
		dividendService,
		realizedGainLossService,
		dataLoaderService,
		portfolioService,
	)
}

func NewTestIbkrService(t *testing.T, db *sql.DB) *service.IbkrService {
	t.Helper()

	ibkrRepo := repository.NewIbkrRepository(db)
	transactionService := service.NewTransactionService(repository.NewTransactionRepository(db))
	fundRepo := repository.NewFundRepository(db)

	return service.NewIbkrService(
		ibkrRepo,
		repository.NewPortfolioRepository(db),
		transactionService,
		fundRepo,
	)
}

func NewTestSystemService(t *testing.T, db *sql.DB) *service.SystemService {
	t.Helper()

	return service.NewSystemService(db)
}

func NewTestDeveloperService(t *testing.T, db *sql.DB) *service.DeveloperService {
	t.Helper()

	developerRepo := repository.NewDeveloperRepository(db)
	fundRepo := repository.NewFundRepository(db)
	return service.NewDeveloperService(developerRepo, fundRepo)
}

// MakeID generates a UUID string for use in tests.
//
// Example usage:
//
//	id := testutil.MakeID()
//	// Returns: "550e8400-e29b-41d4-a716-446655440000"
func MakeID() string {
	return uuid.New().String()
}

// MakeISIN generates a realistic ISIN code for testing.
//
// Example usage:
//
//	isin := testutil.MakeISIN("US")
//	// Returns: "US1A2B3C4D5E"
func MakeISIN(prefix string) string {
	if prefix == "" {
		prefix = "US"
	}
	return prefix + randomAlphanumeric(10)
}

// MakeSymbol generates a stock ticker symbol for testing.
//
// Example usage:
//
//	symbol := testutil.MakeSymbol("AAPL")
//	// Returns: "AAPL1A2B"
func MakeSymbol(base string) string {
	if base == "" {
		base = "TEST"
	}
	return base + randomAlphanumeric(4)
}

// MakePortfolioName generates a unique portfolio name for testing.
//
// Example usage:
//
//	name := testutil.MakePortfolioName("MyPortfolio")
//	// Returns: "MyPortfolio ABC123"
func MakePortfolioName(base string) string {
	if base == "" {
		base = "Portfolio"
	}
	return base + " " + randomAlphanumeric(6)
}

// MakeFundName generates a unique fund name for testing.
//
// Example usage:
//
//	name := testutil.MakeFundName("Tech Fund")
//	// Returns: "Tech Fund XYZ789"
func MakeFundName(base string) string {
	if base == "" {
		base = "Fund"
	}
	return base + " " + randomAlphanumeric(6)
}

// MakeSymbolName generates a unique fund name for testing.
//
// Example usage:
//
//	name := testutil.MakeSymbolName("Tech Symbol")
//	// Returns: "Tech Symbol XYZ789"
func MakeSymbolName(base string) string {
	if base == "" {
		base = "Symbol"
	}
	return base + " " + randomAlphanumeric(6)
}

// randomAlphanumeric generates a random alphanumeric string of specified length.
func randomAlphanumeric(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		//nolint:gosec // G404: Using math/rand for test data generation is acceptable
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// Common test constants

var (
	// CommonCurrencies contains frequently used currency codes
	CommonCurrencies = []string{"USD", "EUR", "GBP", "JPY", "CAD", "CHF", "AUD"}

	// CommonExchanges contains frequently used stock exchanges
	CommonExchanges = []string{"NASDAQ", "NYSE", "LSE", "TSE", "XETRA", "EURONEXT"}

	// CommonCountryPrefixes contains common ISIN country prefixes
	CommonCountryPrefixes = []string{"US", "GB", "DE", "FR", "JP", "CA", "CH", "AU"}
)

// RandomCurrency returns a random currency from CommonCurrencies.
func RandomCurrency() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonCurrencies[rand.Intn(len(CommonCurrencies))]
}

// RandomExchange returns a random exchange from CommonExchanges.
func RandomExchange() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonExchanges[rand.Intn(len(CommonExchanges))]
}

// RandomCountryPrefix returns a random country prefix from CommonCountryPrefixes.
func RandomCountryPrefix() string {
	//nolint:gosec // G404: Using math/rand for test data generation is acceptable
	return CommonCountryPrefixes[rand.Intn(len(CommonCountryPrefixes))]
}
