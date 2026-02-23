# HotPlex Quality Assurance & Audit Report

**Version**: 1.0  
**Date**: 2026-02-23  
**Auditor**: Automated QA System  
**Worktree**: `/Users/huangzhonghui/HotPlex-qa`  
**Branch**: `qa/audit`  
**Base Commit**: `9e5bec9`

---

## Executive Summary

This comprehensive audit covers **concurrency safety**, **security vulnerabilities**, **error handling**, **test coverage**, and **dependency management** for the HotPlex AI Agent Control Plane.

### Overall Assessment

| Category | Status | Critical | High | Medium | Low |
|----------|--------|----------|------|--------|-----|
| Concurrency Safety | ⚠️ Issues Found | 2 | 2 | 0 | 2 |
| Security | ⚠️ Issues Found | 0 | 3 | 4 | 2 |
| Error Handling | ✅ Good | 0 | 0 | 10 | 0 |
| Test Coverage | ⚠️ Needs Improvement | - | - | - | - |
| Dependencies | ✅ Up to Date | 0 | 0 | 0 | 0 |

**Race Detector**: ✅ PASSED (`go test -race ./...`)

**Total Test Coverage**: 47.9%

---

## 1. Concurrency Safety Audit

### 1.1 CRITICAL Issues

#### [C-01] Race Condition: Map Read Without Lock
- **Location**: `internal/engine/pool.go:96`
- **Severity**: CRITICAL
- **Description**: After releasing `sm.mu.RUnlock()` at line 81, line 96 accesses `sm.pending[sessionID]` without holding any lock.
```go
sm.mu.RUnlock()  // Line 81 - lock released
// ... gap ...
if ch, ok := sm.pending[sessionID]; ok {  // Line 96 - RACE!
```
- **Fix**: Access `sm.pending` within lock scope or use separate mutex for pending map.

#### [C-02] Race Condition: Map Write Without Lock
- **Location**: `internal/engine/pool.go:109`
- **Severity**: CRITICAL
- **Description**: `sm.pending[sessionID] = ch` at line 109 executes AFTER `sm.mu.Unlock()` at line 110.
```go
sm.pending[sessionID] = ch  // Line 109
sm.mu.Unlock()              // Line 110
```
- **Fix**: Move map write before unlock or use separate mutex.

### 1.2 HIGH Issues

#### [H-01] Session Field Access Without Lock
- **Location**: `internal/engine/pool.go:368`
- **Severity**: HIGH
- **Description**: Reading `sess.LastActive` without holding `sess.mu` lock.
```go
for sessionID, sess := range sm.sessions {
    idleTime := now.Sub(sess.LastActive)  // Reading without sess.mu lock
}
```
- **Fix**: Call `sess.Touch()` or acquire session lock before reading.

#### [H-02] Potential Deadlock in waitForReady
- **Location**: `internal/engine/session.go:99-131`
- **Severity**: HIGH
- **Description**: The goroutine acquires `s.mu.Lock()` then calls `s.SetStatus()` which also tries to acquire `s.mu.Lock()`. RWMutex is NOT reentrant!
```go
s.mu.Lock()
if s.isAliveLocked() {
    s.mu.Unlock()
    s.SetStatus(SessionStatusReady)  // This calls Lock() again!
    return
}
```
- **Fix**: Refactor to avoid re-entrant locking or use a different pattern.

### 1.3 LOW Issues

| ID | Location | Description |
|----|----------|-------------|
| L-01 | `executor/docker.go:84` | Goroutine using possibly-cancelled context |
| L-02 | `executor/docker.go:51` | Context lifetime vs goroutine timing |

---

## 2. Security Audit

### 2.1 CRITICAL Issues

**None identified** - The core security model is sound. All prompts pass through `CheckInput()` before reaching stdin.

### 2.2 HIGH Issues

#### [S-H-01] Incomplete Process Isolation on Windows
- **Location**: `internal/sys/proc_windows.go:18`
- **Severity**: HIGH
- **Description**: Windows `SetupCmdSysProcAttr()` is empty - no PGID support. Unlike Unix (which sets `Setpgid: true`), Windows relies only on `taskkill` fallback which may leave orphaned child processes.
- **Impact**: Violates AGENT.md rule #3 (zombie prevention) on Windows platform.
- **Fix**: Implement Windows Job Objects for process tree isolation.

