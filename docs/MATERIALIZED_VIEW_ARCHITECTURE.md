# Materialized View Architecture

Reference document for backporting the materialized view pattern to the Python implementation.

---

## 1. Architecture Contract

The core principle: **expensive calculations happen once on the write path; the read path is cheap.**

- **Write path**: iterates transactions, dividends, and prices day-by-day, computing derived values (shares, value, cost, gains, etc.) and storing them in `fund_history_materialized`.
- **Read path**: queries the materialized table with `SELECT ... SUM() ... GROUP BY`. No correlated subqueries, no per-row recalculation.

This separation means the read path scales with the number of rows in the materialized table, not with the complexity of the underlying transaction history.

---

## 2. Schema Definition

Table: **`fund_history_materialized`**

| Column              | Type      | Description                                                                 |
|---------------------|-----------|-----------------------------------------------------------------------------|
| `id`                | INTEGER   | Primary key (auto-increment)                                                |
| `portfolio_fund_id` | INTEGER   | FK to the portfolio-fund association                                        |
| `fund_id`           | INTEGER   | FK to the fund                                                              |
| `date`              | TEXT/DATE | The date this row represents                                                |
| `shares`            | REAL      | Number of shares held on this date                                          |
| `price`             | REAL      | Price per share on this date                                                |
| `value`             | REAL      | Market value (shares * price)                                               |
| `cost`              | REAL      | Total cost basis for held shares                                            |
| `realized_gain`     | REAL      | Realized gain/loss recognized on this date                                  |
| `unrealized_gain`   | REAL      | Unrealized gain/loss (value - cost)                                         |
| `total_gain_loss`   | REAL      | Combined realized + unrealized gain/loss                                    |
| `dividends`         | REAL      | Dividend income on this date                                                |
| `fees`              | REAL      | Fees incurred on this date                                                  |
| `sale_proceeds`     | REAL      | **Cumulative** sale proceeds up to this date (from realized gain/loss data) |
| `original_cost`     | REAL      | **Cumulative** original cost of sold shares up to this date                 |
| `calculated_at`     | DATETIME  | Timestamp of when this row was computed                                     |

`sale_proceeds` and `original_cost` are cumulative per fund, sourced from `processRealizedGainLossForDate` (Go) or its equivalent. They represent the running totals of proceeds received and cost basis of shares sold, respectively.

---

## 3. Write Path Rules

### One row per fund per date

Each row in `fund_history_materialized` represents exactly one fund on one date. The write path iterates from the earliest relevant date to today, computing all columns for each fund on each date.

### Cumulative sale_proceeds and original_cost

`sale_proceeds` and `original_cost` are **cumulative** up to and including that date. On any given date, the write path:

1. Looks up realized gain/loss events for the fund on that date.
2. Adds the day's sale proceeds and original cost to the running totals.
3. Stores the running totals in the row.

This means the read path can take the value directly from any single date row without needing to sum across prior rows.

### Zero-shares guard

To prevent pre-first-buy and post-full-sell bloat, skip writing a row when **all** of the following are zero:

- `shares`
- `realized_gain`
- `dividends`
- `sale_proceeds`
- `original_cost`
- `fees`

This guard preserves rows for sold funds that still carry financial data (e.g., a fund fully sold on a given date will have `realized_gain`, `sale_proceeds`, and `original_cost` populated even though `shares` is zero).

### startDate clamping

When computing materialized data on-the-fly (not from a pre-built cache), clamp the start date to the **oldest transaction date** across all relevant portfolios. Never start iteration from epoch (1970-01-01) or an arbitrary early date. This avoids iterating through years of empty dates before the first transaction exists.

---

## 4. Read Path Rules

### Portfolio history query

Aggregate fund-level rows into portfolio-level history with a simple `SUM/GROUP BY`:

