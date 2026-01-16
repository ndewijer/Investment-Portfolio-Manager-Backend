package main

import (
	"context"
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

	// Create repositories
	portfolioRepo := repository.NewPortfolioRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	fundRepo := repository.NewFundRepository(db)
	dividendRepo := repository.NewDividendRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)
	materializedRepo := repository.NewMaterializedRepository(db)

	systemService := service.NewSystemService(db)
	// Create services
	realizedGainLossService := service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
	transactionService := service.NewTransactionService(
		transactionRepo,
	)
	dividendService := service.NewDividendService(
		dividendRepo,
	)
	fundService := service.NewFundService(
		fundRepo,
		transactionService,
		dividendService,
		realizedGainLossService,
	)
	portfolioService := service.NewPortfolioService(
		portfolioRepo,
		dividendService,
		materializedRepo,
		transactionService,
		fundService,
		realizedGainLossService,
	)

	// Create router
	router := api.NewRouter(systemService, portfolioService, fundService, transactionService, dividendService, realizedGainLossService, cfg)

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
