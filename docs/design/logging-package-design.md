# HotPlex Logging Package Design Document

## 1. Package Overview

### Purpose
The `internal/logging` package provides a unified, production-ready logging abstraction built on Go's `log/slog`. It addresses the following issues identified in the current codebase:

| Issue | Current State | Target State |
|-------|---------------|--------------|
| **Field naming** | Mixed conventions (session_id, SessionID, sessionId) | Consistent snake_case |
| **Context propagation** | Manual field passing at each call site | Centralized LogContext |
| **Sensitivity masking** | None | Multi-level data masking |
| **Float precision** | Default float64 formatting | Configurable precision |
| **Duplicate logs** | Ad-hoc logging without deduplication | Structured log levels |

### Goals
- **Unified API**: Single logger interface across all packages
- **Context isolation**: LogContext carries session/platform/user metadata
- **Security compliance**: Built-in PII/Sensitive data masking
- **Performance**: Zero-allocation path for high-frequency logging
- **Observability**: Native OpenTelemetry attribute integration

---

## 2. Package Structure

```
internal/logging/
├── context.go          # LogContext type and context key definitions
├── logger.go           # Core Logger type and construction
├── mask.go             # Sensitivity levels and masking logic
├── fields.go           # Standard field names and validators
├── formatters.go       # Custom slog.Value handlers (float, duration)
├── context_test.go
├── logger_test.go
├── mask_test.go
├── fields_test.go
└── doc.go              # Package documentation
```

---

## 3. Core Components

### 3.1 Context Keys (context.go)

```go
package logging

import "context"

// ContextKey defines typed keys for log context storage
type ContextKey string

const (
    // Session-level context
    KeySessionID ContextKey = "session_id"        // Persistent session identifier
    KeyProviderSessionID ContextKey = "provider_session_id" // CLI internal session ID
    
    // User/Channel context
    KeyUserID ContextKey = "user_id"             // Platform user identifier
    KeyChannelID ContextKey = "channel_id"       // Platform channel identifier
    
    // Platform context
    KeyPlatform ContextKey = "platform"          // slack, discord, telegram, etc.
    KeyNamespace ContextKey = "namespace"         // Execution namespace
    
    // Request context
    KeyRequestID ContextKey = "request_id"       // Correlation ID
    KeyTraceID ContextKey = "trace_id"          // OpenTelemetry trace ID
    
    // Performance context
    KeyDuration ContextKey = "duration_ms"      // Operation duration
    KeyLatency ContextKey = "latency_ms"         // Request latency
)
```

### 3.2 LogContext (context.go)

