```
// 🔍 PASTE THIS RIGHT AFTER line 643 (after testutil.NewDividend...Build(t, db))
// This will show you exactly what's in the database and why dividends might be 0

// Save database for manual inspection
_, err := db.Exec("VACUUM INTO '/tmp/test_dividend.db'")
if err == nil {
	t.Logf("✅ Database saved to /tmp/test_dividend.db")
	t.Logf("   Inspect with: sqlite3 /tmp/test_dividend.db")
}

// Check what's actually in the dividend table
t.Logf("\n=== DIVIDEND TABLE INSPECTION ===")

var dividendCount int
db.QueryRow("SELECT COUNT(*) FROM dividend").Scan(&dividendCount)
t.Logf("Total dividends in DB: %d", dividendCount)

// Look for the specific dividend we just created
var divID, divFundID, divPFID string
var divShares, divPerShare float64
err = db.QueryRow(`
	SELECT id, fund_id, portfolio_fund_id, shares_owned, dividend_per_share
	FROM dividend
	WHERE portfolio_fund_id = ?
`, pfID).Scan(&divID, &divFundID, &divPFID, &divShares, &divPerShare)

if err == sql.ErrNoRows {
	t.Logf("❌ NO DIVIDEND found for portfolio_fund_id = %s", pfID)
	t.Logf("   This means NewDividend().Build() didn't insert anything!")
} else if err != nil {
	t.Logf("❌ Error querying dividend: %v", err)
} else {
	t.Logf("✅ Dividend found in DB:")
	t.Logf("   ID: %s", divID)
	t.Logf("   FundID: %s (matches created fund: %v)", divFundID, divFundID == fund.ID)
	t.Logf("   PortfolioFundID: %s (matches pfID: %v)", divPFID, divPFID == pfID)
	t.Logf("   SharesOwned: %.2f", divShares)
	t.Logf("   DividendPerShare: $%.2f", divPerShare)
	t.Logf("   Expected Total: %.2f * %.2f = $%.2f", divShares, divPerShare, divShares*divPerShare)
}

// Check what the service query would find
t.Logf("\n=== SIMULATING SERVICE QUERY ===")
rows, err := db.Query(`
	SELECT d.id, d.dividend_per_share, d.shares_owned, d.is_reinvested
	FROM dividend d
	INNER JOIN portfolio_fund pf ON d.portfolio_fund_id = pf.id
	WHERE pf.portfolio_id = ?
`, portfolio.ID)

if err != nil {
	t.Logf("❌ Service query failed: %v", err)
} else {
	defer rows.Close()
	dividendSum := 0.0
	count := 0
	for rows.Next() {
		var id string
		var perShare, shares float64
		var reinvested bool
		rows.Scan(&id, &perShare, &shares, &reinvested)
		total := perShare * shares
		dividendSum += total
		t.Logf("   Dividend #%d: $%.2f per share * %.2f shares = $%.2f", count+1, perShare, shares, total)
		count++
	}
	t.Logf("   Total dividends from service query: $%.2f (found %d dividends)", dividendSum, count)
	if dividendSum == 0 && count > 0 {
		t.Logf("   ⚠️  Found dividends but total is 0 - check if shares_owned or dividend_per_share is 0!")
	}
	if count == 0 {
		t.Logf("   ❌ Service query found 0 dividends - JOIN or WHERE clause issue!")
	}
}

t.Logf("=========================\n")
```

and

```
jsonBytes, _ := json.MarshalIndent(response, "", "  ")
		t.Logf("Response JSON:\n%s", string(jsonBytes))

```

## XML struct generation

[zek](https://github.com/miku/zek/) — generates Go structs from XML input.
Pipe sample XML into it and it produces tagged structs ready to use with `encoding/xml`.

```bash
cat sample.xml | zek -e -p
```

Used for generating IBKR FlexQuery response structs.
