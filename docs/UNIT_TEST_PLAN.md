# Unit Test Plan: Calculation-Focused Coverage

This document identifies gaps in calculation-focused unit testing across the service and repository layers. The current test suite (`portfolio_service_test.go`) covers basic CRUD flows but leaves all financial calculation logic untested.

---

## Current State

| Layer | Test Files | Coverage |
|-------|-----------|----------|
| `internal/service/` | `portfolio_service_test.go` only | Basic GetAll — no calculations |
| `internal/repository/` | None | 0% |

---

## Service Layer

### 1. `general_helpers.go` — `round()` — **BLOCKER**

**What it does:** Rounds values to a fixed precision for all financial output.

**Design intent:** `RoundingPrecision = 1e6` rounds to **6 decimal places**. This is intentional — the application handles fractional shares where cost basis and share quantities require high sub-decimal precision. The doc comment examples on `general_helpers.go` (e.g. `round(123.456789) // returns 123.46`) are incorrect and should be fixed to reflect actual 6dp behaviour.

**Test cases:**
| Input | Expected | Notes |
|-------|---------|-------|
| `1.0000001` | `1.0` | 7th decimal truncated |
| `1.1234564` | `1.123456` | Rounds down at 6th dp |
| `1.1234565` | `1.123457` | Rounds up at 6th dp |
| `-1.1234565` | `-1.123457` | Negative rounding (away from zero) |
| `0.0` | `0.0` | Zero passthrough |
| `math.Inf(1)` / `math.NaN()` | Pass through | No panic |

**Fix needed:** Correct the misleading doc comment examples on `general_helpers.go`.

---

### 2. `fund_metrics.go` — `calculateFundMetrics()`, `getPriceForDate()`, `getLatestPrice()`

**What it calculates:** Total shares, cost basis (weighted average), market value, unrealized gains, dividends, fees — all driven by transaction history.

**Core transaction switch logic:**
- `buy`: shares += tx.Shares; cost += tx.Shares * tx.CostPerShare
- `sell`: shares -= tx.Shares; cost rescaled to remaining shares (or reset to 0)
- `dividend`: dividends += tx.Shares * tx.CostPerShare
- `fee`: cost += tx.CostPerShare; fees += tx.CostPerShare

**`calculateFundMetrics()` test cases:**

| Case | Transactions | Div Shares | Prices | Expected Shares | Expected Cost | Notes |
|------|-------------|-----------|--------|----------------|---------------|-------|
| Single buy | buy 100@$10 | 0 | $10 | 100 | $1,000 | Base case |
| Buy + sell all | buy 100@$10, sell 100 | 0 | $10 | 0 | $0 | Cost resets to 0 |
| Buy + partial sell | buy 100@$10, sell 30 | 0 | $10 | 70 | $700 | Weighted avg preserved |
| Multiple buys | buy 50@$10, buy 50@$12 | 0 | $11 | 100 | $1,100 | Avg cost = $11 |
| Dividend reinvestment seed | (none) | 100 | $10 | 100 | $0 | Seed shares, no cost |
| Fee before buy | fee $5, buy 100@$10 | 0 | $10 | 100 | $1,005 | Fee increases cost basis |
| No price data | buy 100@$10 | 0 | (none) | 100 | $1,000 | Value = 0 |
| Zero shares after sell | buy 100@$10, sell 100 | 0 | $10 | 0 | $0 | Division guard at `shares > 0` |
| Date boundary — tx on exact date | buy on target date | 0 | $10 | 100 | $1,000 | `Before OR Equal` is inclusive |
| Date boundary — tx after date | buy after target date | 0 | $10 | 0 | $0 | Future tx excluded |

**`getPriceForDate()` test cases:**

