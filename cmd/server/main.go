package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/config"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/ibkr"
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

	// Resolve encryption key (env → file → auto-generate)
	dataDir := filepath.Dir(cfg.Database.Path)
	encKeyStr, err := resolveEncryptionKey(cfg.EncryptionKey, dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve encryption key: %v", err)
	}

	var fernetKey *fernet.Key
	if encKeyStr != "" {
		fernetKey, err = fernet.DecodeKey(encKeyStr)
		if err != nil {
			log.Fatalf("Invalid IBKR_ENCRYPTION_KEY: %v", err)
		}
	}

	systemService, portfolioService, fundService, materializedService, transactionService, ibkrService, developerService := createRepoAndServices(db, fernetKey)
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

// resolveEncryptionKey returns the encryption key string from env, file, or auto-generation.
// Priority: env var > file > generate-and-write.
func resolveEncryptionKey(cfgKey, dataDir string) (string, error) {
	// 1. Env var already loaded into cfgKey
	if cfgKey != "" {
		return cfgKey, nil
	}

	// 2. Try file
	keyPath := filepath.Join(dataDir, ".ibkr_encryption_key")
	data, err := os.ReadFile(keyPath)
	if err == nil {
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, nil
		}
	}

	// 3. Generate new key and write to file
	var k fernet.Key
	if err := k.Generate(); err != nil {
		return "", fmt.Errorf("generate encryption key: %w", err)
	}
	encoded := k.Encode()

	if err := os.WriteFile(keyPath, []byte(encoded+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write encryption key to %s: %w", keyPath, err)
	}
	log.Printf("Generated new IBKR encryption key → %s", keyPath)

	return encoded, nil
}

func createRepoAndServices(db *sql.DB, fernetKey *fernet.Key) (
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
	pfRepo := repository.NewPortfolioFundRepository(db)
	dividendRepo := repository.NewDividendRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)
	materializedRepo := repository.NewMaterializedRepository(db)
	ibkrRepo := repository.NewIbkrRepository(db)
	developerRepo := repository.NewDeveloperRepository(db)

	// Create services
	systemService := service.NewSystemService(db)

	// Create web clients
	yahooClient := yahoo.NewFinanceClient()
	ibkrClient := ibkr.NewFinanceClient()

	developerService := service.NewDeveloperService(
		db,
		developerRepo,
		fundRepo,
		transactionRepo,
		pfRepo,
	)

	realizedGainLossService := service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
	transactionService := service.NewTransactionService(
		db,
		transactionRepo,
		pfRepo,
	)
	dividendService := service.NewDividendService(
		db,
		dividendRepo,
		pfRepo,
		transactionRepo,
	)
	portfolioService := service.NewPortfolioService(
		db,
		portfolioRepo,
		pfRepo,
	)
	dataloaderService := service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
		service.DataLoaderWithFundRepository(fundRepo),
		service.DataLoaderWithTransactionService(transactionService),
		service.DataLoaderWithDividendService(dividendService),
		service.DataLoaderWithRealizedGainLossService(realizedGainLossService),
	)
	fundService := service.NewFundService(
		db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithTransactionService(transactionService),
		service.FundWithDividendService(dividendService),
		service.FundWithRealizedGainLossService(realizedGainLossService),
		service.FundWithDataLoaderService(dataloaderService),
		service.FundWithPortfolioRepo(portfolioRepo),
		service.FundWithYahooClient(yahooClient),
	)
	ibkrService := service.NewIbkrService(
		db,
		service.IbkrWithIbkrRepo(ibkrRepo),
		service.IbkrWithPortfolioRepo(portfolioRepo),
		service.IbkrWithFundRepo(fundRepo),
		service.IbkrWithDeveloperRepo(developerRepo),
		service.IbkrWithClient(ibkrClient),
		service.IbkrWithPortfolioFundRepo(pfRepo),
		service.IbkrWithTransactionRepo(transactionRepo),
		service.IbkrWithDividendRepo(dividendRepo),
		service.IbkrWithEncryptionKey(fernetKey),
	)
	materializedService := service.NewMaterializedService(
		service.MaterializedWithMaterializedRepository(materializedRepo),
		service.MaterializedWithPortfolioRepository(portfolioRepo),
		service.MaterializedWithFundRepository(fundRepo),
		service.MaterializedWithTransactionService(transactionService),
		service.MaterializedWithFundService(fundService),
		service.MaterializedWithDividendService(dividendService),
		service.MaterializedWithRealizedGainLossService(realizedGainLossService),
		service.MaterializedWithDataLoaderService(dataloaderService),
		service.MaterializedWithPortfolioService(portfolioService),
	)

	return systemService,
		portfolioService,
		fundService,
		materializedService,
		transactionService,
		ibkrService,
		developerService
}