```go
package logging

import (
    "context"
    "time"
)

// LogContext carries unified metadata for logging operations.
// It serves as the single source of truth for log fields.
type LogContext struct {
    // Core identifiers
    SessionID          string `json:"session_id,omitempty"`
    ProviderSessionID  string `json:"provider_session_id,omitempty"`
    
    // Platform context
    Platform           string `json:"platform,omitempty"`
    Namespace          string `json:"namespace,omitempty"`
    
    // User/Channel
    UserID             string `json:"user_id,omitempty"`
    ChannelID          string `json:"channel_id,omitempty"`
    
    // Correlation
    RequestID          string `json:"request_id,omitempty"`
    TraceID            string `json:"trace_id,omitempty"`
    
    // Sensitivity level for this context
    Sensitivity        SensitivityLevel `json:"-"`
    
    // Timestamp when context was created
    CreatedAt          time.Time `json:"created_at,omitempty"`
}

// NewLogContext creates a new LogContext with sensible defaults
func NewLogContext() *LogContext {
    return &LogContext{
        CreatedAt: time.Now(),
        Sensitivity: LevelNone,
    }
}

// ToAttrs converts LogContext to slog.Attr slice for logging
// Uses snake_case field names throughout
func (c *LogContext) ToAttrs() []any {
    attrs := make([]any, 0, 12)
    
    if c.SessionID != "" {
        attrs = append(attrs, "session_id", c.SessionID)
    }
    if c.ProviderSessionID != "" {
        attrs = append(attrs, "provider_session_id", c.ProviderSessionID)
    }
    if c.Platform != "" {
        attrs = append(attrs, "platform", c.Platform)
    }
    if c.Namespace != "" {
        attrs = append(attrs, "namespace", c.Namespace)
    }
    if c.UserID != "" {
        attrs = append(attrs, "user_id", c.UserID)
    }
    if c.ChannelID != "" {
        attrs = append(attrs, "channel_id", c.ChannelID)
    }
    if c.RequestID != "" {
        attrs = append(attrs, "request_id", c.RequestID)
    }
    if c.TraceID != "" {
        attrs = append(attrs, "trace_id", c.TraceID)
    }
    
    return attrs
}

// WithSessionID returns a new LogContext with SessionID set
func (c *LogContext) WithSessionID(id string) *LogContext {
    copy := *c
    copy.SessionID = id
    return &copy
}

// WithPlatform returns a new LogContext with Platform set
func (c *LogContext) WithPlatform(platform string) *LogContext {
    copy := *c
    copy.Platform = platform
    return &copy
}

// WithUserID returns a new LogContext with UserID set (applies masking if needed)
func (c *LogContext) WithUserID(id string) *LogContext {
    copy := *c
    copy.UserID = MaskValue(id, copy.Sensitivity)
    return &copy
}

// WithChannelID returns a new LogContext with ChannelID set
func (c *LogContext) WithChannelID(id string) *LogContext {
    copy := *c
    copy.ChannelID = id
    return &copy
}

// WithRequestID returns a new LogContext with RequestID set
func (c *LogContext) WithRequestID(id string) *LogContext {
    copy := *c
    copy.RequestID = id
    return &copy
}

// WithSensitivity returns a new LogContext with Sensitivity level set
func (c *LogContext) WithSensitivity(level SensitivityLevel) *LogContext {
    copy := *c
    copy.Sensitivity = level
    // Re-apply masking to sensitive fields
    copy.UserID = MaskValue(copy.UserID, level)
    return &copy
}

// ContextWithLogContext stores LogContext in go context
func ContextWithLogContext(ctx context.Context, lc *LogContext) context.Context {
    return context.WithValue(ctx, contextKeyLogContext{}, lc)
}

// LogContextFromContext retrieves LogContext from go context
func LogContextFromContext(ctx context.Context) (*LogContext, bool) {
    lc, ok := ctx.Value(contextKeyLogContext{}).(*LogContext)
    return lc, ok
}

// contextKeyLogContext is a private type to avoid collisions
type contextKeyLogContext struct{}
```

### 3.3 Logger Type (logger.go)

