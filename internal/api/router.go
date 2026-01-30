package api

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
func NewRouter(
	systemService *service.SystemService,
	portfolioService *service.PortfolioService,
	fundService *service.FundService,
	materializedService *service.MaterializedService,
	transactionService *service.TransactionService,
	ibkrService *service.IbkrService,
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
			r.Get("/funds/{portfolioId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", portfolioHandler.GetPortfolioFunds)
			r.Get("/{portfolioId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", portfolioHandler.GetPortfolio)
		})

		r.Route("/fund", func(r chi.Router) {
			fundHandler := handlers.NewFundHandler(fundService, materializedService)
			r.Get("/", fundHandler.GetAllFunds)
			r.Get("/fund-prices/{fundId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", fundHandler.GetFundPrices)
			r.Get("/symbol/{symbol}", fundHandler.GetSymbol)
			r.Get("/history/{portfolioId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", fundHandler.GetFundHistory)
			r.Get("/{fundId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", fundHandler.GetFund)
		})

		r.Route("/dividend", func(r chi.Router) {
			dividendHandler := handlers.NewDividendHandler(materializedService.DividendService())
			r.Get("/", dividendHandler.GetAllDividends)
			r.Get("/portfolio/{portfolioId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", dividendHandler.DividendPerPortfolio)
		})

		r.Route("/transaction", func(r chi.Router) {
			transactionHandler := handlers.NewTransactionHandler(transactionService)
			r.Get("/", transactionHandler.AllTransactions)
			r.Get("/portfolio/{portfolioId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", transactionHandler.TransactionPerPortfolio)
			r.Get("/{transactionId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}", transactionHandler.GetTransaction)
		})

		r.Route("/ibkr", func(r chi.Router) {
			ibkrHandler := handlers.NewIbkrHandler(ibkrService)
			r.Get("/config", ibkrHandler.GetConfig)
			r.Get("/portfolios", ibkrHandler.GetActivePortfolios)
			r.Get("/dividend/pending", ibkrHandler.GetPendingDividends)
			r.Get("/inbox", ibkrHandler.GetInbox)
			r.Get("/inbox/count", ibkrHandler.GetInboxCount)
			r.Get("/inbox/{transactionId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/allocations", ibkrHandler.GetTransactionAllocations)
			r.Get("/inbox/{transactionId:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}/eligible-portfolios", ibkrHandler.GetEligiblePortfolios)
		})
	})

	return r
}
