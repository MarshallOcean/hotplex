# hotplex Core Architecture Documentation

*Read this in other languages: [English](architecture.md), [简体中文](architecture_zh.md).*

hotplex is a high-performance **Agent Runtime** for AI CLI Agents, designed to transform one-off terminal-based AI tools (like Claude Code or OpenCode) into production-ready, long-lived interactive services. Its core philosophy is "Leverage vs Build"—by maintaining a persistent process pool with hardened security boundaries and a normalized full-duplex protocol layer, hotplex eliminates the spin-up overhead of headless CLI environments and enables millisecond-level responsiveness.

---

## 1. Physical Layout & Clean Architecture

hotplex follows a layered architecture with strict visibility rules, separating the public SDK from internal execution details and protocol adapters.

### 1.1 Directory Structure (Actual)
- **Root (`/`)**: Main entry point for the SDK. Contains `hotplex.go` (public aliases) and `client.go`.
- **`engine/`**: The public execution runner (`Engine`). Orchestrates prompt execution, security WAF, and event dispatching.
- **`provider/`**: The abstraction layer for diverse AI CLI agents. Contains the `Provider` interface and concrete implementations for `claude-code` and `opencode`.
- **`types/`**: Fundamental data structures (`Config`, `StreamMessage`, `UsageStats`).
- **`event/`**: Unified event protocol and callback definitions (`Callback`, `EventWithMeta`).
- **`chatapps/`**: **Platform Access Layer**. Connects HotPlex to social platforms, primarily **Slack** (via Block Kit), but also supports Discord, Telegram, etc.
  - `engine_handler.go`: Bridges platform messages to Engine commands.
  - `manager.go`: Lifecycle management for bot adapters.
  - `processor_*.go`: Middleware chain for message formatting, rate limiting, and thread management.
- **`internal/engine/`**: The core execution engine. Manages the `SessionPool` (thread-safe process multiplexing) and `Session` (I/O piping and state management).
- **`internal/persistence/`**: **Session Resiliency**. Manages the `SessionMarkerStore` for detecting and resuming persistent CLI sessions after system restarts.
- **`internal/security/`**: The Regex-based WAF (`Detector`) for command auditing.
- **`internal/sys/`**: **Safety Primitives**. Cross-platform implementations for Process Group ID (PGID) management and signal cascading.
- **`internal/server/`**: Protocol adapters. Contains `hotplex_ws.go` (WebSocket) and `opencode_http.go` (REST/SSE).
- **`internal/strutil/`**: High-performance string manipulation and path cleaning.

### 1.2 Design Principles
1.  **Public Thin, Private Thick**: The root package `hotplex` provides a stable, minimal API surface.
2.  **Strategy Pattern (Provider)**: Decouples the engine from specific AI tools. `provider.Provider` allows switching backends without changing execution logic.
3.  **PGID-First Security**: Security is not an afterthought; every execution is wrapped in a dedicated process group to prevent orphan leaks.
4.  **IO-Driven State Machine**: `internal/engine` manages process states (Starting, Ready, Busy, Dead) using IO markers rather than fixed sleeps.

---

## 2. Core System Components

### 2.1 Engine & Runner (`engine/runner.go`)
*   **Engine Singleton**: The primary interface for users (`NewEngine`, `Execute`).
*   **Security Injection**: Dynamically injects global `EngineOptions` (like `AllowedTools`) into downstream sessions.
*   **Deterministic Session ID**: Uses UUID v5 to map conversation IDs to persistent sessions, ensuring high context cache hits.

### 2.2 Provider Adapter (`provider/`)
Standardizes diverse CLI protocols into a unified "HotPlex Event Stream":
*   **Provider Interface**: Handles CLI argument construction, input payload formatting, and event parsing.
*   **Factory & Registry**: `ProviderFactory` manages provider instantiation, while `ProviderRegistry` caches active instances for reuse.

### 2.3 Session Manager (`internal/engine/pool.go`)
*   **Hot-Multiplexing**: The `SessionPool` maintains a registry of active processes. Repeat requests to the same sessionID perform a "Hot Execution" (stdin injection).
*   **Slow Path Guard**: Accessing a session is thread-safe; simultaneous cold-start requests for the same ID are queued via a `pending` transition map to prevent redundant process forks.
*   **Graceful GC**: Uses a `cleanupLoop` to sweep idle processes based on `IdleTimeout`.

