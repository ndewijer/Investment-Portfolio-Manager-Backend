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
func NewRouter(systemService *service.SystemService, portfolioService *service.PortfolioService, fundService *service.FundService, transactionService *service.TransactionService, dividendService *service.DividendService, realizedGainLossService *service.RealizedGainLossService, cfg *config.Config) http.Handler {
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
			portfolioHandler := handlers.NewPortfolioHandler(portfolioService, fundService)
			r.Get("/", portfolioHandler.Portfolios)
			r.Get("/summary", portfolioHandler.PortfolioSummary)
			r.Get("/history", portfolioHandler.PortfolioHistory)
			r.Get("/funds", portfolioHandler.PortfolioFunds)
			r.Get("/funds/{portfolioId}", portfolioHandler.GetPortfolioFunds)
			r.Get("/{portfolioId}", portfolioHandler.GetPortfolio)
		})

		r.Route("/fund", func(r chi.Router) {
			fundHandler := handlers.NewFundHandler(fundService)
			r.Get("/", fundHandler.Funds)
		})

		r.Route("/dividend", func(r chi.Router) {
			dividendHandler := handlers.NewDividendHandler(dividendService)
			r.Get("/portfolio/{portfolioId}", dividendHandler.DividendPerPortfolio)
		})
	})

	return r
}
