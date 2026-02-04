# Debugging Tests - Practical Tips

When tests fail and you can't figure out why, these techniques help you see what's actually happening.

---

## 1. Dump Test Database for Manual Inspection

**Problem**: Test fails but you can't see what's in the database.

**Solution**: Export the test database to a file you can open with SQLite.

```go
// Add this anywhere in your test to save a snapshot
_, err := db.Exec("VACUUM INTO '/tmp/test_snapshot.db'")
if err == nil {
    t.Logf("✅ Database saved to /tmp/test_snapshot.db")
    t.Logf("   Inspect with: sqlite3 /tmp/test_snapshot.db")
}
```

**Then inspect:**
```bash
sqlite3 /tmp/test_snapshot.db
sqlite> .tables
sqlite> SELECT * FROM portfolio;
sqlite> .schema portfolio
sqlite> .exit
```

---

## 2. Log Raw Database Contents

**Problem**: Test assertions fail but you don't know what the database actually contains.

**Solution**: Query the database directly and log everything.

```go
t.Logf("\n=== DATABASE INSPECTION ===")

// Count rows
var count int
db.QueryRow("SELECT COUNT(*) FROM dividend").Scan(&count)
t.Logf("Total dividends in DB: %d", count)

// Check specific record
var id, fundID, pfID string
var shares, perShare float64
err := db.QueryRow(`
    SELECT id, fund_id, portfolio_fund_id, shares_owned, dividend_per_share
    FROM dividend
    WHERE portfolio_fund_id = ?
`, expectedPFID).Scan(&id, &fundID, &pfID, &shares, &perShare)

if err == sql.ErrNoRows {
    t.Logf("❌ NO DIVIDEND found for portfolio_fund_id = %s", expectedPFID)
    t.Logf("   This means the builder didn't insert anything!")
} else if err != nil {
    t.Logf("❌ Error querying dividend: %v", err)
} else {
    t.Logf("✅ Dividend found:")
    t.Logf("   ID: %s", id)
    t.Logf("   FundID: %s", fundID)
    t.Logf("   PortfolioFundID: %s", pfID)
    t.Logf("   SharesOwned: %.2f", shares)
    t.Logf("   DividendPerShare: $%.2f", perShare)
    t.Logf("   Expected Total: %.2f * %.2f = $%.2f", shares, perShare, shares*perShare)
}

t.Logf("===========================\n")
```

---

## 3. Simulate Service Queries

**Problem**: Handler test passes but integration test fails. Want to see what the service layer query returns.

**Solution**: Run the exact same query the service uses, log results.

```go
t.Logf("\n=== SIMULATING SERVICE QUERY ===")

rows, err := db.Query(`
    SELECT d.id, d.dividend_per_share, d.shares_owned, d.is_reinvested
    FROM dividend d
    INNER JOIN portfolio_fund pf ON d.portfolio_fund_id = pf.id
    WHERE pf.portfolio_id = ?
`, portfolioID)

if err != nil {
    t.Logf("❌ Service query failed: %v", err)
} else {
    defer rows.Close()
    total := 0.0
    count := 0

    for rows.Next() {
        var id string
        var perShare, shares float64
        var reinvested bool
        rows.Scan(&id, &perShare, &shares, &reinvested)

        dividendAmount := perShare * shares
        total += dividendAmount
        count++

        t.Logf("   Dividend #%d:", count)
        t.Logf("      ID: %s", id)
        t.Logf("      PerShare: $%.2f", perShare)
        t.Logf("      Shares: %.2f", shares)
        t.Logf("      Total: $%.2f", dividendAmount)
        t.Logf("      Reinvested: %v", reinvested)
    }

    t.Logf("\n   Total dividends: $%.2f (found %d records)", total, count)

    if total == 0 && count > 0 {
        t.Logf("   ⚠️  Found dividends but total is 0")
        t.Logf("      → Check if shares_owned or dividend_per_share is 0")
    }
    if count == 0 {
        t.Logf("   ❌ Query found 0 dividends")
        t.Logf("      → Check JOIN conditions or WHERE clause")
    }
}

t.Logf("===========================\n")
```

---

## 4. Pretty-Print JSON Responses

**Problem**: HTTP handler returns JSON but test assertion fails. Want to see the actual response.

