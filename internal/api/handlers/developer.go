package handlers

import (
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

type LogLevel string

const (
	LogLevelDebug    LogLevel = "debug"
	LogLevelInfo     LogLevel = "info"
	LogLevelWarning  LogLevel = "warning"
	LogLevelError    LogLevel = "error"
	LogLevelCritical LogLevel = "critical"
)

func (l LogLevel) IsValid() bool {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarning, LogLevelError, LogLevelCritical:
		return true
	}
	return false
}

type CategoryLevel string

const (
	CategoryPortfolio   CategoryLevel = "portfolio"
	CategoryFund        CategoryLevel = "fund"
	CategoryTransaction CategoryLevel = "transaction"
	CategoryDividend    CategoryLevel = "dividend"
	CategorySystem      CategoryLevel = "system"
	CategoryDatabase    CategoryLevel = "database"
	CategorySecurity    CategoryLevel = "security"
	CategoryIbkr        CategoryLevel = "ibkr"
	CategoryDeveloper   CategoryLevel = "developer"
)

func (l CategoryLevel) IsValid() bool {
	switch l {
	case CategoryPortfolio, CategoryFund, CategoryTransaction, CategoryDividend, CategorySystem, CategoryDatabase, CategorySecurity, CategoryIbkr, CategoryDeveloper:
		return true
	}
	return false
}