| Prices | Target Date | Expected | Notes |
|--------|-------------|---------|-------|
| `[]` | any | `0` | Empty array |
| `[(2025-02-01, $10)]` | `2025-01-01` | `0` | All prices in future |
| `[(2025-01-01, $10)]` | `2025-01-01` | `$10` | Exact match |
| `[(2025-01-01, $10), (2025-01-05, $12)]` | `2025-01-03` | `$10` | Latest before date |
| `[(2025-01-01, $10), (2025-01-01, $11)]` | `2025-01-01` | `$11` | Last wins if same day |

---

### 3. `fund_helpers.go` — `calculateAndAssignFundMetrics()`

**Key concern:** Average cost calculation: `cost / roundedShares`. Protected by `if roundedShares > 0` — but what if raw shares = `0.0004` and `round()` produces `0.00`? The cost would remain non-zero with zero shares.

**Test cases:**

| Scenario | Expected Behavior |
|----------|-----------------|
| Zero shares (all sold) | `averageCost = 0`, value = 0 |
| Shares round to 0 (tiny fractional remainder) | `averageCost = 0` (division guard) |
| No price data | `currentValue = 0`, unrealized gain = negative cost |
| All zero inputs | All output fields = 0 |

---

### 4. `dividend_service.go` — `processDividendSharesForDate()`, `processDividendAmountForDate()`

**Critical assumption:** Both functions `break` early when a record's date exceeds the target — this assumes data is **sorted ascending by date**. If unsorted, data is silently dropped.

**`processDividendSharesForDate()` test cases:**

| Scenario | Expected |
|----------|---------|
| Empty dividend map | Empty result map |
| Dividend with no reinvestment transaction ID | `0` shares for that fund |
| Reinvestment TX ID set but TX not found | `0` shares (silent miss — worth noting in a comment) |
| Multiple dividends, all before date | All accumulated |
| One dividend after date | Excluded |
| Date on exact ex-dividend date | Included (`Equal` check) |
| Unsorted dividends with later date first | **Bug risk** — break exits early |

**`processDividendAmountForDate()` test cases:**

| Scenario | Expected |
|----------|---------|
| Empty array | `0.0` |
| All dividends before date | Sum of all `TotalAmount` |
| Mixed before/after | Only before-date amounts summed |
| Negative `TotalAmount` | Negative accumulation (no guard) |
| Floating-point accumulation over many records | Verify within acceptable precision |

**Dividend status logic — `CreateDividend()` / `UpdateDividend()`:**

| Fund Type | BuyOrderDate | Price | Shares | Expected Status |
|-----------|-------------|-------|--------|----------------|
| STOCK | Not set | — | — | PENDING |
| STOCK | Set | 0 | 0 | PENDING |
| STOCK | Set | $10 | 50 | COMPLETED (if shares*price == total) |
| STOCK | Set | $10 | 49 | PARTIAL |
| non-STOCK | Not set | — | — | COMPLETED |
| non-STOCK | Set | $10 | 50 | COMPLETED |

---

### 5. `realizedGainLoss_service.go` — `processRealizedGainLossForDate()`

Same sorted-data assumption as dividends: `break` when a record's date exceeds target.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty array | `(0, 0, 0, nil)` |
| All records before date | Full accumulation |
| Mix of gain and loss records | Net sum |
| Single loss (negative `RealizedGainLoss`) | Negative total |
| Date on exact transaction date | Included |
| Unsorted data | **Bug risk** — break exits early |

---

### 6. `materialized_helpers.go` — `calculateDisplayDateRange()`

**Test cases:**

| requestedStart | requestedEnd | dataStart | dataEnd | Expected displayStart | Expected displayEnd |
|---------------|-------------|----------|--------|---------------------|-------------------|
| Before dataStart | After dataEnd | 2020-01-01 | 2025-12-31 | dataStart | today (capped) |
| After dataEnd | After dataEnd | — | 2020-01-01 | dataStart (no data) | dataEnd |
| Same as dataStart | Same as dataEnd | 2020-01-01 | 2025-12-31 | 2020-01-01 | 2025-12-31 |
| requestedStart > requestedEnd | — | — | — | Undefined — no validation |

