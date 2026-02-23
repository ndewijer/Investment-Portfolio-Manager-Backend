package validation

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

var ValidLogLevels = map[string]bool{
	"debug": true, "info": true, "warning": true, "error": true, "critical": true,
}

func ValidateUpdateExchangeRate(req request.SetExchangeRateRequest) error {
	errors := make(map[string]string)

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	}
	_, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		errors["date"] = err.Error()
	}

	if strings.TrimSpace(req.ToCurrency) == "" {
		errors["toCurrency"] = "to currency is required"
	}

	if strings.TrimSpace(req.FromCurrency) == "" {
		errors["fromCurrency"] = "from currency is required"
	}

	if strings.TrimSpace(req.Rate) == "" {
		errors["rate"] = "rate is required"
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(req.Rate), 64); err != nil {
		errors["rate"] = "rate not a valid number"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}

func ValidateUpdateFundPrice(req request.SetFundPriceRequest) error {
	errors := make(map[string]string)

	if strings.TrimSpace(req.Date) == "" {
		errors["date"] = "date is required"
	}
	_, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		errors["date"] = err.Error()
	}

	if strings.TrimSpace(req.FundID) == "" {
		errors["fundID"] = "fundID is required"
	}

	if strings.TrimSpace(req.Price) == "" {
		errors["price"] = "price is required"
	}
	if _, err := strconv.ParseFloat(strings.TrimSpace(req.Price), 64); err != nil {
		errors["price"] = "price not a valid number"
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}

func ValidateLoggingConfig(req request.SetLoggingConfig) error {
	errors := make(map[string]string)

	if req.Enabled == nil {
		errors["enabled"] = "enabled is required"
	}

	if strings.TrimSpace(req.Level) == "" {
		errors["level"] = "level is required"
	} else if !ValidLogLevels[req.Level] {
		errors["level"] = fmt.Sprintf("invalid level: %s", req.Level)
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}

	return nil

}
