package errors

import "errors"

// Domain entity errors represent missing or invalid entities in the system.
// These errors indicate that a requested resource does not exist.
var (
	// ErrPortfolioNotFound indicates that a portfolio with the given ID does not exist.
	ErrPortfolioNotFound = errors.New("portfolio not found")

	// ErrFundNotFound indicates that a fund with the given ID does not exist.
	ErrFundNotFound = errors.New("fund not found")

	// ErrTransactionNotFound indicates that a transaction with the given ID does not exist.
	ErrTransactionNotFound = errors.New("transaction not found")

	// ErrDividendNotFound indicates that a dividend record with the given ID does not exist.
	ErrDividendNotFound = errors.New("dividend not found")

	// ErrRealizedGainLossNotFound indicates that a realized gain/loss record does not exist.
	ErrRealizedGainLossNotFound = errors.New("realized gain/loss not found")

	// ErrPortfolioFundNotFound indicates that a portfolio-fund relationship does not exist.
	ErrPortfolioFundNotFound = errors.New("portfolio-fund relationship not found")

	ErrIBKRTransactionNotFound = errors.New("ibkr transaction not found")
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
)

// Data integrity errors represent inconsistencies or corruption in the data.
var (
	// ErrDataInconsistency indicates that the data is in an inconsistent state
	// (e.g., a portfolio-fund relationship exists but the fund doesn't exist).
	ErrDataInconsistency = errors.New("data inconsistency detected")

	// ErrMissingRequiredField indicates that a required field is missing or empty.
	ErrMissingRequiredField = errors.New("missing required field")
)
