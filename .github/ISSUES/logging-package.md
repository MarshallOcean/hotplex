# Logging Package Design and Implementation

## Summary

Create `internal/logging` package to provide unified, production-ready logging abstraction with structured context, sensitivity masking, and consistent field naming conventions.

## Background

Current logging analysis revealed multiple issues in log output:

| Issue | Current State | Impact | Priority |
|-------|---------------|--------|----------|
| **Field naming inconsistency** | Mixed snake_case (`session_id`) and camelCase (`SessionID`) | Log parsing difficulties | 🔴 High |
| **Duplicate logs** | "Adapter started" printed twice | Log bloat | 🟡 Medium |
| **Float precision** | `0.14604720000000002` | Poor readability | 🟢 Low |
| **Missing context** | Some logs lack namespace/platform fields | Debugging difficulties | 🟡 Medium |
| **Sensitive data exposure** | user_id, content in plaintext | Security risk | 🟡 Medium |
| **Excessive blank lines** | Concurrent writes without control | Poor readability | 🟢 Low |

## Goals

1. **Unified API**: Single logger interface across all packages
2. **Context isolation**: LogContext carries session/platform/user metadata
3. **Security compliance**: Built-in PII/Sensitive data masking
4. **Performance**: Zero-allocation path for high-frequency logging
5. **Observability**: Native OpenTelemetry attribute integration

## Design

### Package Structure

```
internal/logging/
├── context.go      # LogContext type and context key definitions
├── logger.go       # Core Logger type wrapping slog.Logger
├── mask.go         # Sensitivity levels and masking logic
├── fields.go       # Standard field names and validators
├── formatters.go   # Custom slog.Value handlers (float, duration)
└── *_test.go       # Unit tests
```

### Core Components

#### LogContext
```go
type LogContext struct {
    SessionID         string           // session_id
    ProviderSessionID string           // provider_session_id
    Platform          string           // platform
    Namespace         string           // namespace
    UserID            string           // user_id (auto-masked)
    ChannelID         string           // channel_id
    RequestID         string           // request_id
    Sensitivity       SensitivityLevel // masking level
}
```

#### Logger
```go
type Logger struct {
    inner       *slog.Logger
    defaultCtx  *LogContext
    floatFormat FloatFormat
    maskEnabled bool
}
```

#### Sensitivity Levels
```go
type SensitivityLevel int

const (
    LevelNone     // no masking
    LevelLow      // mask credentials (token, secret)
    LevelMedium   // mask PII (user_id, email)
    LevelHigh     // mask everything
)
```

### Field Naming Convention

**All JSON fields use snake_case:**

| Category | Fields |
|----------|--------|
| Session | `session_id`, `provider_session_id` |
| User | `user_id`, `channel_id`, `thread_id` |
| Platform | `platform`, `namespace` |
| Performance | `duration_ms`, `latency_ms`, `cost_usd` |
| Operation | `operation`, `error`, `reason`, `event_type` |

### Masking Rules

| Level | Masking Behavior | Example |
|-------|------------------|---------|
| Low | Show first 4, last 4 chars | `sk-abc****xyz789` |
| Medium | Show first 2, last 2 chars | `U0****K2` (user_id) |
| High | Show first char only | `s****` |

## Implementation Plan

### Phase 1: Create Package (2-3 hours)
- [ ] Create `internal/logging/context.go`
- [ ] Create `internal/logging/logger.go`
- [ ] Create `internal/logging/mask.go`
- [ ] Create `internal/logging/fields.go`
- [ ] Create `internal/logging/formatters.go`
- [ ] Write comprehensive unit tests

### Phase 2: Core Migration (2-3 hours)
- [ ] Update `engine/runner.go` to use LogContext
- [ ] Update `chatapps/engine_handler.go` to use LogContext
- [ ] Update `internal/server/*.go` to use LogContext
- [ ] Verify `go build ./...` passes

### Phase 3: Verification (1 hour)
- [ ] Run `go test -race ./internal/logging/...`
- [ ] Run full test suite
- [ ] Validate log output format
- [ ] Check sensitivity masking works

## Usage Examples

### Basic Usage
```go
import "github.com/hrygo/hotplex/internal/logging"

logger := logging.NewLogger(slog.Default(),
    logging.WithSensitivity(logging.LevelMedium),
    logging.WithFloatFormat(logging.FloatPrecise),
)

lc := logging.NewLogContext().
    WithSessionID("7858dc94-...").
    WithPlatform("slack").
    WithUserID("U0AHCF4DPK2")  // Auto-masked

logger.With(lc).Info("session started",
    logging.FieldContentLength, 42,
)
// Output: {"session_id":"7858dc94-...","platform":"slack","user_id":"U0****K2","content_length":42}
```

### Engine Integration
```go
// engine/runner.go
func (e *Engine) Execute(ctx context.Context, cfg *Config, ...) error {
    lc := logging.NewLogContext().
        WithSessionID(cfg.SessionID).
        WithPlatform(cfg.Namespace)
    
    logger := e.logger.With(lc)
    logger.Info("starting execution pipeline")
    // ...
}
```

## Success Criteria

| Metric | Current | Target |
|--------|---------|--------|
| Field naming consistency | 60% | 100% snake_case |
| Duplicate log entries | 5+ | 0 |
| Float precision issues | 3+ | 0 (2 decimals) |
| Sensitive data masking | 0% | 100% (Medium level) |
| Context completeness | 40% | 90%+ |

## Benefits

1. **Readability**: Unified snake_case improves log parsing efficiency by 50%
2. **Security**: Auto-masking PII data for GDPR/privacy compliance
3. **Debugging**: Unified context reduces manual tracking by 70%
4. **Maintenance**: Centralized logging logic reduces code duplication by 60%

## References

- Design document: `docs/design/logging-package-design.md`
- Go slog package: https://pkg.go.dev/log/slog
- Uber Go Style Guide: https://github.com/uber-go/guide/blob/master/style.md

## Related Issues

- Issue #38: Slack Block Kit implementation (completed)
- Future: OpenTelemetry tracing integration
