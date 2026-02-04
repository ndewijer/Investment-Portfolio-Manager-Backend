# Developer Logs API Endpoint

## Endpoint
```
GET /api/developer/logs
```

## ⚠️ Important Notes

1. **Parameter names use camelCase and singular forms**:
   - Use `level` (supports comma-separated values: `level=error,critical`)
   - Use `category` (supports comma-separated values: `category=ibkr,system`)
   - Use `sortDir`, `perPage`, `startDate`, `endDate`, `source`, `cursor`

2. **Filter values are case-insensitive**:
   - Both `error` and `ERROR` work
   - Both `ibkr` and `IBKR` work
   - Values are automatically converted to match database storage

## Description
Retrieves application logs with advanced filtering, sorting, and pagination capabilities.

## Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `level` | string | No | - | Comma-separated list of log levels to filter by |
| `category` | string | No | - | Comma-separated list of categories to filter by |
| `startDate` | string | No | - | Start of date range (ISO 8601 or YYYY-MM-DD) |
| `endDate` | string | No | - | End of date range (ISO 8601 or YYYY-MM-DD) |
| `source` | string | No | - | Filter by source (partial match) |
| `message` | string | No | - | Filter by message text (partial match) |
| `sortDir` | string | No | desc | Sort direction: `asc` or `desc` |
| `cursor` | string | No | - | Pagination cursor from previous response |
| `perPage` | integer | No | 50 | Number of results per page (1-100) |

### Valid Values

**Levels**:
- `debug`
- `info`
- `warning`
- `error`
- `critical`

**Categories**:
- `portfolio`
- `fund`
- `transaction`
- `dividend`
- `system`
- `database`
- `security`
- `ibkr`
- `developer`

**Date Formats**:
- Date only: `2024-01-01`
- ISO 8601 with Z: `2024-01-01T00:00:00Z`
- ISO 8601 with offset: `2024-01-01T00:00:00+01:00`

## Response Format

```json
{
  "logs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "timestamp": "2024-01-15T10:30:00Z",
      "level": "error",
      "category": "portfolio",
      "message": "Failed to load portfolio",
      "details": "Database connection timeout after 30s",
      "source": "PortfolioHandler.GetPortfolio:142",
      "requestId": "req-abc-123",
      "httpStatus": "500",
      "ipAddress": "192.168.1.100",
      "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
    }
  ],
  "nextCursor": "2024-01-15T10:30:00Z_550e8400-e29b-41d4-a716-446655440000",
  "hasMore": true,
  "count": 50
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `logs` | array | Array of log entries |
| `nextCursor` | string | Cursor for fetching next page (empty if no more results) |
| `hasMore` | boolean | Whether more results are available |
| `count` | integer | Number of logs in current response |

### Log Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier for the log entry |
| `timestamp` | string | ISO 8601 timestamp when log was created |
| `level` | string | Log level (debug, info, warning, error, critical) |
| `category` | string | Log category |
| `message` | string | Main log message |
| `details` | string | Additional details (optional) |
| `source` | string | Source code location that generated the log |
| `requestId` | string | Request ID for tracing (optional) |
| `httpStatus` | string | HTTP status code if applicable (optional) |
| `ipAddress` | string | Client IP address (optional) |
| `userAgent` | string | Client user agent (optional) |

## Examples

### 1. Get all logs (default pagination)
```bash
curl "http://localhost:5000/api/developer/logs"
```

### 2. Filter by error and critical levels
```bash
curl "http://localhost:5000/api/developer/logs?level=error,critical"
```

### 3. Filter by multiple categories
```bash
curl "http://localhost:5000/api/developer/logs?category=portfolio,fund,transaction"
```

### 4. Filter by date range
```bash
curl "http://localhost:5000/api/developer/logs?startDate=2024-01-01T00:00:00Z&endDate=2024-01-31T23:59:59Z"
```

**Note**: You can also use date-only format:
```bash
curl "http://localhost:5000/api/developer/logs?startDate=2024-01-01&endDate=2024-01-31"
```

### 5. Filter by source (partial match)
```bash
curl "http://localhost:5000/api/developer/logs?source=Portfolio"
```

This will match any source containing "Portfolio", such as:
- `PortfolioHandler.GetPortfolio`
- `PortfolioService.CreatePortfolio`
- `PortfolioRepository.UpdatePortfolio`

### 6. Sort in ascending order (oldest first)
```bash
curl "http://localhost:5000/api/developer/logs?sortDir=asc"
```

### 7. Limit results per page
```bash
curl "http://localhost:5000/api/developer/logs?perPage=10"
```

### 8. Combined filters
```bash
curl "http://localhost:5000/api/developer/logs?level=error,critical&category=portfolio,fund&startDate=2024-01-01&source=Handler&sortDir=desc&perPage=25"
```

### 9. Pagination

**First page:**
```bash
curl "http://localhost:5000/api/developer/logs?perPage=10"
```

**Response:**
```json
{
  "logs": [...10 logs...],
  "nextCursor": "2024-01-15T10:30:00Z_550e8400-e29b-41d4-a716-446655440000",
  "hasMore": true,
  "count": 10
}
```

**Next page:**
```bash
curl "http://localhost:5000/api/developer/logs?perPage=10&cursor=2024-01-15T10:30:00Z_550e8400-e29b-41d4-a716-446655440000"
```

**Continue until `hasMore` is false**

### 10. Complex Query Example

Get error and critical logs from portfolio and transaction categories in the last month, sorted by newest first, 20 per page:

```bash
curl "http://localhost:5000/api/developer/logs?level=error,critical&category=portfolio,transaction&startDate=2024-01-01&endDate=2024-01-31&sortDir=desc&perPage=20"
```

## Error Responses

### 400 Bad Request - Invalid Parameters
```json
{
  "error": "Invalid filter parameters",
  "details": "invalid log level: invalid_level"
}
```

**Common validation errors:**
- `invalid log level: xyz`
- `invalid category: abc`
- `invalid startDate format: parsing time "..." as "2006-01-02": cannot parse "..." as "2006"`
- `invalid endDate format: ...`
- `invalid sortDir: must be 'asc' or 'desc'`
- `invalid perPage: must be between 1 and 100`
- `invalid perPage: must be a number`

### 500 Internal Server Error
```json
{
  "error": "Failed to retrieve logs",
  "details": "database connection failed"
}
```

## Filter Logic

### Multiple Values (OR Logic)
When you specify multiple values for the same filter type, they are combined with OR:

```bash
# Returns logs where level = 'error' OR level = 'critical'
?level=error,critical

