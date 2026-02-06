package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// DeveloperHandler handles HTTP requests for Developer endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the DeveloperService.
type DeveloperHandler struct {
	DeveloperService *service.DeveloperService
}

// NewDeveloperHandler creates a new DeveloperHandler with the provided service dependency.
func NewDeveloperHandler(DeveloperService *service.DeveloperService) *DeveloperHandler {
	return &DeveloperHandler{
		DeveloperService: DeveloperService,
	}
}

type TemplateModel struct {
	Headers     []string          `json:"headers"`
	Example     map[string]string `json:"example"`
	Description string            `json:"description"`
}

func (h *DeveloperHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse filter parameters
	filters, err := request.ParseLogFilters(
		r.URL.Query().Get("level"),
		r.URL.Query().Get("category"),
		r.URL.Query().Get("startDate"),
		r.URL.Query().Get("endDate"),
		r.URL.Query().Get("source"),
		r.URL.Query().Get("message"),
		r.URL.Query().Get("sortDir"),
		r.URL.Query().Get("cursor"),
		r.URL.Query().Get("perPage"),
	)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Call service with filters
	logs, err := h.DeveloperService.GetLogs(r.Context(), filters)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "Failed to retrieve logs", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, logs)
}

func (h *DeveloperHandler) GetLoggingConfig(w http.ResponseWriter, _ *http.Request) {
	setting, err := h.DeveloperService.GetLoggingConfig()
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "Failed to retreived log settings", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, setting)
}

func (h *DeveloperHandler) GetFundPriceCSVTemplate(w http.ResponseWriter, _ *http.Request) {

	headers := []string{"date", "price"}
	example := map[string]string{
		"date":  "2024-03-21",
		"price": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Price date in YYYY-MM-DD format
- price: Fund price (decimal numbers)`

	template := TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}

func (h *DeveloperHandler) GetTransactionCSVTemplate(w http.ResponseWriter, _ *http.Request) {

	headers := []string{"date", "type", "shares", "cost_per_share"}
	example := map[string]string{
		"date":           "2024-03-21",
		"type":           "buy/sell",
		"shares":         "10.5",
		"cost_per_share": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Transaction date in YYYY-MM-DD format
- type: Transaction type, either "buy" or "sell"
- shares: Number of shares (decimal numbers allowed)
- cost_per_share: Cost per share in the fund's currency`

	template := TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}
