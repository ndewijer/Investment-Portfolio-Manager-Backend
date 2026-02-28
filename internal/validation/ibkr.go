package validation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
)

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
				errors["flexQueryId"] = "flexToken must be 10 characters or less"
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
			// var allocReq request.Allocation
			// err := json.Unmarshal(req.DefaultAllocations, &allocReq)
			// if err != nil {
			// 	errors["defaultAllocations"] = fmt.Sprintf("DefaultAllocations could not be unmarshaled: %v", err)
			// }
		}
	}

	if len(errors) > 0 {
		return &Error{Fields: errors}
	}
	return nil
}