# Returns logs where category = 'portfolio' OR category = 'fund'
?category=portfolio,fund
```

### Multiple Filter Types (AND Logic)
When you specify different filter types, they are combined with AND:

```bash
# Returns logs where (level = 'error' OR level = 'critical')
#                AND (category = 'portfolio' OR category = 'fund')
#                AND timestamp >= '2024-01-01'
?level=error,critical&category=portfolio,fund&startDate=2024-01-01
```

## Pagination Best Practices

1. **Always use perPage**: Set an appropriate page size for your use case (default is 50)
2. **Use cursor for next page**: Always use the `nextCursor` from the previous response
3. **Check hasMore**: Stop pagination when `hasMore` is false
4. **Maintain filter parameters**: When paginating, keep all filter parameters the same
5. **Combine cursor with filters**: Cursor works with all filter combinations

**Example pagination loop:**
```javascript
let cursor = null;
let hasMore = true;

while (hasMore) {
  let url = `/api/developer/logs?perPage=50&level=error,critical`;
  if (cursor) {
    url += `&cursor=${encodeURIComponent(cursor)}`;
  }

  const response = await fetch(url);
  const data = await response.json();

  // Process data.logs
  processlogs(data.logs);

  cursor = data.nextCursor;
  hasMore = data.hasMore;
}
```

## Performance Considerations

1. **Use specific filters**: More specific filters reduce the result set and improve performance
2. **Limit page size**: Don't request more than you need (max is 100)
3. **Index recommendations**: Ensure these database indexes exist:
   - `(timestamp, id)` for cursor pagination
   - `(level, category, timestamp)` for common filters
   - `source` for source filtering

## Security

- All parameters are validated and sanitized
- SQL injection prevention through parameterized queries
- Rate limiting recommended for production use
- Consider authentication/authorization before production deployment

## Common Use Cases

### Get latest errors
```bash
curl "http://localhost:5000/api/developer/logs?level=error,critical&sortDir=desc&perPage=20"
```

### Get all IBKR logs from today
```bash
curl "http://localhost:5000/api/developer/logs?category=ibkr&startDate=2024-01-21"
```

### Search for specific function/source errors
```bash
curl "http://localhost:5000/api/developer/logs?source=fetch_statement&level=error"
```

### Debug recent activity
```bash
curl "http://localhost:5000/api/developer/logs?sortDir=desc&perPage=50"
```

## Troubleshooting

### Not getting expected results?

Check that you're using:
1. ✅ **Singular** parameter names: `level`, `category` (not `levels`, `categories`)
2. ✅ **camelCase**: `sortDir`, `perPage`, `startDate`, `endDate`
3. ✅ **Comma-separated** lists: `level=error,warning`
4. ✅ **Valid values**: Check the "Valid Values" section above
5. ✅ **ISO 8601 dates**: `2024-01-21T00:00:00Z` or `2024-01-21`

### Empty results?

- Verify filter values are valid (see Valid Values section)
- Check date range includes data
- Try without filters to see if any logs exist

## Related Endpoints

- `DELETE /api/developer/logs` - Clear all logs (planned)
- `GET /api/developer/logs/stats` - Get log statistics (planned)
- `GET /api/developer/logs/{id}` - Get specific log entry (planned)
