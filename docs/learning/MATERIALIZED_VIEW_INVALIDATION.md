# Materialized View Invalidation & Regeneration (Issue #35)

## Overview

The materialized view system (`fund_history_materialized` table) caches pre-calculated daily
portfolio/fund metrics. When source data changes (transactions, prices, dividends), the cache
must be invalidated and regenerated to avoid serving stale data.

## Architecture

### Two-Path Design

**Path 1: Write-path hooks** — After any write operation commits, an async goroutine
triggers `RegenerateMaterializedTable()` to rebuild affected cache entries.

**Path 2: Read-path fallback** — `GetPortfolioHistoryWithFallback()` and
`GetFundHistoryWithFallback()` detect stale/empty cache via `checkStaleData()`, return
on-demand calculations immediately, and trigger background regeneration for next time.

### Circular Dependency Resolution

The `MaterializedInvalidator` interface breaks the import cycle between services and
`MaterializedService`. All write-path services depend on the interface, not the concrete
type. The concrete `MaterializedService` is injected in `cmd/server/main.go` after all
services are constructed:

```go
fundService.SetMaterializedInvalidator(materializedService)
transactionService.SetMaterializedInvalidator(materializedService)
dividendService.SetMaterializedInvalidator(materializedService)
ibkrService.SetMaterializedInvalidator(materializedService)
developerService.SetMaterializedInvalidator(materializedService)
```

### Duplicate Prevention & Supersede Logic

`MaterializedService` uses a `sync.Mutex`-protected `map[string]time.Time` (`regenInFlight`)
to track which portfolio IDs have a background regeneration job running and from which date.

When a new regen request arrives for a portfolio:
- **No job running** → start one immediately
- **Job running, new request has earlier startDate** → update the tracked date; the running
  job will finish, then a follow-up job starts from the earlier date
- **Job running, new request has later/equal startDate** → drop it (already covered)

### Write Serialization

A `regenWriteMu sync.Mutex` serializes all background `RegenerateMaterializedTable` writes.
SQLite only supports one writer at a time; without this, concurrent API requests each
launching regen goroutines cause `SQLITE_BUSY`. WAL mode + `busy_timeout = 5000` provide
a baseline, but the mutex prevents all contention.

### Nil Guard Pattern