```go
package logging

import (
    "context"
    "log/slog"
    "time"
)

// Logger wraps slog.Logger with HotPlex-specific functionality.
// It provides structured logging with automatic context propagation.
type Logger struct {
    inner       *slog.Logger
    defaultCtx  *LogContext
    floatFormat FloatFormat
    maskEnabled bool
}

// FloatFormat defines how floating point numbers are formatted
type FloatFormat int

const (
    // FloatRaw outputs raw float64 (may have precision issues)
    FloatRaw FloatFormat = iota
    // FloatPrecise outputs with 2 decimal precision
    FloatPrecise
    // FloatScientific outputs in scientific notation
    FloatScientific
)

// LoggerOption configures Logger behavior
type LoggerOption func(*Logger)

// WithFloatFormat sets the float formatting behavior
func WithFloatFormat(f FloatFormat) LoggerOption {
    return func(l *Logger) { l.floatFormat = f }
}

// WithSensitivity enables sensitivity masking
func WithSensitivity(level SensitivityLevel) LoggerOption {
    return func(l *Logger) {
        l.maskEnabled = true
        if l.defaultCtx == nil {
            l.defaultCtx = NewLogContext()
        }
        l.defaultCtx.Sensitivity = level
    }
}

// WithDefaultContext sets a default LogContext for all log calls
func WithDefaultContext(ctx *LogContext) LoggerOption {
    return func(l *Logger) { l.defaultCtx = ctx }
}

// NewLogger creates a new Logger wrapping slog.Logger
func NewLogger(inner *slog.Logger, opts ...LoggerOption) *Logger {
    l := &Logger{
        inner:       inner,
        defaultCtx:  NewLogContext(),
        floatFormat: FloatPrecise,
        maskEnabled: false,
    }
    
    for _, opt := range opts {
        opt(l)
    }
    
    return l
}

// Default returns a Logger wrapping slog.Default()
func Default() *Logger {
    return NewLogger(slog.Default())
}

// With creates a child logger with additional context
func (l *Logger) With(ctx *LogContext) *Logger {
    // Merge contexts: child inherits parent defaults but can override
    child := &Logger{
        inner:       l.inner,
        defaultCtx:  l.defaultCtx,
        floatFormat: l.floatFormat,
        maskEnabled: l.maskEnabled,
    }
    
    if ctx != nil {
        // Child context takes precedence
        child.defaultCtx = l.defaultCtx
        if ctx.SessionID != "" {
            child.defaultCtx.SessionID = ctx.SessionID
        }
        if ctx.Platform != "" {
            child.defaultCtx.Platform = ctx.Platform
        }
        if ctx.UserID != "" {
            child.defaultCtx.UserID = MaskValue(ctx.UserID, child.defaultCtx.Sensitivity)
        }
        if ctx.ChannelID != "" {
            child.defaultCtx.ChannelID = ctx.ChannelID
        }
        if ctx.RequestID != "" {
            child.defaultCtx.RequestID = ctx.RequestID
        }
        if ctx.TraceID != "" {
            child.defaultCtx.TraceID = ctx.TraceID
        }
    }
    
    return child
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...any) {
    attrs := l.buildAttrs(args...)
    l.inner.Debug(msg, attrs...)
}

// DebugContext logs at debug level with explicit LogContext
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
    lc := l.resolveContext(ctx)
    attrs := l.mergeAttrs(lc, args...)
    l.inner.DebugContext(ctx, msg, attrs...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...any) {
    attrs := l.buildAttrs(args...)
    l.inner.Info(msg, attrs...)
}

// InfoContext logs at info level with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
    lc := l.resolveContext(ctx)
    attrs := l.mergeAttrs(lc, args...)
    l.inner.InfoContext(ctx, msg, attrs...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...any) {
    attrs := l.buildAttrs(args...)
    l.inner.Warn(msg, attrs...)
}

// WarnContext logs at warn level with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
    lc := l.resolveContext(ctx)
    attrs := l.mergeAttrs(lc, args...)
    l.inner.WarnContext(ctx, msg, attrs...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...any) {
    attrs := l.buildAttrs(args...)
    l.inner.Error(msg, attrs...)
}

// ErrorContext logs at error level with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
    lc := l.resolveContext(ctx)
    attrs := l.mergeAttrs(lc, args...)
    l.inner.ErrorContext(ctx, msg, attrs...)
}

// Log logs at the specified level
func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
    lc := l.resolveContext(ctx)
    attrs := l.mergeAttrs(lc, args...)
    l.inner.Log(ctx, level, msg, attrs...)
}

// buildAttrs converts args to slog.Attr slice, applying float formatting
func (l *Logger) buildAttrs(args ...any) []any {
    if len(args) == 0 {
        return nil
    }
    
    attrs := make([]any, 0, len(args)+len(l.defaultCtx.ToAttrs()))
    
    // Prepend default context attrs
    attrs = append(attrs, l.defaultCtx.ToAttrs()...)
    
    // Process args with float handling
    for i := 0; i < len(args); i++ {
        key, ok := args[i].(string)
        if !ok {
            // Non-string key: convert to string
            attrs = append(args, "key", args[i])
            continue
        }
        attrs = append(attrs, key)
        
        if i+1 < len(args) {
            val := args[i+1]
            attrs = append(attrs, l.formatValue(val))
            i++
        }
    }
    
    return attrs
}

// mergeAttrs merges LogContext with additional args
func (l *Logger) mergeAttrs(lc *LogContext, args ...any) []any {
    // Start with LogContext attrs
    attrs := lc.ToAttrs()
    
    // Merge with additional args
    for i := 0; i < len(args); i++ {
        key, ok := args[i].(string)
        if !ok {
            continue
        }
        attrs = append(attrs, key)
        
        if i+1 < len(args) {
            val := args[i+1]
            // Apply sensitivity masking to known sensitive fields
            if l.maskEnabled {
                val = MaskValueByKey(key, val, lc.Sensitivity)
            }
            attrs = append(attrs, l.formatValue(val))
            i++
        }
    }
    
    return attrs
}

// resolveContext extracts LogContext from context or returns default
func (l *Logger) resolveContext(ctx context.Context) *LogContext {
    if lc, ok := LogContextFromContext(ctx); ok {
        return lc
    }
    return l.defaultCtx
}

// formatValue applies float formatting if needed
func (l *Logger) formatValue(v any) any {
    switch val := v.(type) {
    case float64:
        return l.formatFloat(val)
    case float32:
        return l.formatFloat(float64(val))
    case time.Duration:
        // Always output as milliseconds for consistency
        return float64(val.Milliseconds())
    default:
        return v
    }
}

// formatFloat applies configured float formatting
func (l *Logger) formatFloat(f float64) float64 {
    switch l.floatFormat {
    case FloatPrecise:
        // Round to 2 decimal places
        return float64(int64(f*100)) / 100
    case FloatScientific:
        // Return as-is (slog handles it)
        return f
    default:
        return f
    }
}

// Handler returns the underlying slog.Handler for composition
func (l *Logger) Handler() slog.Handler {
    return l.inner.Handler()
}

// Level returns the logger's minimum level
func (l *Logger) Level() slog.Level {
    return slog.LevelInfo // Could be extended to track actual level
}

// Underlying returns the raw *slog.Logger
func (l *Logger) Underlying() *slog.Logger {
    return l.inner
}
```

