package validation

import (
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

type Error struct {
	Fields map[string]string
}

func (e *Error) Error() string {
	msgs := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(msgs, "; ")
}

func ValidateCreatePortfolio(req request.CreatePortfolioRequest) error {
	errors := make(map[string]string)

	// Required field
	if strings.TrimSpace(req.Name) == "" {
		errors["name"] = "name is required"
	} else if len(req.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}

	// Optional but has constraints
	if len(req.Description) > 500 {
		errors["description"] = "description must be 500 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

func ValidateUpdatePortfolio(req request.UpdatePortfolioRequest) error {
	errors := make(map[string]string)

	// Only validate provided fields
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors["name"] = "name cannot be empty"
		} else if len(*req.Name) > 100 {
			errors["name"] = "name must be 100 characters or less"
		}
	}

	if req.Description != nil && len(*req.Description) > 500 {
		errors["description"] = "description must be 500 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

func ValidateCreatePortfolioFund(req request.CreatePortfolioFundRequest) error {
	fundErr := ValidateUUID(req.FundID)
	if fundErr != nil {
		return fundErr
	}
	portfolioErr := ValidateUUID(req.PortfolioID)
	if portfolioErr != nil {
		return portfolioErr
	}
	return nil
}
