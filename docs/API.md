# API Reference

All endpoints are served under `/api`. IDs are UUIDs unless noted otherwise.

## System

| Method | Path                | Description            |
|--------|---------------------|------------------------|
| GET    | `/system/health`    | Health check           |
| GET    | `/system/version`   | Version information    |

## Portfolio

| Method | Path                          | Description                      |
|--------|-------------------------------|----------------------------------|
| GET    | `/portfolio`                  | List all portfolios              |
| POST   | `/portfolio`                  | Create portfolio                 |
| GET    | `/portfolio/{id}`             | Get portfolio by ID              |
| PUT    | `/portfolio/{id}`             | Update portfolio                 |
| DELETE | `/portfolio/{id}`             | Delete portfolio                 |
| POST   | `/portfolio/{id}/archive`     | Archive portfolio                |
| POST   | `/portfolio/{id}/unarchive`   | Unarchive portfolio              |
| GET    | `/portfolio/summary`          | Portfolio summary (materialized) |
| GET    | `/portfolio/history`          | Portfolio history (materialized) |
| GET    | `/portfolio/funds/{id}`       | Funds in a portfolio             |
| POST   | `/portfolio/funds`            | Add fund to portfolio            |
| DELETE | `/portfolio/fund/{id}`        | Remove fund from portfolio       |

## Fund

| Method | Path                              | Description                          |
|--------|-----------------------------------|--------------------------------------|
| GET    | `/fund`                           | List all funds                       |
| POST   | `/fund`                           | Create fund                          |
| GET    | `/fund/{id}`                      | Get fund details                     |
| PUT    | `/fund/{id}`                      | Update fund                          |
| DELETE | `/fund/{id}`                      | Delete fund                          |
| GET    | `/fund/{id}/check-usage`          | Check if fund is in use              |
| GET    | `/fund/fund-prices/{id}`          | Price history for a fund             |
| POST   | `/fund/fund-prices/{id}/update`   | Update fund prices (Yahoo Finance)   |
| GET    | `/fund/history/{portfolioId}`     | Historical fund values for portfolio |
| GET    | `/fund/symbol/{symbol}`           | Look up trading symbol               |
| POST   | `/fund/update-all-prices`         | Update prices for all funds (API key required) |

## Transaction

| Method | Path                                | Description                    |
|--------|-------------------------------------|--------------------------------|
| GET    | `/transaction`                      | List all transactions          |
| POST   | `/transaction`                      | Create transaction             |
| GET    | `/transaction/{id}`                 | Get transaction by ID          |
| PUT    | `/transaction/{id}`                 | Update transaction             |
| DELETE | `/transaction/{id}`                 | Delete transaction             |
| GET    | `/transaction/portfolio/{id}`       | Transactions for a portfolio   |

## Dividend

| Method | Path                            | Description                  |
|--------|---------------------------------|------------------------------|
| GET    | `/dividend`                     | List all dividends           |
| POST   | `/dividend`                     | Create dividend              |
| GET    | `/dividend/{id}`                | Get dividend by ID           |
| PUT    | `/dividend/{id}`                | Update dividend              |
| DELETE | `/dividend/{id}`                | Delete dividend              |
| GET    | `/dividend/portfolio/{id}`      | Dividends for a portfolio    |
| GET    | `/dividend/fund/{id}`           | Dividends for a fund         |

## IBKR

| Method | Path                                          | Description                              |
|--------|-----------------------------------------------|------------------------------------------|
| GET    | `/ibkr/config`                                | Get IBKR configuration status            |
| POST   | `/ibkr/config`                                | Create or update IBKR configuration      |
| DELETE | `/ibkr/config`                                | Delete IBKR configuration                |
| POST   | `/ibkr/config/test`                           | Test IBKR connection                     |
| POST   | `/ibkr/import`                                | Trigger IBKR Flex report import          |
| GET    | `/ibkr/portfolios`                            | Available portfolios for allocation      |
| GET    | `/ibkr/dividend/pending`                      | Pending dividends for matching           |
| GET    | `/ibkr/inbox`                                 | List imported IBKR transactions          |
| GET    | `/ibkr/inbox/count`                           | Count of IBKR inbox transactions         |
| POST   | `/ibkr/inbox/bulk-allocate`                   | Bulk allocate transactions               |
| GET    | `/ibkr/inbox/{id}`                            | Get IBKR transaction details             |
| DELETE | `/ibkr/inbox/{id}`                            | Delete IBKR transaction                  |
| POST   | `/ibkr/inbox/{id}/allocate`                   | Allocate transaction to portfolios       |
| POST   | `/ibkr/inbox/{id}/unallocate`                 | Unallocate a processed transaction       |
| GET    | `/ibkr/inbox/{id}/allocations`                | Get allocation details                   |
| PUT    | `/ibkr/inbox/{id}/allocations`                | Modify allocation percentages            |
| GET    | `/ibkr/inbox/{id}/eligible-portfolios`        | Get eligible portfolios for transaction  |
| POST   | `/ibkr/inbox/{id}/ignore`                     | Mark transaction as ignored              |
| POST   | `/ibkr/inbox/{id}/match-dividend`             | Match dividend to existing records       |

## Developer

| Method | Path                                 | Description                          |
|--------|--------------------------------------|--------------------------------------|
| GET    | `/developer/logs`                    | Get system logs (cursor-based)       |
| DELETE | `/developer/logs`                    | Clear all system logs                |
| GET    | `/developer/system-settings/logging` | Get logging configuration            |
| PUT    | `/developer/system-settings/logging` | Update logging configuration         |
| GET    | `/developer/csv/fund-prices/template`| CSV template for fund price import   |
| GET    | `/developer/csv/transactions/template`| CSV template for transaction import |
| GET    | `/developer/exchange-rate`           | Get exchange rate for currency pair  |
| POST   | `/developer/exchange-rate`           | Set exchange rate for currency pair  |
| GET    | `/developer/fund-price`              | Get fund price for specific date     |
| POST   | `/developer/fund-price`              | Set fund price for specific date     |
| POST   | `/developer/import-fund-prices`      | Import fund prices from CSV          |
| POST   | `/developer/import-transactions`     | Import transactions from CSV         |