```sql
SELECT
    date,
    portfolio_id,
    SUM(value)           AS total_value,
    SUM(cost)            AS total_cost,
    SUM(realized_gain)   AS total_realized_gain,
    SUM(unrealized_gain) AS total_unrealized_gain,
    SUM(dividends)       AS total_dividends,
    SUM(sale_proceeds)   AS total_sale_proceeds,
    SUM(original_cost)   AS total_original_cost,
    SUM(fees)            AS total_fees,
    SUM(unrealized_gain) + SUM(realized_gain) AS total_gain_loss
FROM fund_history_materialized fhm
JOIN portfolio_funds pf ON fhm.portfolio_fund_id = pf.id
WHERE pf.portfolio_id = ?
  AND fhm.date BETWEEN ? AND ?
GROUP BY date, pf.portfolio_id
ORDER BY date;
```

**No correlated subqueries.** All values come directly from the materialized table. The read path never touches the transactions, dividends, or fund_price tables.

### Summary endpoint optimization

For endpoints that only need the current state (not full history), query only the latest date:

```sql
SELECT
    SUM(value)           AS total_value,
    SUM(cost)            AS total_cost,
    SUM(realized_gain)   AS total_realized_gain,
    SUM(unrealized_gain) AS total_unrealized_gain,
    SUM(dividends)       AS total_dividends,
    SUM(sale_proceeds)   AS total_sale_proceeds,
    SUM(original_cost)   AS total_original_cost,
    SUM(fees)            AS total_fees,
    SUM(unrealized_gain) + SUM(realized_gain) AS total_gain_loss
FROM fund_history_materialized fhm
JOIN portfolio_funds pf ON fhm.portfolio_fund_id = pf.id
WHERE pf.portfolio_id = ?
  AND fhm.date = (
      SELECT MAX(fhm2.date)
      FROM fund_history_materialized fhm2
      JOIN portfolio_funds pf2 ON fhm2.portfolio_fund_id = pf2.id
      WHERE pf2.portfolio_id = ?
  )
GROUP BY pf.portfolio_id;
```

This avoids loading and discarding full history when only the latest snapshot is needed.

---

## 5. Staleness Detection

The materialized table is a cache. It becomes stale when any of its source data changes.

### Source tables and their freshness signals

| Source Table   | Freshness Column |
|----------------|------------------|
| transactions   | `created_at`     |
| fund_price     | `date`           |
| dividend       | `created_at`     |

### Detection logic

```
materialized_freshness = MAX(calculated_at) FROM fund_history_materialized
                         WHERE portfolio_fund_id IN (relevant funds)

stale = (MAX(transactions.created_at) > materialized_freshness)
     OR (MAX(fund_price.date)         > materialized_freshness)
     OR (MAX(dividend.created_at)     > materialized_freshness)
```

If **any** source is newer than the materialized data:

1. The cache is stale.
2. Fall back to on-demand (on-the-fly) calculation for the current request.
3. Trigger background regeneration of the materialized data so subsequent requests hit the cache.

---

## 6. Checklist for Python Implementation

- [ ] **Add columns**: Add `sale_proceeds` and `original_cost` columns to the `fund_history_materialized` table (migration).
- [ ] **Write path -- cumulative values**: Update the write path to compute and store cumulative `sale_proceeds` and `original_cost` per fund per date.
- [ ] **Write path -- zero-shares guard**: Update the skip condition to check all six fields (`shares`, `realized_gain`, `dividends`, `sale_proceeds`, `original_cost`, `fees`). Only skip when all are zero.
- [ ] **Read path -- replace correlated subqueries**: Replace any correlated subqueries in portfolio history / summary queries with simple `SUM() ... GROUP BY` against `fund_history_materialized`.
- [ ] **Read path -- latest-date-only query**: Add (or update) summary endpoints to query only `WHERE date = (SELECT MAX(date) ...)` instead of loading full history.
- [ ] **On-the-fly calculation -- startDate clamping**: When computing materialized data on demand, clamp the start date to the oldest transaction date. Never iterate from epoch.
- [ ] **Staleness detection**: Implement the freshness check comparing `calculated_at` against source table timestamps, with fallback to on-demand calculation and background regeneration.
