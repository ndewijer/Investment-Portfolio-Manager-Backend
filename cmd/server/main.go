package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/config"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/yahoo"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Open database connection
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	log.Printf("Connected to database: %s", cfg.Database.Path)

	systemService, portfolioService, fundService, materializedService, transactionService, ibkrService, developerService := createRepoAndServices(db)
	// Create router
	router := api.NewRouter(
		systemService,
		portfolioService,
		fundService,
		materializedService,
		transactionService,
		ibkrService,
		developerService,
		cfg,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", cfg.Server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func createRepoAndServices(db *sql.DB) (
	*service.SystemService,
	*service.PortfolioService,
	*service.FundService,
	*service.MaterializedService,
	*service.TransactionService,
	*service.IbkrService,
	*service.DeveloperService,
) {
	// Create repositories
	portfolioRepo := repository.NewPortfolioRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	fundRepo := repository.NewFundRepository(db)
	dividendRepo := repository.NewDividendRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)
	materializedRepo := repository.NewMaterializedRepository(db)
	ibkrRepo := repository.NewIbkrRepository(db)
	developerRepo := repository.NewDeveloperRepository(db)

	// Create services
	systemService := service.NewSystemService(db)

	// Create yahoo client
	yahooClient := yahoo.NewFinanceClient()

	developerService := service.NewDeveloperService(
		developerRepo,
		fundRepo,
	)

	realizedGainLossService := service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
	transactionService := service.NewTransactionService(
		db,
		transactionRepo,
		fundRepo,
	)
	dividendService := service.NewDividendService(
		db,
		dividendRepo,
		fundRepo,
		transactionRepo,
	)
	portfolioService := service.NewPortfolioService(
		db,
		portfolioRepo,
	)
	dataloaderService := service.NewDataLoaderService(
		portfolioRepo,
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService,
	)
	fundService := service.NewFundService(
		db,
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService,
		dataloaderService,
		portfolioRepo,
		yahooClient,
	)
	ibkrService := service.NewIbkrService(
		ibkrRepo,
		portfolioRepo,
		transactionService,
		fundRepo,
	)
	materializedService := service.NewMaterializedService(
		materializedRepo,
		portfolioRepo,
		fundRepo,
		transactionService,
		fundService,
		dividendService,
		realizedGainLossService,
		dataloaderService,
		portfolioService,
	)

	return systemService,
		portfolioService,
		fundService,
		materializedService,
		transactionService,
		ibkrService,
		developerService
}
