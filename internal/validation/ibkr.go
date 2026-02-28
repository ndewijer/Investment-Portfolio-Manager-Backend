package validation

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

//nolint:gocyclo // Comprehensive validation of ibkr config updates, cannot be split well.
func ValidateUpdateIbkrConfig(req request.UpdateIbkrConfigRequest) error {
	errors := make(map[string]string)

	// Required field
	if req.Enabled != nil && *req.Enabled {
		if req.FlexToken != nil {
			if strings.TrimSpace(*req.FlexToken) != "" {
				if len(*req.FlexToken) != 25 {
					errors["flexToken"] = "flexToken must be 25 characters"
				} else if _, err := strconv.Atoi(strings.TrimSpace(*req.FlexToken)); err != nil {
					errors["flexToken"] = "flexToken must be a number"
				}
			}
		}

		if req.FlexQueryID != nil {
			if *req.FlexQueryID == "" {
				errors["flexQueryId"] = "flexQueryId must be set"
			} else if len(*req.FlexQueryID) > 10 {
				errors["flexQueryId"] = "flexQueryId must be 10 characters or less"
			} else if _, err := strconv.Atoi(strings.TrimSpace(*req.FlexQueryID)); err != nil {
				errors["flexQueryId"] = "flexQueryId must be a number"
			}

		}

		if req.TokenExpiresAt != nil {
			time, err := ParseTime(*req.TokenExpiresAt)
			if err != nil {
				errors["tokenExpiresAt"] = fmt.Sprintf("tokenExpiresAt cannot be parsed: %v", err)
			} else if time.IsZero() {
				errors["tokenExpiresAt"] = "tokenExpiresAt must be set"
			}
		}

		if req.DefaultAllocationEnabled != nil && *req.DefaultAllocationEnabled {
			if len(req.DefaultAllocations) == 0 {
				errors["defaultAllocations"] = "defaultAllocations should have at least 1 portfolio when enabled"
			} else {
				var perc float64
				for _, v := range req.DefaultAllocations {
					perc += *v.Percentage
					if v.Percentage == nil || *v.Percentage == 0.0 {
						errors["defaultAllocations"] = "allocation percentage must be set"
					} else if v.PortfolioID == nil || *v.PortfolioID == "" {
						errors["defaultAllocations"] = "portfolio must be set"
					}
				}
				if math.Abs(perc-100) > 0.01 {
					errors["defaultAllocations"] = "defaultAllocations do not add up to 100%"
				}
			}
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}