### 2.4 Security & Process Isolation (`internal/sys/`)
To prevent "zombie" processes when an agent spawns background tasks, HotPlex enforces group-level isolation:
*   **Unix**: Uses `setpgid(2)` on `fork()`. Termination sends `SIGKILL` to the negative PID (e.g., `kill -PID`), wiping the entire process tree.
*   **Windows**: Uses **Job Objects** (`CreateJobObjectW`). Adds the `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` flag to ensure that when the primary Job handle closes, all associated sub-processes are guaranteed to be terminated by the OS.
*   **WAF Detector**: Scans input prompts for malicious command strings (e.g., `rm -rf /`) before they reach the provider.

### 2.5 Event Hooks & Observability (`hooks/`, `telemetry/`)
*   **Webhooks & Audit**: Asynchronously broadcasts payload events to external systems (Slack, Webhooks) without blocking the hot-execution path.
*   **Tracing & Metrics**: Pushes native OpenTelemetry spans and exposes `/metrics` for Prometheus scraping.

---

## 3. Session Lifecycle & Data Flow

```mermaid
sequenceDiagram
    participant Social as "Slack / Social Platforms"
    participant ChatApps as "chatapps.EngineHandler"
    participant Client as "Client (WebSocket/SDK)"
    participant Server as "internal/server"
    participant Engine as "engine.Engine"
    participant Storage as "internal/persistence"
    participant Pool as "internal/engine.SessionPool"
    participant Provider as "provider.Provider"
    participant Proc as "CLI Process (OS)"
    
    Note over Social, ChatApps: AI Bot Integration Path
    Social->>ChatApps: Webhook / Message Event
    ChatApps->>Engine: Execute(Config, Prompt)
    
    Note over Client, Server: Direct API Path
    Client->>Server: Request (WebSocket Message / POST)
    Server->>Engine: Execute(Config, Prompt)
    
    Engine->>Engine: Check WAF (Detector)
    Engine->>Hooks: Start Trace Span
    Engine->>Pool: GetOrCreateSession(ID)
    
    alt Cold Start (Not in Pool)
        Pool->>Storage: Check Marker (Resume?)
        Pool->>Provider: Build CLI Args/Env
        Pool->>Proc: fork() with PGID / Job Object
        Pool->>Proc: Inject Context/SystemPrompt
    end
    
    Engine->>Proc: Write stdin (JSON payload)
    
    loop Stream Event Normalization
        Proc-->>Provider: Raw tool specific output
        Provider-->>Engine: Normalized ProviderEvent
        Engine-->>Hooks: Emit Event (Webhook/Log)
        
        alt Routing back to ChatApps
            Engine-->>ChatApps: Callback Event
            ChatApps-->>Social: Platform-specific Response (Throttled 1/s)
        else Routing back to API
            Engine-->>Server: Public EventWithMeta
            Server-->>Client: WebSocket/SSE Event
        end
    end
    
    Engine->>Pool: Touch(ID) to refresh idle timer
    Engine->>Hooks: End Trace & Record Metrics
```

---

## 4. Feature Matrix

### Core Capabilities
- [x] Clean Architecture with `internal/` isolation
- [x] Strategy-based Provider pattern (Claude Code, OpenCode)
- [x] Resilient Session Hot-Multiplexing
- [x] Multi-platform PGID management (Windows Job Objects / Unix PGID)
- [x] Regex-based Security WAF
- [x] **Dual Protocol Gateway**: Native WebSocket and OpenCode-compatible REST/SSE API
- [x] **Event Hooks**: Plugin system for Webhooks, Slack, and custom audit sinks
- [x] **Observability**: OpenTelemetry native tracing and Prometheus metrics (`/metrics`)

### Planned Enhancements
- **L2/L3 Isolation**: Integrating Linux Namespaces (PID/Net) and WASM sandboxing
- **Multi-Agent Bus**: Orchestrating multiple specialized agents behind a single namespace