### 3.4 Sensitivity Levels & Masking (mask.go)

```go
package logging

// SensitivityLevel defines data sensitivity classification
type SensitivityLevel int

const (
    // LevelNone - no masking applied
    LevelNone SensitivityLevel = iota
    // LevelLow - mask only credentials/secrets
    LevelLow
    // LevelMedium - mask user identifiers and PII
    LevelMedium
    // LevelHigh - mask all potentially sensitive data
    LevelHigh
)

// sensitiveKeys defines field names that contain sensitive data
var sensitiveKeys = map[string]SensitivityLevel{
    // Credentials
    "password":           LevelLow,
    "passwd":             LevelLow,
    "secret":             LevelLow,
    "token":              LevelLow,
    "api_key":            LevelLow,
    "apikey":             LevelLow,
    "access_token":       LevelLow,
    "refresh_token":      LevelLow,
    "private_key":        LevelLow,
    
    // User identifiers (PII)
    "user_id":            LevelMedium,
    "userid":             LevelMedium,
    "email":              LevelMedium,
    "phone":              LevelMedium,
    "mobile":             LevelMedium,
    "ip_address":         LevelMedium,
    "ip":                 LevelMedium,
    
    // Platform-specific
    "session_token":      LevelMedium,
    "oauth_token":       LevelMedium,
    
    // High sensitivity
    "credit_card":        LevelHigh,
    "ssn":                LevelHigh,
    "bank_account":       LevelHigh,
}

// MaskValue masks a value based on sensitivity level
func MaskValue(value string, level SensitivityLevel) string {
    if value == "" || level == LevelNone {
        return value
    }
    
    switch level {
    case LevelLow:
        return maskCredential(value)
    case LevelMedium:
        return maskMedium(value)
    case LevelHigh:
        return maskHigh(value)
    default:
        return value
    }
}

// MaskValueByKey masks a value based on its key and sensitivity level
func MaskValueByKey(key string, value any, level SensitivityLevel) any {
    if level == LevelNone {
        return value
    }
    
    // Check if key is in sensitive map
    if sl, ok := sensitiveKeys[key]; ok && sl <= level {
        if str, ok := value.(string); ok {
            return MaskValue(str, level)
        }
    }
    
    return value
}

// maskCredential masks credentials, showing first 4 and last 4 chars
func maskCredential(s string) string {
    if len(s) <= 8 {
        return "****"
    }
    return s[:4] + "****" + s[len(s)-4:]
}

// maskMedium masks PII, showing first 2 and last 2 chars
func maskMedium(s string) string {
    if len(s) <= 4 {
        return "****"
    }
    if len(s) <= 8 {
        return "**" + s[len(s)-2:]
    }
    return s[:2] + "****" + s[len(s)-2:]
}

// maskHigh masks everything except first char
func maskHigh(s string) string {
    if len(s) <= 1 {
        return "*"
    }
    return s[0] + "****"
}

// IsSensitiveKey returns true if the key contains sensitive data
func IsSensitiveKey(key string) bool {
    _, ok := sensitiveKeys[key]
    return ok
}

// GetSensitivityLevel returns the sensitivity level for a key
func GetSensitivityLevel(key string) SensitivityLevel {
    if level, ok := sensitiveKeys[key]; ok {
        return level
    }
    return LevelNone
}
```