#### [S-H-02] Environment Variable Inheritance
- **Location**: `internal/engine/pool.go:222`
- **Severity**: HIGH
- **Description**: `cmd.Env = append(os.Environ(), ...)` inherits ALL parent environment variables. Malicious variables like `LD_PRELOAD`, `PATH`, or `HOME` could influence subprocess behavior.
- **Fix**: Use controlled environment variable allowlist instead of inheriting `os.Environ()`.

#### [S-H-03] Default WorkDir Not Validated
- **Location**: `internal/server/controller.go:47`
- **Severity**: HIGH
- **Description**: Default `workDir = "/tmp/hotplex_sandbox"` is used without passing through `Engine.ValidateConfig()`. This path is not validated against the Detector's allowed paths.
- **Fix**: Ensure default WorkDir goes through full validation path.

### 2.3 MEDIUM Issues

| ID | Location | Description |
|----|----------|-------------|
| S-M-01 | `detector.go:527-540` | Relative path resolution uses `os.Getwd()` - could be exploited if cwd is controlled |
| S-M-02 | `detector.go:315` | Bypass mode skips ALL checks - no rate limiting on bypass attempts |
| S-M-03 | `detector.go:193` | WorkDir check doesn't validate against `SetAllowPaths()` |
| S-M-04 | `detector.go:509-518` | Path prefix matching could be bypassed via symlinks or case-insensitive FS |

### 2.4 Security Positive Findings

| Location | Finding |
|----------|---------|
| `engine/runner.go:109` | ✅ CheckInput() properly gates all input before execution |
| `internal/engine/pool.go:220` | ✅ exec.Command uses args slice (not shell string) - no injection |
| `internal/sys/proc_unix.go:13` | ✅ Setpgid: true correctly implemented on Unix |
| `detector.go:322-353` | ✅ Null byte and control character filtering |
| `detector.go:489` | ✅ Constant-time token comparison prevents timing attacks |
| `engine/runner.go:193` | ✅ Path traversal ("..") detection for WorkDir |

---

## 3. Error Handling Audit

### 3.1 PASSED Checks

| Check | Status | Notes |
|-------|--------|-------|
| Swallowed errors | ✅ NONE | Clean error handling |
| panic() in core | ✅ NONE | Only in examples |
| log.Fatal in library | ✅ NONE | Only in cmd/examples |
| Incorrect wrapping | ✅ NONE | Correctly uses %w |

### 3.2 MEDIUM Issues - Missing Context Wrapping

The following locations return errors directly without wrapping context:

| File | Line | Current | Suggested Fix |
|------|------|---------|---------------|
| `executor/docker.go` | 122 | `return err` | `return fmt.Errorf("image pull %s: %w", img, err)` |
| `internal/engine/session.go` | 147 | `return err` | `return fmt.Errorf("json marshal: %w", err)` |
| `internal/engine/session.go` | 154 | `return err` | `return fmt.Errorf("stdin write: %w", err)` |
| `config/hotreload.go` | 44 | `return err` | `return fmt.Errorf("read config %s: %w", path, err)` |
| `config/hotreload.go` | 51 | `return err` | `return fmt.Errorf("parse config: %w", err)` |
| `config/hotreload.go` | 61 | `return err` | `return fmt.Errorf("create watcher: %w", err)` |
| `config/hotreload.go` | 68 | `return err` | `return fmt.Errorf("watch path %s: %w", path, err)` |
| `internal/server/controller.go` | 73 | `return err` | Already logged - optional wrap |
| `provider/provider.go` | 235 | `return err` | `return fmt.Errorf("validate binary: %w", err)` |
| `provider/provider.go` | 240 | `return err` | `return fmt.Errorf("get version: %w", err)` |

---

## 4. Test Coverage Analysis

### 4.1 Current Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| `types` | 100.0% | ✅ Excellent |
| `event` | 100.0% | ✅ Excellent |
| `internal/sys` | 100.0% | ✅ Excellent |
| `internal/security` | 87.8% | ✅ Good |
| `internal/strutil` | 80.0% | ✅ Good |
| `provider` | 59.9% | ⚠️ Needs Work |
| `engine` | 56.9% | ⚠️ Needs Work |
| `telemetry` | 53.3% | ⚠️ Needs Work |
| `internal/server` | 46.7% | ⚠️ Needs Work |
| `internal/engine` | 41.0% | ⚠️ Needs Work |
| `hooks` | 26.2% | ❌ Critical |
| `cmd/hotplexd` | 0.0% | ❌ No Tests |
| `config` | 0.0% | ❌ No Tests |
| `executor` | 0.0% | ❌ No Tests |

