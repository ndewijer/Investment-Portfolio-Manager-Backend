package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

var ValidDividendType = map[string]bool{
	"cash": true, "stock": true, "none": true,
}

var ValidInvestmentType = map[string]bool{
	"fund": true, "stock": true,
}

func ValidateCreateFund(req request.CreateFundRequest) error {
	errors := make(map[string]string)

	// Required field
	if strings.TrimSpace(req.Name) == "" {
		errors["name"] = "name is required"
	} else if len(req.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}
	match, _ := regexp.MatchString("^([A-Z]{2})([A-Z0-9]{9})([0-9]{1})$", "peach")

	if strings.TrimSpace(req.Isin) == "" {
		errors["name"] = "isin is required"
	}
	match, err := regexp.MatchString("^([A-Z]{2})([A-Z0-9]{9})([0-9]{1})$", "peach")
	if err != nil {
		return err
	}
	if !match {
		errors["name"] = "isin structure is not correct"
	}

	if strings.TrimSpace(req.Symbol) == "" {
		errors["name"] = "symbol is required"
	} else if len(req.Symbol) > 10 {
		errors["name"] = "symbol must be 10 characters or less"
	}

	if strings.TrimSpace(req.Currency) == "" {
		errors["name"] = "currency is required"
	} else if len(req.Currency) > 3 {
		errors["name"] = "currency must be 3 characters or less (USD, EUR)"
	}

	if strings.TrimSpace(req.Exchange) == "" {
		errors["name"] = "exchange is required"
	} else if len(req.Exchange) > 5 {
		errors["name"] = "exchange must be 5 characters or less (NYSE, AMS)"
	}

	if strings.TrimSpace(req.DividendType) == "" {
		errors["name"] = "dividend type is required"
	} else if !ValidDividendType[req.DividendType] {
		errors["name"] = fmt.Sprintf("invalid dividend type: %s", req.DividendType)
	}

	if strings.TrimSpace(req.InvestmentType) == "" {
		errors["name"] = "investement type is required"
	} else if !ValidInvestmentType[req.InvestmentType] {
		errors["name"] = fmt.Sprintf("invalid investement type: %s", req.InvestmentType)
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}