**Solution**: Pretty-print the JSON response.

```go
// After making HTTP request and decoding response
jsonBytes, _ := json.MarshalIndent(response, "", "  ")
t.Logf("Response JSON:\n%s", string(jsonBytes))
```

**Or for raw response body:**
```go
// Before you decode it
bodyBytes, _ := io.ReadAll(resp.Body)
t.Logf("Raw Response Body:\n%s", string(bodyBytes))

// Re-wrap it so you can still decode
resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
```

---

## 5. Log All Test Fixtures Being Created

**Problem**: Test uses multiple builders and you lose track of what was created.

**Solution**: Log each fixture as you build it.

```go
portfolio := testutil.NewPortfolio().
    WithName("Test Portfolio").
    Build(t, db)
t.Logf("Created Portfolio: ID=%s, Name=%s", portfolio.ID, portfolio.Name)

fund := testutil.NewFund().
    WithSymbol("VWCE").
    Build(t, db)
t.Logf("Created Fund: ID=%s, Symbol=%s", fund.ID, fund.Symbol)

pf := testutil.NewPortfolioFund().
    WithPortfolioID(portfolio.ID).
    WithFundID(fund.ID).
    Build(t, db)
t.Logf("Created PortfolioFund: ID=%s, Links Portfolio %s → Fund %s", pf.ID, portfolio.ID, fund.ID)

dividend := testutil.NewDividend().
    WithPortfolioFundID(pf.ID).
    WithShares(100).
    WithDividendPerShare(1.50).
    Build(t, db)
t.Logf("Created Dividend: ID=%s, Shares=%.2f, PerShare=$%.2f, Total=$%.2f",
    dividend.ID, 100.0, 1.50, 150.0)
```

---

## 6. Compare Expected vs Actual

**Problem**: Assertion fails but diff is hard to read.

**Solution**: Log both side-by-side with clear labels.

```go
t.Logf("\n=== COMPARISON ===")
t.Logf("Expected:")
t.Logf("  Total: $%.2f", expectedTotal)
t.Logf("  Count: %d", expectedCount)
t.Logf("Actual:")
t.Logf("  Total: $%.2f", actualTotal)
t.Logf("  Count: %d", actualCount)
t.Logf("Match: %v", expectedTotal == actualTotal && expectedCount == actualCount)
t.Logf("==================\n")
```

---

## 7. Trace Query Execution

**Problem**: Complex query returns wrong results. Want to see what SQLite is doing.

**Solution**: Use SQLite's EXPLAIN QUERY PLAN.

```go
rows, _ := db.Query("EXPLAIN QUERY PLAN " + yourQuery, args...)
defer rows.Close()

t.Logf("\n=== QUERY PLAN ===")
for rows.Next() {
    var id, parent, notused int
    var detail string
    rows.Scan(&id, &parent, &notused, &detail)
    t.Logf("%d | %s", id, detail)
}
t.Logf("==================\n")
```

---

## 8. Check Foreign Key Relationships

**Problem**: Insert fails with foreign key constraint but you don't know which FK.

**Solution**: Query foreign key info and validate manually.

```go
t.Logf("\n=== FOREIGN KEY CHECK ===")

// Check if portfolio exists
var portfolioExists bool
db.QueryRow("SELECT EXISTS(SELECT 1 FROM portfolio WHERE id = ?)", portfolioID).Scan(&portfolioExists)
t.Logf("Portfolio %s exists: %v", portfolioID, portfolioExists)

// Check if fund exists
var fundExists bool
db.QueryRow("SELECT EXISTS(SELECT 1 FROM fund WHERE id = ?)", fundID).Scan(&fundExists)
t.Logf("Fund %s exists: %v", fundID, fundExists)

// Check if portfolio_fund exists
var pfExists bool
db.QueryRow("SELECT EXISTS(SELECT 1 FROM portfolio_fund WHERE id = ?)", pfID).Scan(&pfExists)
t.Logf("PortfolioFund %s exists: %v", pfID, pfExists)

t.Logf("==========================\n")
```

---

## 9. Dump All Tables

**Problem**: Nuclear option - you have no idea what's wrong.

**Solution**: Dump everything in the test database.

