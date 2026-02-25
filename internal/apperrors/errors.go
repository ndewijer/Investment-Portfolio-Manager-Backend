package apperrors

import "errors"

// Domain entity errors represent missing or invalid entities in the system.
// These errors indicate that a requested resource does not exist.
var (
	// ErrPortfolioNotFound indicates that a portfolio with the given ID does not exist.
	ErrPortfolioNotFound = errors.New("portfolio not found")

	// ErrFundNotFound indicates that a fund with the given ID does not exist.
	ErrFundNotFound = errors.New("fund not found")

	// ErrFundPriceNotFound indicates no record for a specific fund and date combination.
	ErrFundPriceNotFound = errors.New("fund price not found")

	// ErrTransactionNotFound indicates that a transaction with the given ID does not exist.
	ErrTransactionNotFound = errors.New("transaction not found")

	// ErrDividendNotFound indicates that a dividend record with the given ID does not exist.
	ErrDividendNotFound = errors.New("dividend not found")

	// ErrRealizedGainLossNotFound indicates that a realized gain/loss record does not exist.
	ErrRealizedGainLossNotFound = errors.New("realized gain/loss not found")

	// ErrPortfolioFundNotFound indicates that a portfolio-fund relationship does not exist.
	ErrPortfolioFundNotFound = errors.New("portfolio-fund relationship not found")

	ErrIBKRTransactionNotFound = errors.New("ibkr transaction not found")

	// ErrSymbolNotFound indicates that a symbol lookup returned no results
	ErrSymbolNotFound = errors.New("symbol not found")

	// ErrIbkrConfigNotFound indicates IBKR configuration has not been set up
	ErrIbkrConfigNotFound = errors.New("ibkr configuration not found")

	// ErrExchangeRateNotFound indicates no record for a specific currency and date combination
	ErrExchangeRateNotFound = errors.New("exchange rate for currency/date not found")
)

// Business logic errors represent validation failures or constraint violations.
// These errors indicate that an operation cannot be completed due to business rules.
var (
	// ErrInsufficientShares indicates that a sell transaction cannot be completed
	// because the portfolio does not hold enough shares of the fund.
	ErrInsufficientShares = errors.New("insufficient shares for sale")

	// ErrInvalidDateRange indicates that the provided date range is invalid
	// (e.g., start date is after end date).
	ErrInvalidDateRange = errors.New("invalid date range")

	// ErrInvalidUUID indicates that a provided ID is not a valid UUID format.
	ErrInvalidUUID = errors.New("invalid UUID format")

	// ErrEmptyID indicates that a required ID parameter is empty or missing.
	ErrEmptyID = errors.New("ID cannot be empty")

	// ErrNegativeAmount indicates that an amount field has an invalid negative value.
	ErrNegativeAmount = errors.New("amount cannot be negative")

	// ErrDuplicateEntry indicates that an entity with the same unique constraint already exists.
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrFundInUse indicates that a fund cannot be deleted because it is being used by portfolios.
	ErrFundInUse = errors.New("fund is in use")

	// Validation errors for required fields
	ErrInvalidPortfolioID   = errors.New("portfolio ID is required")
	ErrInvalidFundID        = errors.New("fund ID is required")
	ErrInvalidSymbol        = errors.New("symbol is required")
	ErrInvalidTransactionID = errors.New("transaction ID is required")
	ErrInvalidCurrency      = errors.New("currency parameter is required")
	ErrInvalidDate          = errors.New("date parameter is required")

	// Generic operation failure constants
	ErrFailedToRetrieve = errors.New("failed to retrieve data")
)

// Operation failure errors represent system-level failures when retrieving or processing data.
// These errors indicate that an operation failed, but not due to missing entities or validation issues.
var (
	// Dividend operation errors
	ErrFailedToRetrieveDividends       = errors.New("failed to retrieve dividends")
	ErrFailedToRetrievePendingDividend = errors.New("failed to retrieve pending dividend")

	// Fund operation errors
	ErrFailedToRetrieveFunds       = errors.New("failed to retrieve funds")
	ErrFailedToRetrieveFund        = errors.New("failed to retrieve fund")
	ErrFailedToRetrieveFundHistory = errors.New("failed to retrieve fund history")
	ErrFailedToRetrieveSymbol      = errors.New("failed to retrieve symbol")
	ErrFailedToRetrieveUsage       = errors.New("failed to retrieve fund usage")

	// Portfolio operation errors
	ErrFailedToRetrievePortfolios     = errors.New("failed to retrieve portfolios")
	ErrFailedToRetrievePortfolioFunds = errors.New("failed to retrieve portfolio funds")
	ErrFailedToGetPortfolioSummary    = errors.New("failed to get portfolio summary")
	ErrFailedToGetPortfolioHistory    = errors.New("failed to get portfolio history")
	ErrFailedToGetPortfolioFunds      = errors.New("failed to get portfolio funds")

	// Transaction operation errors
	ErrFailedToRetrieveTransactions = errors.New("failed to retrieve transactions")
	ErrFailedToRetrieveTransaction  = errors.New("failed to retrieve transaction")

	// IBKR operation errors
	ErrFailedToRetrieveIbkrConfig        = errors.New("failed to retrieve ibkr config")
	ErrFailedToRetrieveInboxTransactions = errors.New("failed to retrieve inbox transactions")
	ErrFailedToGetTransactionAllocations = errors.New("failed to get transaction allocations")
	ErrFailedToGetEligiblePortfolios     = errors.New("failed to get eligible portfolios")
	ErrFailedToGetNewFlexRapport         = errors.New("failed to get new flex rapport")

	// System operation errors
	ErrFailedToGetVersionInfo = errors.New("failed to get version information")

	// Developer operation errors
	ErrFailedToRetrieveLogs          = errors.New("failed to retrieve logs")
	ErrFailedToRetrieveLoggingConfig = errors.New("failed to retrieve logging configuration")
	ErrFailedToRetrieveExchangeRate  = errors.New("failed to retrieve exchange rate")
	ErrFailedToRetrieveFundPrice     = errors.New("failed to retrieve fund price")
	ErrFailedToSetLoggingConfig      = errors.New("failed to set logging configuration")
	ErrFailedToUpdateExchangeRate    = errors.New("failed to update exchange rate")
	ErrFailedToUpdateFundPrice       = errors.New("failed to update fund price")
	ErrFailedToDeleteLogs            = errors.New("failed to delete logs")
	ErrFailedToImportFundPrices      = errors.New("failed to import fund prices")
	ErrFailedToImportTransactions    = errors.New("failed to import transactions")
	ErrInvalidCSVHeaders             = errors.New("invalid CSV headers")
)

// Data integrity errors represent inconsistencies or corruption in the data.
var (
	// ErrDataInconsistency indicates that the data is in an inconsistent state
	// (e.g., a portfolio-fund relationship exists but the fund doesn't exist).
	ErrDataInconsistency = errors.New("data inconsistency detected")

	// ErrMissingRequiredField indicates that a required field is missing or empty.
	ErrMissingRequiredField = errors.New("missing required field")
)
