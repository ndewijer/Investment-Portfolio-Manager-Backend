package logging

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// DBHandler is a dual-write slog.Handler that writes structured logs to both
// stderr (console) and the database log table.
type DBHandler struct {
	db      *sql.DB
	console slog.Handler // TextHandler for stderr
	enabled *atomic.Bool
	level   *atomic.Int32 // slog.Level stored as int32
	attrs   []slog.Attr   // pre-bound attributes from WithAttrs
	groups  []string      // pre-bound groups from WithGroup
}

// NewDBHandler creates a DBHandler with defaults (enabled=true, level=INFO).
func NewDBHandler(db *sql.DB) *DBHandler {
	enabled := &atomic.Bool{}
	enabled.Store(true)
	level := &atomic.Int32{}
	level.Store(int32(slog.LevelInfo))

	return &DBHandler{
		db: db,
		console: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:       slog.LevelDebug, // console doesn't filter; our Enabled() gates
			ReplaceAttr: ReplaceLevel,
		}),
		enabled: enabled,
		level:   level,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *DBHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.Level(h.level.Load())
}

// Handle writes the log record to console (always) and database (if enabled).
//
//nolint:funlen // Dual-write logic with DB fallback needs the length.
func (h *DBHandler) Handle(ctx context.Context, record slog.Record) error {
	// Always write to console — the console handler has pre-bound attrs from WithAttrs.
	_ = h.console.Handle(ctx, record) //nolint:errcheck // Console write is best-effort; nothing to do on failure.

	// Only write to DB if enabled.
	if !h.enabled.Load() {
		return nil
	}

	// Extract category from pre-bound attrs (default "SYSTEM").
	// IMPORTANT: "category" and "status" are reserved attr keys — they map
	// to dedicated DB columns. If caller code passes either as a log arg
	// (e.g. devLog.Debug("msg", "category", value)), the record-level attr
	// overrides the pre-bound value. Use prefixed keys instead (e.g.
	// "filter_category", "response_status").
	category := "SYSTEM"
	var httpStatus *int
	var details []string

	for _, a := range h.attrs {
		if a.Key == "category" {
			category = strings.ToUpper(a.Value.String())
			continue
		}
		if a.Key == "status" {
			if v, ok := a.Value.Any().(int); ok {
				httpStatus = &v
			}
			continue
		}
		details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
	}

	// Also check record attrs for category/status; collect everything else as details.
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "category" {
			category = strings.ToUpper(a.Value.String())
			return true
		}
		if a.Key == "status" {
			if v, ok := a.Value.Any().(int); ok {
				httpStatus = &v
			}
			return true
		}
		details = append(details, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
		return true
	})

	// Request metadata from context.
	requestID := RequestIDFromContext(ctx)
	ipAddress := IPFromContext(ctx)
	userAgent := UserAgentFromContext(ctx)

	// Source from record.PC.
	source := ""
	if record.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := frames.Next()
		if f.Function != "" {
			parts := strings.Split(f.Function, "/")
			source = parts[len(parts)-1]
		}
	}

	// Stack trace for ERROR and CRITICAL.
	var stackTrace string
	if record.Level >= slog.LevelError {
		stackTrace = captureStackTrace(4)
	}

	// INSERT with a 2s timeout; on failure fall back to stderr.
	dbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := h.db.ExecContext(dbCtx,
		`INSERT INTO log (id, timestamp, level, category, message, details, source, request_id, stack_trace, http_status, ip_address, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(),
		record.Time.UTC().Format("2006-01-02 15:04:05"),
		SlogLevelToDBString(record.Level),
		category,
		record.Message,
		strings.Join(details, "; "),
		source,
		requestID,
		stackTrace,
		httpStatus, // http_status — extracted from "status" attr if present
		ipAddress,
		userAgent,
	)
	if err != nil {
		// Fall back silently — the console already has the log line.
		_, _ = fmt.Fprintf(os.Stderr, "logging: DB write failed: %v\n", err)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes pre-bound.
// The new handler shares db, enabled, and level with the parent.
func (h *DBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)

	return &DBHandler{
		db:      h.db,
		console: h.console.WithAttrs(attrs),
		enabled: h.enabled, // shared pointer
		level:   h.level,   // shared pointer
		attrs:   newAttrs,
		groups:  h.groups,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *DBHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)

	return &DBHandler{
		db:      h.db,
		console: h.console.WithGroup(name),
		enabled: h.enabled,
		level:   h.level,
		attrs:   h.attrs,
		groups:  newGroups,
	}
}

// SetEnabled enables or disables database logging at runtime.
func (h *DBHandler) SetEnabled(v bool) {
	h.enabled.Store(v)
}

// SetLevel sets the minimum log level at runtime.
func (h *DBHandler) SetLevel(level slog.Level) {
	h.level.Store(int32(level)) //nolint:gosec // G115: slog levels are small constants (-4 to 12); overflow impossible.
}

// captureStackTrace returns a formatted stack trace, skipping `skip` frames.
func captureStackTrace(skip int) string {
	const maxFrames = 32
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return ""
	}
	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)

	var b strings.Builder
	for {
		f, more := frames.Next()
		fmt.Fprintf(&b, "%s\n\t%s:%d\n", f.Function, f.File, f.Line)
		if !more {
			break
		}
	}
	return b.String()
}