**Total Coverage**: 47.9%

### 4.2 Missing Test Files

| Package | Priority | Risk |
|---------|----------|------|
| `cmd/hotplexd` | Medium | Entry point - consider E2E tests |
| `config` | High | Hot reload logic untested |
| `executor` | High | Docker execution untested |

### 4.3 Specific Gaps

| Location | Gap | Suggested Test |
|----------|-----|----------------|
| `telemetry/tracer.go` | 0% coverage | Add tracer lifecycle tests |
| `provider/types.go:62` | `GetUnifiedToolID` 0% | Add unit test |
| `hooks/hooks.go` | 26.2% | Add hook execution tests |

---

## 5. Dependency Audit

### 5.1 Status

- **Go Version**: 1.24
- **Vulnerability Scan**: Clean (no known CVEs in direct dependencies)

### 5.2 Available Updates

| Dependency | Current | Latest | Action |
|------------|---------|--------|--------|
| `cel.dev/expr` | v0.24.0 | v0.25.1 | Minor update available |
| `github.com/docker/docker` | v28.0.0 | v28.5.2 | Update recommended |
| `golang.org/x/crypto` | v0.47.0 | v0.48.0 | Security - update |
| `golang.org/x/net` | v0.49.0 | v0.50.0 | Minor update |
| `golang.org/x/sys` | v0.40.0 | v0.41.0 | Minor update |

---

## 6. AGENT.md Compliance Check

| Rule | Status | Notes |
|------|--------|-------|
| SRP (Single Responsibility) | ✅ PASS | Clear separation of concerns |
| Concurrency Safety | ⚠️ ISSUES | 2 CRITICAL, 2 HIGH issues found |
| Process Lifecycle (PGID) | ⚠️ PARTIAL | Unix OK, Windows incomplete |
| Error Handling (no panic) | ✅ PASS | No panics in core engine |
| Error Wrapping (%w) | ⚠️ PARTIAL | 10 locations need wrapping |
| Logging (slog) | ✅ PASS | Consistent structured logging |
| CheckInput Gate | ✅ PASS | All inputs pass through WAF |
| No Shell Hacks | ✅ PASS | No sh -c / bash -c usage |

---

## 7. Recommendations

### 7.1 Immediate Actions (P0)

1. **Fix CRITICAL race conditions in `pool.go:96, 109`**
   - Wrap `pending` map access in proper locks
   - Consider using `sync.Map` for pending sessions

2. **Fix HIGH deadlock in `session.go:99-131`**
   - Refactor `waitForReady` to avoid re-entrant locking

### 7.2 Short-term Actions (P1)

3. **Implement Windows Job Objects** (`proc_windows.go`)
   - Required for AGENT.md compliance on Windows

4. **Sanitize environment variables** (`pool.go:222`)
   - Use explicit allowlist instead of `os.Environ()`

5. **Add missing test coverage**
   - Target: `config` package, `executor` package
   - Increase `hooks` from 26% to 80%+

### 7.3 Medium-term Actions (P2)

6. **Add error context wrapping** (10 locations)
7. **Implement rate limiting** for bypass mode attempts
8. **Update dependencies** (`golang.org/x/crypto`, `docker/docker`)

---

## 8. Verification Commands

```bash
# Run race detector
go test -race ./...

# Run with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out

# Run go vet
go vet ./...

# Build verification
go build ./...
```

---

## Appendix A: Files Analyzed

- `internal/engine/pool.go` (424 lines)
- `internal/engine/session.go` (300 lines)
- `internal/security/detector.go` (616 lines)
- `internal/sys/proc_unix.go`
- `internal/sys/proc_windows.go`
- `engine/runner.go`
- `executor/docker.go`
- `config/hotreload.go`
- `provider/provider.go`
- `internal/server/controller.go`
- `internal/server/security.go`
- `internal/server/hotplex_ws.go`

---

## Appendix B: Audit Methodology

1. **Static Analysis**: `go vet`, code review
2. **Race Detection**: `go test -race ./...`
3. **Coverage Analysis**: `go test -cover`
4. **Pattern Matching**: AST grep for anti-patterns
5. **Security Flow Analysis**: Input → CheckInput → Execution path tracing

---

*Report generated automatically by QA audit system.*
*Next audit recommended: After P0/P1 fixes are merged.*