```go
func dumpDatabase(t *testing.T, db *sql.DB) {
    t.Helper()
    t.Logf("\n=== FULL DATABASE DUMP ===")

    tables := []string{"portfolio", "fund", "portfolio_fund", "transaction", "dividend"}

    for _, table := range tables {
        var count int
        db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
        t.Logf("\n%s: %d rows", table, count)

        if count > 0 && count < 10 { // Only dump if small
            rows, _ := db.Query(fmt.Sprintf("SELECT * FROM %s", table))
            defer rows.Close()

            cols, _ := rows.Columns()
            t.Logf("  Columns: %v", cols)

            for rows.Next() {
                values := make([]interface{}, len(cols))
                valuePtrs := make([]interface{}, len(cols))
                for i := range values {
                    valuePtrs[i] = &values[i]
                }
                rows.Scan(valuePtrs...)
                t.Logf("    %v", values)
            }
        }
    }

    t.Logf("==========================\n")
}

// Use it:
dumpDatabase(t, db)
```

---

## 10. Verify Test Isolation

**Problem**: Tests pass individually but fail when run together.

**Solution**: Add logging to verify database is clean between tests.

```go
func TestMyFeature(t *testing.T) {
    db := testutil.SetupTestDB(t)

    // Verify database is empty
    var portfolioCount, fundCount int
    db.QueryRow("SELECT COUNT(*) FROM portfolio").Scan(&portfolioCount)
    db.QueryRow("SELECT COUNT(*) FROM fund").Scan(&fundCount)

    t.Logf("Test start: portfolios=%d, funds=%d", portfolioCount, fundCount)

    if portfolioCount > 0 || fundCount > 0 {
        t.Fatalf("❌ Database not clean at test start!")
    }

    // ... rest of test
}
```

---

## Common Patterns

### When JOIN returns no rows:
```go
// Check each table separately
var portfolioCount, fundCount, pfCount int
db.QueryRow("SELECT COUNT(*) FROM portfolio WHERE id = ?", pID).Scan(&portfolioCount)
db.QueryRow("SELECT COUNT(*) FROM fund WHERE id = ?", fID).Scan(&fundCount)
db.QueryRow("SELECT COUNT(*) FROM portfolio_fund WHERE portfolio_id = ? AND fund_id = ?",
    pID, fID).Scan(&pfCount)

t.Logf("Portfolio exists: %v", portfolioCount > 0)
t.Logf("Fund exists: %v", fundCount > 0)
t.Logf("PortfolioFund link exists: %v", pfCount > 0)
```

### When aggregate is 0 unexpectedly:
```go
// Check individual rows before aggregation
rows, _ := db.Query("SELECT id, amount FROM transactions WHERE portfolio_id = ?", pID)
defer rows.Close()

t.Logf("Individual transactions:")
total := 0.0
for rows.Next() {
    var id string
    var amount float64
    rows.Scan(&id, &amount)
    t.Logf("  %s: $%.2f", id, amount)
    total += amount
}
t.Logf("Manual sum: $%.2f", total)

// Compare to aggregate query
var dbTotal float64
db.QueryRow("SELECT SUM(amount) FROM transactions WHERE portfolio_id = ?", pID).Scan(&dbTotal)
t.Logf("Database SUM(): $%.2f", dbTotal)
```

---

## Quick Reference

| Problem | Solution |
|---------|----------|
| Can't see database state | `VACUUM INTO '/tmp/test.db'` |
| Don't know what query returns | Copy query to test, log results |
| JSON assertion fails | `json.MarshalIndent()` and log |
| Foreign key error | Check each FK exists manually |
| JOIN returns nothing | Check each table separately |
| Aggregate is wrong | Sum manually, compare to query |
| Tests interfere | Log counts at test start |
| Query is slow | `EXPLAIN QUERY PLAN` |

---

## Tips

1. **Use `t.Logf()`, not `fmt.Println()`** - Logs only appear for failing tests
2. **Export database early** - Don't wait until the end, export right after setup
3. **Log before assertions** - If assertion fails, you'll have context
4. **Keep debug code** - Comment it out but leave it in the test for next time
5. **Use `make test-v`** - Shows all logs even for passing tests

---

*File created from practical debugging experience during test development*