### 3.5 Standard Fields (fields.go)

```go
package logging

// Standard field names for consistent logging across HotPlex.
// All field names use snake_case as per project convention.
const (
    // Session fields
    FieldSessionID          = "session_id"
    FieldProviderSessionID  = "provider_session_id"
    
    // User/Channel fields
    FieldUserID             = "user_id"
    FieldChannelID          = "channel_id"
    FieldThreadID           = "thread_id"
    
    // Platform fields
    FieldPlatform           = "platform"
    FieldNamespace          = "namespace"
    
    // Request/Trace fields
    FieldRequestID          = "request_id"
    FieldTraceID            = "trace_id"
    FieldSpanID             = "span_id"
    
    // Performance fields
    FieldDuration           = "duration_ms"
    FieldLatency            = "latency_ms"
    FieldProcessingTime     = "processing_time_ms"
    FieldQueueTime          = "queue_time_ms"
    
    // Operation fields
    FieldOperation          = "operation"
    FieldAction             = "action"
    FieldEventType          = "event_type"
    FieldError              = "error"
    FieldErrorCode          = "error_code"
    FieldReason             = "reason"
    
    // Content fields
    FieldContentLength     = "content_length"
    FieldContentType       = "content_type"
    FieldMessageLength     = "message_length"
    
    // Result fields
    FieldSuccess            = "success"
    FieldStatusCode        = "status_code"
    FieldResultCount       = "result_count"
)

// Common field value constructors for type safety

// SessionFields creates a slice of session-related fields
func SessionFields(sessionID, providerSessionID string) []any {
    return []any{
        FieldSessionID, sessionID,
        FieldProviderSessionID, providerSessionID,
    }
}

// UserChannelFields creates a slice of user/channel fields
func UserChannelFields(userID, channelID string) []any {
    return []any{
        FieldUserID, userID,
        FieldChannelID, channelID,
    }
}

// PlatformFields creates a slice of platform fields
func PlatformFields(platform, namespace string) []any {
    return []any{
        FieldPlatform, platform,
        FieldNamespace, namespace,
    }
}

// TraceFields creates a slice of tracing fields
func TraceFields(traceID, spanID string) []any {
    return []any{
        FieldTraceID, traceID,
        FieldSpanID, spanID,
    }
}

// PerformanceFields creates a slice of performance fields
func PerformanceFields(durationMs int64) []any {
    return []any{
        FieldDuration, durationMs,
    }
}

// ErrorFields creates a slice of error fields
func ErrorFields(err error) []any {
    if err == nil {
        return nil
    }
    return []any{
        FieldError, err.Error(),
    }
}

// ResultFields creates a slice of result fields
func ResultFields(success bool, statusCode int) []any {
    return []any{
        FieldSuccess, success,
        FieldStatusCode, statusCode,
    }
}
```

### 3.6 Custom Formatters (formatters.go)

