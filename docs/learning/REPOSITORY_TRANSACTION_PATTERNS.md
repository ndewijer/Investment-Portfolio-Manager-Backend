# Repository Transaction Patterns

## Pattern 1: Composable (WithTx) — Standard pattern

Used by all write methods in this codebase. The repository participates in an external transaction if one is provided, otherwise uses the raw DB connection.

```go
func (r *FundRepository) getQuerier() interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    // ... other methods
} {
    if r.tx != nil {
        return r.tx
    }
    return r.db
}

func (r *FundRepository) DeleteFund(ctx context.Context, id string) error {
    _, err := r.getQuerier().ExecContext(ctx, `DELETE FROM fund WHERE id = ?`, id)
    return err
}
```

The service owns the transaction:

```go
tx, err := s.db.BeginTx(ctx, nil)
if err != nil { ... }
defer func() { _ = tx.Rollback() }()

err = s.fundRepo.WithTx(tx).DeleteFund(ctx, id)
if err != nil { ... }

return tx.Commit()
```

---

## Pattern 2: Standalone-safe AND composable

Use this when a repository method needs to be both:
- Called standalone (no external transaction) — must be atomic on its own
- Called within an external transaction — must participate in it

This was previously used in `InsertFundPrices` but was removed in favour of Pattern 1 so that the service-level transaction in `UpdateHistoricalFundPrice` owns atomicity end-to-end. The repo method now uses `getQuerier()` and participates in whatever transaction the service provides. Kept here as a reference for cases where Pattern 2 is genuinely needed.

```go
func (r *FundRepository) InsertFundPrices(ctx context.Context, fundPrices []model.FundPrice) error {
    if len(fundPrices) == 0 {
        return nil
    }

    var tx *sql.Tx
    var err error
    var shouldCommit bool

    if r.tx != nil {
        // Participating in an external transaction — don't commit
        tx = r.tx
        shouldCommit = false
    } else {
        // No external transaction — own our atomicity
        tx, err = r.db.BeginTx(ctx, nil)
        if err != nil {
            return fmt.Errorf("begin transaction: %w", err)
        }
        shouldCommit = true
        defer func() {
            if err != nil {
                _ = tx.Rollback()
            }
        }()
    }

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO fund_price (id, fund_id, date, price)
        VALUES (?, ?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()

    for _, fp := range fundPrices {
        _, err = stmt.ExecContext(ctx, fp.ID, fp.FundID, fp.Date.Format("2006-01-02"), fp.Price)
        if err != nil {
            return fmt.Errorf("failed to insert fund price for %s on %s: %w",
                fp.FundID, fp.Date.Format("2006-01-02"), err)
        }
    }

    if shouldCommit {
        if err = tx.Commit(); err != nil {
            return fmt.Errorf("commit transaction: %w", err)
        }
    }

    return nil
}
```

### When to use Pattern 2 vs Pattern 1

| Scenario | Pattern |
|----------|---------|
| Method is always called from a service that manages the transaction | Pattern 1 |
| Method may be called standalone (e.g. a background job, CLI command) AND from within a service transaction | Pattern 2 |
| Batch insert that must be atomic even without an external transaction | Pattern 2 |

In practice, prefer Pattern 1 and let the service own transaction management. Only reach for Pattern 2 if a repo method genuinely needs to be safe when called without a wrapping transaction.
