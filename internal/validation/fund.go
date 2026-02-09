package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

var ValidDividendType = map[string]bool{
	"CASH": true, "STOCK": true, "NONE": true,
}

var ValidInvestmentType = map[string]bool{
	"FUND": true, "STOCK": true,
}

func ValidateCreateFund(req request.CreateFundRequest) error {
	errors := make(map[string]string)

	// Required field
	if strings.TrimSpace(req.Name) == "" {
		errors["name"] = "name is required"
	} else if len(req.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}
	if strings.TrimSpace(req.Isin) == "" {
		errors["isin"] = "isin is required"
	}
	match, err := regexp.MatchString("^([A-Z]{2})([A-Z0-9]{9})([0-9]{1})$", req.Isin)
	if err != nil {
		return err
	}
	if !match {
		errors["isin"] = "isin structure is not correct"
	}

	if strings.TrimSpace(req.Currency) == "" {
		errors["currency"] = "currency is required"
	} else if len(req.Currency) > 3 {
		errors["currency"] = "currency must be 3 characters or less (USD, EUR)"
	}

	if strings.TrimSpace(req.Exchange) == "" {
		errors["exchange"] = "exchange is required"
	} else if len(req.Exchange) > 15 {
		errors["exchange"] = "exchange must be 15 characters or less (NYSE, AMS)"
	}

	if strings.TrimSpace(req.DividendType) == "" {
		errors["dividendType"] = "dividend type is required"
	} else if !ValidDividendType[req.DividendType] {
		errors["dividendType"] = fmt.Sprintf("invalid dividend type: %s", req.DividendType)
	}

	if strings.TrimSpace(req.InvestmentType) == "" {
		errors["investmentType"] = "investement type is required"
	} else if !ValidInvestmentType[req.InvestmentType] {
		errors["investmentType"] = fmt.Sprintf("invalid investement type: %s", req.InvestmentType)
	}

	// optional
	if len(req.Symbol) > 10 {
		errors["symbol"] = "symbol must be 10 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}

func ValidateUpdateFund(req request.UpdateFundRequest) error {
	errors := make(map[string]string)

	// Required field
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors["name"] = "name is required"
		} else if len(*req.Name) > 100 {
			errors["name"] = "name must be 100 characters or less"
		}
	}
	if req.Isin != nil {
		if strings.TrimSpace(*req.Isin) == "" {
			errors["isin"] = "isin is required"
		}
		match, err := regexp.MatchString("^([A-Z]{2})([A-Z0-9]{9})([0-9]{1})$", *req.Isin)
		if err != nil {
			return err
		}
		if !match {
			errors["isin"] = "isin structure is not correct"
		}
	}
	if req.Currency != nil {
		if strings.TrimSpace(*req.Currency) == "" {
			errors["currency"] = "currency is required"
		} else if len(*req.Currency) > 3 {
			errors["currency"] = "currency must be 3 characters or less (USD, EUR)"
		}
	}
	if req.Exchange != nil {
		if strings.TrimSpace(*req.Exchange) == "" {
			errors["exchange"] = "exchange is required"
		} else if len(*req.Exchange) > 15 {
			errors["exchange"] = "exchange must be 15 characters or less (NYSE, AMS)"
		}
	}
	if req.DividendType != nil {
		if strings.TrimSpace(*req.DividendType) == "" {
			errors["dividendType"] = "dividend type is required"
		} else if !ValidDividendType[*req.DividendType] {
			errors["dividendType"] = fmt.Sprintf("invalid dividend type: %s", *req.DividendType)
		}
	}
	if req.InvestmentType != nil {
		if strings.TrimSpace(*req.InvestmentType) == "" {
			errors["investmentType"] = "investement type is required"
		} else if !ValidInvestmentType[*req.InvestmentType] {
			errors["investmentType"] = fmt.Sprintf("invalid investement type: %s", *req.InvestmentType)
		}
	}
	// optional
	if req.Symbol != nil && len(*req.Symbol) > 10 {
		errors["symbol"] = "symbol must be 10 characters or less"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}