```go
package logging

import (
    "fmt"
    "log/slog"
    "math"
    "reflect"
    "time"
)

// Float64Handler is a custom slog.Value handler for float64 with precision control
type Float64Handler struct {
    Precision int
}

func (h Float64Handler) Handle(attr slog.Attr) slog.Attr {
    if attr.Value.Kind() != slog.KindFloat64 {
        return attr
    }
    
    f := attr.Value.Float64()
    if math.IsNaN(f) || math.IsInf(f, 0) {
        return attr
    }
    
    // Apply precision
    precision := h.Precision
    if precision < 0 {
        precision = 2
    }
    
    // Round to specified precision
    multiplier := math.Pow10(precision)
    rounded := math.Round(f*multiplier) / multiplier
    
    return slog.Attr{
        Key: attr.Key,
        Value: slog.Float64Value(rounded),
    }
}

// DurationMillisHandler formats duration as milliseconds
type DurationMillisHandler struct{}

func (h DurationMillisHandler) Handle(attr slog.Attr) slog.Attr {
    if attr.Value.Kind() != slog.KindDuration {
        return attr
    }
    
    ms := attr.Value.Duration().Milliseconds()
    return slog.Attr{
        Key: attr.Key,
        Value: slog.Int64Value(ms),
    }
}

// SensitiveStringHandler masks sensitive string values
type SensitiveStringHandler struct {
    Level SensitivityLevel
}

func (h SensitiveStringHandler) Handle(attr slog.Attr) slog.Attr {
    if attr.Value.Kind() != slog.KindString {
        return attr
    }
    
    masked := MaskValue(attr.Value.String(), h.Level)
    return slog.Attr{
        Key: attr.Key,
        Value: slog.StringValue(masked),
    }
}

// PrettyTimeHandler formats time as ISO8601
type PrettyTimeHandler struct{}

func (h PrettyTimeHandler) Handle(attr slog.Attr) slog.Attr {
    if attr.Value.Kind() != slog.KindTime {
        return attr
    }
    
    formatted := attr.Value.Time().Format(time.RFC3339)
    return slog.Attr{
        Key: attr.Key,
        Value: slog.StringValue(formatted),
    }
}

// CustomJSONHandler creates a JSON handler with custom formatters
func CustomJSONHandler(opts *slog.HandlerOptions) slog.Handler {
    return slog.NewJSONHandler(nil, opts)
}

// FormatAny attempts to format any value as a loggable type
func FormatAny(v any) any {
    if v == nil {
        return nil
    }
    
    switch v.(type) {
    case string, int, int8, int16, int32, int64:
        return v
    case uint, uint8, uint16, uint32, uint64:
        return v
    case float32:
        return float64(v.(float32))
    case float64:
        f := v.(float64)
        // Apply 2 decimal precision
        return math.Round(f*100) / 100
    case bool:
        return v
    case time.Duration:
        return v.(time.Duration).Milliseconds()
    case error:
        return v.(error).Error()
    default:
        // For complex types, use reflection
        rv := reflect.ValueOf(v)
        if rv.Kind() == reflect.Ptr {
            rv = rv.Elem()
        }
        
        switch rv.Kind() {
        case reflect.Struct:
            return fmt.Sprintf("%+v", v)
        case reflect.Slice, reflect.Map:
            return fmt.Sprintf("%v", v)
        default:
            return fmt.Sprintf("%v", v)
        }
    }
}
```

---

## 4. Usage Examples

### 4.1 Basic Logger Initialization

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/hrygo/hotplex/internal/logging"
)