All invalidator calls are wrapped with `if s.materializedInvalidator != nil` so that
tests (which don't wire up the full dependency graph) don't panic.

## Write-Path Coverage

Every service that modifies data affecting the materialized view triggers regeneration
after its DB transaction commits. There are **16 invalidation points** across 5 services:

| Service | Method | Regen Scope | Edge Case |
|---------|--------|-------------|-----------|
| `TransactionService` | `CreateTransaction` | portfolioFundID | #1 |
| `TransactionService` | `UpdateTransaction` | portfolioFundID | #1 |
| `TransactionService` | `DeleteTransaction` | portfolioFundID | #1 |
| `FundService` | `UpdateCurrentFundPrice` | fundID (all portfolios) | #2 |
| `FundService` | `UpdateHistoricalFundPrice` | fundID, from earliest date | #6 |
| `DividendService` | `CreateDividend` | portfolioFundID, from ex-div date | #3 |
| `DividendService` | `UpdateDividend` | portfolioFundID, from min(old, new) date | #5 |
| `DividendService` | `DeleteDividend` | portfolioFundID, from ex-div date | #3 |
| `IbkrService` | `AllocateIbkrTransaction` | portfolioIDs from allocations | — |
| `IbkrService` | `UnallocateIbkrTransaction` | portfolioIDs (collected pre-delete) | — |
| `IbkrService` | `ModifyAllocations` | portfolioIDs from old + new allocations | — |
| `IbkrService` | `MatchDividend` | portfolioIDs from allocations | — |
| `DeveloperService` | `UpdateFundPrice` | fundID | #2 |
| `DeveloperService` | `ImportFundPrices` (CSV) | fundID, from earliest date | #9 |
| `DeveloperService` | `ImportTransactions` (CSV) | portfolioFundID, from earliest date | #8 |

### IBKR Regen Pattern

IBKR operations resolve affected portfolios by looking up the `ibkr_transaction_allocation`
records, which contain `portfolio_id` directly. This avoids the fragile ISIN/Symbol-based
fund lookup and uses the same IDs that were involved in the allocation.

For `UnallocateIbkrTransaction` and `ModifyAllocations`, the portfolio IDs are collected
*before* the unallocation deletes the allocation records.

## Read-Path Stale Detection

`checkStaleData()` checks all three data sources (per Issue #35 requirements):

1. **Date coverage** — Does materialized max date reach the requested `endDate`?
2. **Transaction freshness** — `MAX(transaction.created_at)` vs materialized `MAX(calculated_at)`
3. **Price freshness** — `MAX(fund_price.date)` vs materialized max date
4. **Dividend freshness** — `MAX(dividend.created_at)` vs materialized `MAX(calculated_at)`

The three source dates are fetched in a single SQL query with subselects.

Note: `fund_price` has no `created_at` column, so we compare the latest price date against
the materialized date coverage instead. This handles the "nightly price update" scenario
(Issue #35 Edge Case 2) where prices are added without new transactions.

## RegenerateMaterializedTable

Core regeneration method that:

1. Resolves which portfolios to regenerate based on `portfolioID`, `fundID`, or `portfolioFundID`
2. Calculates fund history on-the-fly (read-heavy, done outside DB transaction)
3. Collects the unique `portfolio_fund_id` values from the calculated entries
4. Opens a short write transaction to invalidate + insert atomically
5. Uses `materializedRepo.WithTx(tx)` for both operations to ensure transactional consistency

### Scoped Invalidation

`InvalidateMaterializedTable` deletes only the rows matching the specific
`portfolio_fund_id` values being regenerated, from the start date forward. This ensures
that regenerating Portfolio A does not delete cached data for Portfolio B.

### Parameter Resolution

- `portfolioID` → regenerates that single portfolio
- `fundID` → finds all portfolios holding that fund, regenerates each (Edge Case 4)
- `portfolioFundID` → looks up the portfolio ID, regenerates that portfolio

## Issue #35 Coverage Matrix

### Edge Cases

| # | Edge Case | Implementation | Write-Path Hook | Stale Detection | Test Coverage |
|---|-----------|----------------|-----------------|-----------------|---------------|
| 1 | Backdated transactions | Stale detection sees newer `created_at` | `TransactionService` CRUD | `transaction.created_at` > `calculated_at` | Stale detection test + write-path hook tests (Create/Update/Delete) |
| 2 | Price updates without transactions | Stale detection compares price date vs materialized date | `FundService.UpdateCurrentFundPrice`, `DeveloperService.UpdateFundPrice` | `fund_price.date` > materialized max date | Stale detection test + `UpdateFundPrice` hook test |
| 3 | Dividend recording without transactions | Stale detection checks `dividend.created_at` | `DividendService.CreateDividend` | `dividend.created_at` > `calculated_at` | Stale detection test + write-path hook test |
| 4 | Multi-portfolio price updates | fundID-based regen resolves all portfolios via `GetPortfolioFundsbyFundID` | `FundService` (fundID param) | — | `RegenerateMaterializedTable` by fundID test |
| 5 | Dividend date changes (old+new) | `UpdateDividend` uses `min(oldExDivDate, newExDivDate)` | `DividendService.UpdateDividend` | — | Two write-path hook tests (new earlier + new later) |
| 6 | Historical price backfills | `UpdateHistoricalFundPrice` uses earliest missing price date | `FundService.UpdateHistoricalFundPrice` | — | Requires Yahoo mock (see Remaining Gaps) |
| 7 | Sell transactions with realized gains | Sell creates RGL inside same tx; invalidation after commit | `TransactionService` CRUD | `transaction.created_at` | Full: RGL creation, update lifecycle, delete cleanup, insufficient shares |
| 8 | CSV transaction import | `ImportTransactions` triggers regen from earliest date | `DeveloperService.ImportTransactions` | — | Write-path hook test (verifies earliest date) |
| 9 | CSV price import | `ImportFundPrices` triggers regen from earliest date | `DeveloperService.ImportFundPrices` | — | Write-path hook test (verifies earliest date + fundID) |

### Complete Data Change Matrix (from Issue #35)

| Data Change | Invalidation Hook | Regen Parameters | Tested |
|-------------|-------------------|------------------|--------|
| Transaction CRUD | `TransactionService` Create/Update/Delete | `(txDate, "", "", portfolioFundID)` | Yes |
| Price Update (Today) | `FundService.UpdateCurrentFundPrice` | `(priceDate, "", fundID, "")` | Partial |
| Price Update (Manual) | `DeveloperService.UpdateFundPrice` | `(priceDate, "", fundID, "")` | Yes |
| Price Update (Historical) | `FundService.UpdateHistoricalFundPrice` | `(earliestDate, "", fundID, "")` | Partial |
| Dividend CRUD | `DividendService` Create/Update/Delete | `(exDivDate, "", "", portfolioFundID)` | Yes |
| Dividend Date Change | `DividendService.UpdateDividend` | `(min(old,new), "", "", portfolioFundID)` | Yes |
| IBKR Allocation | `IbkrService.AllocateIbkrTransaction` | `(txDate, portfolioID, "", "")` | No |
| IBKR Modify Allocation | `IbkrService.ModifyAllocations` | `(txDate, portfolioID, "", "")` for old+new | No |
| IBKR Unallocate | `IbkrService.UnallocateIbkrTransaction` | `(txDate, portfolioID, "", "")` | No |
| IBKR Match Dividend | `IbkrService.MatchDividend` | `(txDate, portfolioID, "", "")` | No |
| Sell Transaction | `TransactionService` (same as CRUD) | `(txDate, "", "", portfolioFundID)` | Yes |
| CSV Transaction Import | `DeveloperService.ImportTransactions` | `(earliestDate, "", "", portfolioFundID)` | Yes |
| CSV Price Import | `DeveloperService.ImportFundPrices` | `(earliestDate, "", fundID, "")` | Yes |

## Test Coverage

### Test Files

| File | Tests | What's Covered |
|------|-------|----------------|
| `internal/service/materialized_service_test.go` | 16 | Regen from empty, by fundID/portfolioFundID, scoped invalidation, portfolio/fund history fallback, stale detection (EC 1-3), on-demand calculation |
| `internal/service/write_path_hooks_test.go` | 12 | Write-path hooks for TransactionService (Create/Update/Delete), DividendService (Create, UpdateMinDate x2), DeveloperService (UpdateFundPrice, ImportTransactions, ImportFundPrices), nil-guard safety |

### MockMaterializedInvalidator

`internal/testutil/mock_invalidator.go` provides a thread-safe mock that records calls
with `WaitForCall(timeout)` to synchronize with async goroutines.

### Remaining Test Gaps

| Gap | Why | Recommendation |
|-----|-----|----------------|
| `FundService.UpdateCurrentFundPrice` hook | Requires Yahoo Finance mock returning controllable data | Extend `MockYahooClient` to return specific prices; wire into `NewTestFundServiceWithMockYahoo` |
| `FundService.UpdateHistoricalFundPrice` hook | Requires Yahoo mock + transactions in place | Same as above |
| IBKR service hooks (4 methods) | Complex setup: IBKR config, allocations, encryption | Add IBKR-specific mock infrastructure |
| Duplicate prevention / supersede logic | Needs concurrent goroutine test | Test `triggerBackgroundRegeneration` with overlapping dates |
| Concurrent write + read | Needs parallel test with real regen | Integration test with multiple goroutines |

## Edge Case 7: Sell Transactions with Realized Gains

Realized gain/loss records are created, updated, and deleted **inside the same DB transaction**
as the sell transaction itself. The `TransactionService` handles the full lifecycle:

- **CreateTransaction** (type=sell) → validates sufficient shares, creates RGL record, commits,
  then triggers invalidation
- **UpdateTransaction** → if old type was sell, deletes old RGL; if new type is sell, creates
  new RGL; handles all transitions (buy→sell, sell→buy, sell→sell)
- **DeleteTransaction** → cleans up RGL record if type is sell, plus IBKR allocation cleanup

The `RealizedGainLossService` has no direct `MaterializedInvalidator` integration — this is
intentional since RGL records are always a side-effect of sell transactions, never written
independently. The `TransactionService` invalidation hooks cover all paths.

Test coverage: `internal/service/transaction_service_test.go` has 15+ tests for RGL creation,
weighted average cost calculation, insufficient shares validation, update recalculation,
delete cleanup, and IBKR status reversion.

## SQLite Concurrency

### WAL Mode + Busy Timeout

Both production (`internal/database/database.go`) and test (`internal/testutil/database.go`)
databases are configured with:

```sql
PRAGMA journal_mode = WAL;     -- concurrent reads during writes
PRAGMA busy_timeout = 5000;    -- queue writers instead of SQLITE_BUSY
```

### Write Mutex

`regenWriteMu sync.Mutex` in `MaterializedService` serializes all background regen writes.
Without this, concurrent API requests each launching regen goroutines still cause
`SQLITE_BUSY` despite WAL + busy_timeout (because busy_timeout only helps when contention
is brief; regen writes can take hundreds of milliseconds).

## File Reference

| File | What changed |
|------|-------------|
| `internal/service/materialized_service.go` | `checkStaleData`, `triggerBackgroundRegeneration`, `runRegenLoop`, `regenInFlight`, `regenWriteMu`, `RegenerateMaterializedTable` |
| `internal/repository/materialized_repository.go` | `GetLatestMaterializedDate`, `GetLatestSourceDates`, scoped `InvalidateMaterializedTable`, `InsertMaterializedEntries` |
| `internal/service/transaction_service.go` | `materializedInvalidator` + nil-guarded regen calls |
| `internal/service/fund_service.go` | `materializedInvalidator` + nil-guarded regen calls |
| `internal/service/dividend_service.go` | `materializedInvalidator` + nil-guarded regen calls + Edge Case 5 |
| `internal/service/ibkr_service.go` | `materializedInvalidator` + allocation-based portfolio ID regen |
| `internal/service/developer_service.go` | `materializedInvalidator` + CSV import hooks + manual price update hook |
| `cmd/server/main.go` | Wiring: `SetMaterializedInvalidator` for all services |
| `internal/service/materialized_service_test.go` | 16 tests: regen, fallback, stale detection |
| `internal/service/write_path_hooks_test.go` | 12 tests: write-path hook verification |
| `internal/service/transaction_service_test.go` | 15+ tests: sell RGL lifecycle, insufficient shares, IBKR cleanup |
| `internal/testutil/mock_invalidator.go` | `MockMaterializedInvalidator` for async hook tests |
| `internal/testutil/helpers.go` | `NewTestMaterializedService` with full dependencies |
| `internal/testutil/factories.go` | `DividendBuilder` with `WithExDividendDate`, `WithRecordDate` |
| `internal/database/database.go` | `_texttotime=1`, WAL mode, busy_timeout |
| `internal/testutil/database.go` | `_texttotime=1`, WAL mode, busy_timeout |

## Bugs Fixed

1. **Date not propagated to FundHistoryEntry** — `calculateFundEntry` doesn't set the Date
   field; `RegenerateMaterializedTable` now propagates it from `FundHistoryResponse.Date`
2. **Wrong parameter in portfolioID branch** — Called `GetPortfolioFund(portfolioFundID)`
   instead of using the portfolio ID
3. **portfolioFundID treated as portfolioID** — Passed a portfolio_fund ID to
   `calculateFundHistoryOnFly` which expects a portfolio ID
4. **Invalidation/insertion outside DB transaction** — Used `s.materializedRepo` directly
   instead of `s.materializedRepo.WithTx(tx)`, so the delete + insert weren't atomic
5. **Long-running reads inside write tx** — `calculateFundHistoryOnFly` (read-heavy) was
   inside the DB transaction, holding it open longer than necessary. Moved outside the tx.
6. **DeveloperService.UpdateFundPrice missing hook** — Manual price updates via the developer
   API did not trigger cache invalidation. Fixed by adding regen call after commit.