---

### 7. `materialized_helpers.go` — `calculateSinglePortfolioSummary()` — rounding order

**Key concern:** `TotalUnrealizedGainLoss = round(value - cost)` vs `round(value) - round(cost)` — these can produce different results.

```
value = 100.005, cost = 100.001
round(100.005 - 100.001) = round(0.004) = 0.000000  (at 6dp)
round(100.005) - round(100.001) = 100.005 - 100.001 = 0.004
```

**Test cases:**
- Values that round consistently
- Values where intermediate rounding differs from final rounding
- Verify `TotalGainLoss = UnrealizedGain + RealizedGainLoss` matches individual rounded components

---

### 8. `ibkr_service.go` — `GetTransactionAllocations()`, `GetEligiblePortfolios()`

**`GetTransactionAllocations()` test cases:**

| Scenario | Expected |
|----------|---------|
| No allocations | Empty slice |
| Only fees, no trades | Empty response (fees skipped in output loop) |
| Multiple fees for same portfolio | Fees correctly accumulated before output |
| Fees for portfolio with no trades | Fees in map but not emitted |

**`GetEligiblePortfolios()` test cases:**

| Scenario | Expected |
|----------|---------|
| ISIN matches | Returns fund by ISIN |
| Symbol matches (no ISIN) | Returns fund by symbol |
| Both ISIN and Symbol provided, ISIN matches | ISIN preferred |
| Symbol has exchange suffix in DB (`AAPL.NASDAQ`) | Matches input `AAPL` |
| Neither matches | Returns `found=false`, no error |
| DB error on ISIN lookup | Returns error |
| DB error on Symbol lookup | Returns error |

---

## Repository Layer

### Test Infrastructure

All repository tests should use the existing `testutil` package:

```go
db := testutil.SetupTestDB(t)
// Build test data with fluent builders:
portfolio := testutil.CreatePortfolio(t, db, "Test Portfolio")
fund := testutil.NewFund().Build(t, db)
pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
```

---

### 9. `transaction_repository.go` — `GetSharesOnDate()`

Aggregates total shares via SQL CASE: buys/dividends add, sells subtract.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty `portfolioFundID` | Error (`ErrInvalidPortfolioID`) |
| No transactions | `0.0` (not error) |
| Buy-only transactions | Sum of all buy shares |
| Sell-only | Negative result (short position) |
| Dividend transactions | Add to shares (same as buy) |
| Buy + partial sell | Correct net shares |
| Date before first transaction | `0.0` |
| Date exactly on transaction date | Included |
| Date after last transaction | Full total |
| Multiple transactions on same date | All included |

---

### 10. `transaction_repository.go` — `GetTransactions()`

Returns `map[pfID][]Transaction` with date range filtering.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty `pfIDs` | Empty map (not nil) |
| No transactions in range | Map with no keys |
| Single portfolio fund | One key in map |
| Multiple portfolio funds | Correct grouping per key |
| Date range inclusive (start == end) | Single-day transactions included |
| `startDate > endDate` | Empty map |
| Sorted by date ASC | Verify order |

---

### 11. `transaction_repository.go` — `GetOldestTransaction()`

Returns `time.Time{}` (zero value) on all failure modes — no error returned.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty `pfIDs` | `time.Time{}` |
| No transactions | `time.Time{}` |
| Single transaction | That transaction's date |
| Multiple transactions | Earliest date returned |
| Multiple portfolio funds | Earliest across all |

---

### 12. `dividend_repository.go` — `GetDividendPerPF()`

Groups dividends by portfolio fund ID, handles nullable fields.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty `pfIDs` | Empty map |
| No dividends in date range | Empty map |
| Nullable `buy_order_date` (NULL) | `BuyOrderDate` is zero time |
| Nullable `reinvestment_transaction_id` (NULL) | Empty string |
| Multiple dividends per fund | Correctly grouped |
| Date filtering by `ex_dividend_date` | Inclusive range |
| Status `pending` vs `completed` | Correct field mapping |

