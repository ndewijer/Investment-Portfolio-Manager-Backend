package testutil

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

func NewTestPortfolioService(t *testing.T, db *sql.DB) *service.PortfolioService {
	t.Helper()

	portfolioRepo := repository.NewPortfolioRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	fundRepo := repository.NewFundRepository(db)
	dividendRepo := repository.NewDividendRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)

	return service.NewPortfolioService(
		portfolioRepo,
		transactionRepo,
		fundRepo,
		dividendRepo,
		realizedGainLossRepo,
	)
}

func init() {
	// Seed random number generator for test helpers
	rand.Seed(time.Now().UnixNano())
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

// randomAlphanumeric generates a random alphanumeric string of specified length.
func randomAlphanumeric(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// randomNumeric generates a random numeric string of specified length.
func randomNumeric(length int) string {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := range result {
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
	return CommonCurrencies[rand.Intn(len(CommonCurrencies))]
}

// RandomExchange returns a random exchange from CommonExchanges.
func RandomExchange() string {
	return CommonExchanges[rand.Intn(len(CommonExchanges))]
}

// RandomCountryPrefix returns a random country prefix from CommonCountryPrefixes.
func RandomCountryPrefix() string {
	return CommonCountryPrefixes[rand.Intn(len(CommonCountryPrefixes))]
}
