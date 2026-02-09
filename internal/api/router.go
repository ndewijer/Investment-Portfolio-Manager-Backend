package api //nolint:revive // var-naming: api is a standard package name

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
	custommiddleware "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/middleware"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/config"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// NewRouter creates and configures the HTTP router
//
//nolint:funlen // Needs to be in the same function
func NewRouter(
	systemService *service.SystemService,
	portfolioService *service.PortfolioService,
	fundService *service.FundService,
	materializedService *service.MaterializedService,
	transactionService *service.TransactionService,
	ibkrService *service.IbkrService,
	developerService *service.DeveloperService,
	cfg *config.Config,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS middleware
	corsMiddleware := custommiddleware.NewCORS(cfg.CORS.AllowedOrigins)
	r.Use(corsMiddleware.Handler)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// System namespace
		r.Route("/system", func(r chi.Router) {
			systemHandler := handlers.NewSystemHandler(systemService)
			r.Get("/health", systemHandler.Health)
			r.Get("/version", systemHandler.Version)
		})

		r.Route("/portfolio", func(r chi.Router) {
			portfolioHandler := handlers.NewPortfolioHandler(portfolioService, fundService, materializedService)
			r.Get("/", portfolioHandler.Portfolios)
			r.Get("/summary", portfolioHandler.PortfolioSummary)
			r.Get("/history", portfolioHandler.PortfolioHistory)
			r.Get("/funds", portfolioHandler.PortfolioFunds)
			r.Post("/", portfolioHandler.CreatePortfolio)
			r.Route("/fund/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Delete("/", portfolioHandler.DeletePortfolioFund)
			})
			r.Route("/funds/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", portfolioHandler.GetPortfolioFunds)
			})
			r.Post("/funds", portfolioHandler.CreatePortfolioFund)

			r.Route("/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", portfolioHandler.GetPortfolio)
				r.Put("/", portfolioHandler.UpdatePortfolio)
				r.Delete("/", portfolioHandler.DeletePortfolio)
				r.Post("/archive", portfolioHandler.ArchivePortfolio)
				r.Post("/unarchive", portfolioHandler.UnarchivePortfolio)
			})
		})

		r.Route("/fund", func(r chi.Router) {
			fundHandler := handlers.NewFundHandler(fundService, materializedService)
			r.Get("/", fundHandler.GetAllFunds)
			r.Post("/", fundHandler.CreateFund)
			r.Get("/symbol/{symbol}", fundHandler.GetSymbol)

			r.Route("/fund-prices/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", fundHandler.GetFundPrices)
			})

			r.Route("/history/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", fundHandler.GetFundHistory)
			})

			r.Route("/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", fundHandler.GetFund)
				r.Put("/", fundHandler.UpdateFund)
				r.Get("/check-usage", fundHandler.CheckUsage)
				r.Delete("/", fundHandler.DeleteFund)
			})
		})

		r.Route("/dividend", func(r chi.Router) {
			dividendHandler := handlers.NewDividendHandler(materializedService.DividendService())
			r.Get("/", dividendHandler.GetAllDividends)

			r.Route("/portfolio/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", dividendHandler.DividendPerPortfolio)
			})
		})

		r.Route("/transaction", func(r chi.Router) {
			transactionHandler := handlers.NewTransactionHandler(transactionService)
			r.Get("/", transactionHandler.AllTransactions)

			r.Route("/portfolio/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", transactionHandler.TransactionPerPortfolio)
			})

			r.Route("/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/", transactionHandler.GetTransaction)
			})
		})

		r.Route("/ibkr", func(r chi.Router) {
			ibkrHandler := handlers.NewIbkrHandler(ibkrService)
			r.Get("/config", ibkrHandler.GetConfig)
			r.Get("/portfolios", ibkrHandler.GetActivePortfolios)
			r.Get("/dividend/pending", ibkrHandler.GetPendingDividends)
			r.Get("/inbox", ibkrHandler.GetInbox)
			r.Get("/inbox/count", ibkrHandler.GetInboxCount)

			r.Route("/inbox/{uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", func(r chi.Router) {
				r.Use(custommiddleware.ValidateUUIDMiddleware)
				r.Get("/allocations", ibkrHandler.GetTransactionAllocations)
				r.Get("/eligible-portfolios", ibkrHandler.GetEligiblePortfolios)
			})
		})

		r.Route("/developer", func(r chi.Router) {
			developerHandler := handlers.NewDeveloperHandler(developerService)
			r.Get("/logs", developerHandler.GetLogs)
			r.Get("/system-settings/logging", developerHandler.GetLoggingConfig)
			r.Get("/csv/fund-prices/template", developerHandler.GetFundPriceCSVTemplate)
			r.Get("/csv/transactions/template", developerHandler.GetTransactionCSVTemplate)
			r.Get("/exchange-rate", developerHandler.GetExchangeRate)
			r.Get("/fund-price", developerHandler.GetFundPrice)
		})
	})

	return r
}