func main() {
    // Configure JSON logger with debug level
    logger := logging.NewLogger(slog.New(
        slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        }),
    ))
    
    // Or use default
    defaultLogger := logging.Default()
    
    defaultLogger.Info("application started", 
        "version", "1.0.0",
    )
}
```

### 4.2 Session-Based Logging

```go
func handleSessionRequest(ctx context.Context, sessionID, platform string) {
    logger := logging.Default()
    
    // Create context with session info
    lc := logging.NewLogContext().
        WithSessionID(sessionID).
        WithPlatform(platform).
        WithSensitivity(logging.LevelMedium)
    
    // Attach to Go context
    ctx = logging.ContextWithLogContext(ctx, lc)
    
    // Use context-based logging
    logger.InfoContext(ctx, "session request received",
        "action", "session_start",
    )
    
    // Or create a derived logger
    sessionLogger := logger.With(lc)
    sessionLogger.Info("processing session")
}
```

### 4.3 User/Channel Context

```go
func handleChatMessage(ctx context.Context, userID, channelID, content string) {
    logger := logging.Default()
    
    lc := logging.NewLogContext().
        WithUserID(userID).
        WithChannelID(channelID).
        WithPlatform("slack").
        WithSensitivity(logging.LevelMedium)
    
    ctx = logging.ContextWithLogContext(ctx, lc)
    
    // UserID is automatically masked at LevelMedium
    logger.InfoContext(ctx, "message received",
        logging.FieldContentLength, len(content),
    )
    // Output: {"level":"INFO","msg":"message received","user_id":"us****","channel_id":"C123","content_length":42}
}
```

### 4.4 Error Handling with Context

```go
func processWithErrorHandling(ctx context.Context, sessionID string) error {
    logger := logging.Default()
    
    lc := logging.NewLogContext().
        WithSessionID(sessionID).
        WithSensitivity(logging.LevelLow)
    
    ctx = logging.ContextWithLogContext(ctx, lc)
    
    result, err := doWork(ctx)
    if err != nil {
        logger.ErrorContext(ctx, "work failed",
            logging.FieldError, err,
            logging.FieldReason, "database_timeout",
        )
        return err
    }
    
    logger.InfoContext(ctx, "work completed",
        logging.FieldResultCount, len(result),
    )
    
    return nil
}
```

### 4.5 Performance Logging

```go
func logPerformance(logger *logging.Logger, operation string, startTime time.Time) {
    duration := time.Since(startTime).Milliseconds()
    
    logger.Info("operation completed",
        logging.FieldOperation, operation,
        logging.FieldDuration, duration,
    )
}
```

### 4.6 With Engine Integration

```go
package main

import (
    "github.com/hrygo/hotplex"
    "github.com/hrygo/hotplex/internal/logging"
)

func main() {
    // Initialize logging package
    logger := logging.NewLogger(
        hotplex.Logger, // Use engine's logger
        logging.WithSensitivity(logging.LevelMedium),
        logging.WithFloatFormat(logging.FloatPrecise),
    )
    
    // Create session context
    lc := logging.NewLogContext().
        WithSessionID("session-123").
        WithPlatform("discord").
        WithUserID("user-456")
    
    // Attach to context
    ctx := logging.ContextWithLogContext(context.Background(), lc)
    
    // Execute with context-aware logging
    err := engine.Execute(ctx, config, prompt, func(eventType string, data any) error {
        logger.InfoContext(ctx, "event received",
            "event_type", eventType,
        )
        return nil
    })
}
```

---

## 5. Migration Guide

### 5.1 Current Patterns to Avoid

| Pattern | Problem | Migration |
|---------|---------|-----------|
| `log.Printf("session: %s", id)` | Unstructured text | Use `logger.Info("session started", "session_id", id)` |
| `slog.Info("msg", "SessionID", id)` | PascalCase field | Use `slog.Info("msg", "session_id", id)` |
| `logger.Info("msg", "float_value", float64(1.23456))` | Uncontrolled precision | Use `FloatPrecise` format |
| `logger.Info("msg", "user", userID)` | PII exposure | Use `MaskValue()` or `SensitivityLevel` |
| Manual field passing | Duplication | Use `LogContext` |

### 5.2 Step-by-Step Migration

#### Step 1: Replace slog imports
```go
// Before
import "log/slog"

// After
import "github.com/hrygo/hotplex/internal/logging"
```

#### Step 2: Update field names to snake_case
```go
// Before
logger.Info("session started", "SessionID", sess.ID)

// After
logger.Info("session started", "session_id", sess.ID)
```

#### Step 3: Create LogContext for repeated fields
```go
// Before
logger.Info("processing", "session_id", id, "user_id", user, "channel_id", ch)
logger.Info("sending", "session_id", id, "user_id", user, "channel_id", ch)

// After
lc := logging.NewLogContext().
    WithSessionID(id).
    WithUserID(user).
    WithChannelID(ch)

