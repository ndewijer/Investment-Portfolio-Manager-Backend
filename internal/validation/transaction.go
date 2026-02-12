package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

// ValidTransactionType contains the allowed transaction type values.
var ValidTransactionType = map[string]bool{
	"buy": true, "sell": true, "dividend": true, "fee": true,
}

// ValidateCreateFund validates a fund creation request.
// Checks all required fields and validates their formats and constraints.
//
// Required fields:
//   - name: Must be non-empty and max 100 characters
//   - isin: Must be non-empty and match format: 2 letters + 9 alphanumeric + 1 digit
//   - currency: Must be non-empty and max 3 characters (e.g., USD, EUR)
//   - exchange: Must be non-empty and max 15 characters (e.g., NYSE, AMS)
//   - dividend_type: Must be one of: CASH, STOCK, NONE
//   - investment_type: Must be one of: FUND, STOCK
//
// Optional fields:
//   - symbol: Max 10 characters if provided
//
// Returns a validation Error with field-specific error messages if validation fails.
//
//nolint:gocyclo // Comprehensive validation of fund creation, cannot be split well.
func ValidateCreateTransaction(req request.CreateTransactionRequest) error {
	errors := make(map[string]string)

	portfolioFundErr := ValidateUUID(req.PortfolioFundID)
	if portfolioFundErr != nil {
		return portfolioFundErr
	}

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	}
	_, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		errors["date"] = err.Error()
	}

	if strings.TrimSpace(req.Type) == "" {
		errors["transactionType"] = "type is required"
	} else if !ValidTransactionType[req.Type] {
		errors["transactionType"] = fmt.Sprintf("invalid type: %s", req.Type)
	}

	if req.Shares == 0.0 {
		errors["shares"] = "shares is required"
	}

	if req.CostPerShare == 0.0 {
		errors["costPerShare"] = "costPerShare is required"
	}

	return nil
}

// ValidateUpdateFund validates a fund update request.
// All fields are optional, but if provided, they must meet the same constraints as create.
//
// Optional fields (validated if provided):
//   - name: Max 100 characters if provided
//   - isin: Must match format: 2 letters + 9 alphanumeric + 1 digit if provided
//   - currency: Max 3 characters if provided (e.g., USD, EUR)
//   - exchange: Max 15 characters if provided (e.g., NYSE, AMS)
//   - dividend_type: Must be one of: CASH, STOCK, NONE if provided
//   - investment_type: Must be one of: FUND, STOCK if provided
//   - symbol: Max 10 characters if provided
//
// Returns a validation Error with field-specific error messages if validation fails.

//nolint:gocyclo // Comprehensive validation of fund updates, cannot be split well.
func ValidateUpdateTransaction(req request.UpdateTransactionRequest) error {
	errors := make(map[string]string)

	if req.PortfolioFundID != nil {
		portfolioFundErr := ValidateUUID(*req.PortfolioFundID)
		if portfolioFundErr != nil {
			return portfolioFundErr
		}
	}
	if req.Date != nil {
		if strings.TrimSpace(*req.Date) == "" {
			errors["date"] = "date is required"
		}
		_, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			errors["date"] = err.Error()
		}
	}
	if req.Type != nil {
		if strings.TrimSpace(*req.Type) == "" {
			errors["transactionType"] = "type is required"
		} else if !ValidTransactionType[*req.Type] {
			errors["transactionType"] = fmt.Sprintf("invalid type: %s", *req.Type)
		}
	}
	if req.Shares != nil {
		if *req.Shares == 0.0 {
			errors["shares"] = "shares is required"
		}
	}
	if req.CostPerShare != nil {
		if *req.CostPerShare == 0.0 {
			errors["costPerShare"] = "costPerShare is required"
		}
	}

	return nil
}