---

### 13. `fund_repository.go` — `GetFundPrice()`

Returns `map[fundID][]FundPrice` with sort direction control.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| `startDate > endDate` | Returns error |
| No prices in range | Empty map |
| `ascending=true` | Oldest price first |
| `ascending=false` | Newest price first |
| Multiple funds | Correct per-fund grouping |
| Fund with no prices | Not in map |
| Multiple prices on same day | All included |

---

### 14. `fund_repository.go` — `GetFundBySymbolOrIsin()`

Symbol matching strips exchange suffix: `AAPL.NASDAQ` → matches input `AAPL`.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Both empty | Error |
| Symbol only | Match by symbol |
| ISIN only | Match by ISIN |
| DB has `AAPL.NASDAQ`, input `AAPL` | Match (suffix stripped) |
| DB has `AAPL`, input `AAPL` | Match (no dot) |
| No match | `ErrFundNotFound` |

---

### 15. `realized_gain_loss_repository.go` — `GetRealizedGainLossByPortfolio()`

Aggregates by portfolio ID with date filtering.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty portfolios | Empty map |
| No records in date range | Empty map |
| Gain record | Positive `RealizedGainLoss` |
| Loss record | Negative `RealizedGainLoss` |
| Multiple portfolios | Correct grouping |
| Date filtering (inclusive) | Boundary dates included |

---

### 16. `materialized_repository.go` — `GetMaterializedHistory()`

Streaming aggregation using callback pattern; correlated subqueries for cumulative totals.

**Test cases:**

| Scenario | Expected |
|----------|---------|
| Empty `portfolioIDs` | Returns `nil` immediately, callback never called |
| Callback returns error | Error propagated |
| Callback invoked in date order | Sorted ASC |
| Multiple funds in one portfolio on same date | Values summed |
| Cumulative realized gains | Includes all prior dates, not just range |
| Cumulative dividends | Same |
| NULL subquery result (no dividends ever) | `COALESCE` returns 0 |

---

## Recommended Implementation Order

| Priority | File | Reason |
|----------|------|--------|
| 1 | `general_helpers_test.go` | Clarify/fix `round()` precision — blocks everything else |
| 2 | `fund_metrics_test.go` | Core per-fund calculation engine |
| 3 | `dividend_service_test.go` | Reinvestment logic and status state machine |
| 4 | `realizedGainLoss_service_test.go` | Gain/loss accumulation |
| 5 | `transaction_repository_test.go` | `GetSharesOnDate` is foundational |
| 6 | `fund_helpers_test.go` | Orchestration layer — builds on 2–4 |
| 7 | `materialized_helpers_test.go` | Date range and portfolio summary rounding |
| 8 | `dividend_repository_test.go` | Nullable field and date boundary tests |
| 9 | `fund_repository_test.go` | Symbol matching, price sorting |
| 10 | `materialized_repository_test.go` | Most complex — cumulative subquery logic |
| 11 | `ibkr_service_test.go` | IBKR allocation matching |

---

## Cross-Cutting Concerns to Test in Each Layer

1. **Sorted-data assumption** — `processDividendSharesForDate`, `processDividendAmountForDate`, `processRealizedGainLossForDate`, `getPriceForDate` all `break` when a date exceeds the target. Add a test with unsorted data to document/expose this behavior.

2. **Date boundary inclusivity** — Most functions use `Before(date) || Equal(date)` or SQL `date <= ?`. Always test the exact boundary date.

3. **Floating-point accumulation** — Test functions that accumulate in loops over many records (10–100 iterations) and assert the result matches `round()` applied once to the expected total.

4. **Empty/nil inputs** — Every aggregation function should be tested with empty slices and empty maps; most return zero values rather than errors.

5. **Zero-division guards** — Verify `averageCost` is `0` (not NaN/Inf) when `shares == 0`.