logger = logger.With(lc)
logger.Info("processing")
logger.Info("sending")
```

#### Step 4: Add sensitivity masking
```go
// Before
logger.Info("user action", "user_id", userID, "content", content)

// After
lc := logging.NewLogContext().
    WithUserID(userID).
    WithSensitivity(logging.LevelMedium)

logger.With(lc).Info("user action", "content", content)
// Output: user_id is masked
```

#### Step 5: Use standard field constants
```go
// Before
logger.Info("completed", "duration", durationMs, "error", err)

// After
logger.Info("completed", 
    logging.FieldDuration, durationMs,
    logging.FieldError, err,
)
```

### 5.3 Code Change Examples

#### engine/runner.go
```go
// Before
func (e *Engine) Execute(...) error {
    e.logger.Info("Starting session", "session_id", cfg.SessionID)
    // ...
}

// After
func (e *Engine) Execute(...) error {
    lc := logging.NewLogContext().
        WithSessionID(cfg.SessionID).
        WithPlatform(cfg.Platform)
    
    e.logger.With(lc).Info("Starting session")
    // ...
}
```

#### chatapps/engine_handler.go
```go
// Before
c.logger.Info("SendAggregatedMessage called", 
    "session_id", c.sessionID, 
    "content_len", len(msg.Content))

// After
lc := logging.NewLogContext().
    WithSessionID(c.sessionID)

c.logger.With(lc).Info("SendAggregatedMessage called",
    logging.FieldContentLength, len(msg.Content))
```

#### internal/server/hotplex_ws.go
```go
// Before
h.logger.Info("Stop request", "session_id", req.SessionID, "reason", reason)

// After
lc := logging.NewLogContext().
    WithSessionID(req.SessionID)

h.logger.With(lc).Info("Stop request",
    logging.FieldReason, reason)
```

---

## 6. Implementation Checklist

- [ ] Create `internal/logging/context.go`
- [ ] Create `internal/logging/logger.go`
- [ ] Create `internal/logging/mask.go`
- [ ] Create `internal/logging/fields.go`
- [ ] Create `internal/logging/formatters.go`
- [ ] Create `internal/logging/doc.go`
- [ ] Write unit tests for each file
- [ ] Update `engine/runner.go` to use new package
- [ ] Update `chatapps/engine_handler.go` to use new package
- [ ] Update `internal/server/*.go` to use new package
- [ ] Run `go build ./...` to verify
- [ ] Run `go test -race ./internal/logging/...`

---

## 7. Appendix: Field Reference

### Session Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `session_id` | string | Persistent session identifier | Low |
| `provider_session_id` | string | CLI internal session ID | Low |

### User/Channel Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `user_id` | string | Platform user identifier | Medium |
| `channel_id` | string | Platform channel identifier | Low |
| `thread_id` | string | Thread/message thread ID | Low |

### Platform Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `platform` | string | slack, discord, telegram, etc. | Low |
| `namespace` | string | Execution namespace | Low |

### Tracing Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `request_id` | string | Correlation ID | Low |
| `trace_id` | string | OpenTelemetry trace ID | Low |
| `span_id` | string | OpenTelemetry span ID | Low |

### Performance Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `duration_ms` | int64 | Operation duration in milliseconds | Low |
| `latency_ms` | int64 | Request latency in milliseconds | Low |
| `processing_time_ms` | int64 | Processing time in milliseconds | Low |
| `queue_time_ms` | int64 | Queue wait time in milliseconds | Low |

### Operation Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `operation` | string | Operation name | Low |
| `action` | string | Action being performed | Low |
| `event_type` | string | Event type identifier | Low |
| `error` | string | Error message | Low |
| `error_code` | string | Error code | Low |
| `reason` | string | Reason for action | Low |

### Content Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `content_length` | int | Content byte length | Low |
| `content_type` | string | MIME type | Low |
| `message_length` | int | Message character count | Low |

### Result Fields
| Field | Type | Description | Sensitivity |
|-------|------|-------------|-------------|
| `success` | bool | Operation success status | Low |
| `status_code` | int | HTTP/response status code | Low |
| `result_count` | int | Number of results | Low |
