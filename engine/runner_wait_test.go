package engine

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	intengine "github.com/hrygo/hotplex/internal/engine"
	"github.com/hrygo/hotplex/provider"
	"github.com/hrygo/hotplex/types"
)

func TestEngine_createEventBridge(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	// Create event bridge
	cb := engine.createEventBridge(cfg, nil, stats, doneChan)

	if cb == nil {
		t.Fatal("createEventBridge returned nil")
	}

	// Test runner_exit event
	err := cb("runner_exit", nil)
	if err != nil {
		t.Errorf("runner_exit callback error: %v", err)
	}

	// doneChan should be closed after runner_exit
	select {
	case <-doneChan:
		// Expected
	default:
		t.Error("doneChan should be closed after runner_exit")
	}
}

func TestEngine_createEventBridge_RawLine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	var received string
	userCb := func(eventType string, data any) error {
		if eventType == "answer" {
			received = data.(string)
		}
		return nil
	}

	cb := engine.createEventBridge(cfg, userCb, stats, doneChan)

	// Test raw_line event with invalid JSON (should be passed as answer)
	err := cb("raw_line", "not valid json")
	if err != nil {
		t.Errorf("raw_line callback error: %v", err)
	}

	if received != "not valid json" {
		t.Errorf("received = %q, want 'not valid json'", received)
	}
}

func TestEngine_createEventBridge_ResultMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create mock manager with session
	mockMgr := &mockSessionManager{sessions: make(map[string]*intengine.Session)}
	mockMgr.sessions["test-session"] = intengine.NewTestSession("test-session", intengine.SessionStatusBusy)

	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		manager:  mockMgr,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	var sessionStatsReceived bool
	userCb := func(eventType string, data any) error {
		if eventType == "session_stats" {
			sessionStatsReceived = true
		}
		return nil
	}

	cb := engine.createEventBridge(cfg, userCb, stats, doneChan)

	// Test result message
	msg := types.StreamMessage{
		Type:     "result",
		Duration: 1000,
		Usage: &types.UsageStats{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}

	err := cb("result", msg)
	if err != nil {
		t.Errorf("result callback error: %v", err)
	}

	// doneChan should be closed after result
	select {
	case <-doneChan:
		// Expected
	default:
		t.Error("doneChan should be closed after result message")
	}

	if !sessionStatsReceived {
		t.Error("session_stats event should be sent")
	}
}

func TestEngine_createEventBridge_SystemMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	var called bool
	userCb := func(eventType string, data any) error {
		called = true
		return nil
	}

	cb := engine.createEventBridge(cfg, userCb, stats, doneChan)

	// Test system message - should be silently ignored
	msg := types.StreamMessage{Type: "system"}
	err := cb("pre-parsed", msg)
	if err != nil {
		t.Errorf("system message callback error: %v", err)
	}

	if called {
		t.Error("system message should not trigger user callback")
	}

	// doneChan should NOT be closed for system message
	select {
	case <-doneChan:
		t.Error("doneChan should NOT be closed for system message")
	default:
		// Expected
	}
}

func TestEngine_createEventBridge_NonStreamMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	var received string
	userCb := func(eventType string, data any) error {
		received = eventType
		return nil
	}

	cb := engine.createEventBridge(cfg, userCb, stats, doneChan)

	// Test non-types.StreamMessage data (legacy path)
	err := cb("custom_event", "some data")
	if err != nil {
		t.Errorf("non-types.StreamMessage callback error: %v", err)
	}

	if received != "custom_event" {
		t.Errorf("received = %q, want 'custom_event'", received)
	}
}

func TestEngine_createEventBridge_WithCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockMgr := &mockSessionManager{sessions: make(map[string]*intengine.Session)}
	mockMgr.sessions["test-session"] = intengine.NewTestSession("test-session", intengine.SessionStatusBusy)

	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		manager:  mockMgr,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	var receivedType string
	userCb := func(eventType string, data any) error {
		receivedType = eventType
		return nil
	}

	cb := engine.createEventBridge(cfg, userCb, stats, doneChan)

	// Test with a message that goes through dispatchCallback
	msg := types.StreamMessage{
		Type: "thinking",
		Content: []types.ContentBlock{
			{Type: "text", Text: "thinking..."},
		},
	}

	err := cb("pre-parsed", msg)
	if err != nil {
		t.Errorf("thinking message callback error: %v", err)
	}

	if receivedType != "thinking" {
		t.Errorf("receivedType = %q, want 'thinking'", receivedType)
	}
}

func TestEngine_createEventBridge_RawLineNotString(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	prv, _ := provider.NewClaudeCodeProvider(provider.ProviderConfig{}, logger)
	engine := &Engine{
		opts:     EngineOptions{Namespace: "test"},
		logger:   logger,
		provider: prv,
	}

	cfg := &types.Config{
		WorkDir:   "/tmp",
		SessionID: "test-session",
	}

	stats := &SessionStats{SessionID: "test-session"}
	doneChan := make(chan struct{})

	cb := engine.createEventBridge(cfg, nil, stats, doneChan)

	// Test raw_line with non-string data - should be silently ignored
	err := cb("raw_line", 12345)
	if err != nil {
		t.Errorf("raw_line with non-string error: %v", err)
	}
}

func TestEngine_waitForSession(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eng := &Engine{
		opts:   EngineOptions{Namespace: "test"},
		logger: logger,
	}

	t.Run("session ready", func(t *testing.T) {
		sess := intengine.NewTestSession("test", intengine.SessionStatusReady)

		ctx := context.Background()
		err := eng.waitForSession(ctx, sess, "test-session")
		if err != nil {
			t.Errorf("waitForSession error: %v", err)
		}
	})

	t.Run("session busy", func(t *testing.T) {
		sess := intengine.NewTestSession("test", intengine.SessionStatusBusy)

		ctx := context.Background()
		err := eng.waitForSession(ctx, sess, "test-session")
		if err != nil {
			t.Errorf("waitForSession error: %v", err)
		}
	})

	t.Run("session dead", func(t *testing.T) {
		sess := intengine.NewTestSession("test", intengine.SessionStatusDead)

		ctx := context.Background()
		err := eng.waitForSession(ctx, sess, "test-session")
		if err == nil {
			t.Error("waitForSession should fail for dead session")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		sess := intengine.NewTestSession("test", intengine.SessionStatusStarting)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := eng.waitForSession(ctx, sess, "test-session")
		if err != context.Canceled {
			t.Errorf("waitForSession error = %v, want context.Canceled", err)
		}
	})
}

func TestEngine_waitForSession_StatusChange(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eng := &Engine{
		opts:   EngineOptions{Namespace: "test"},
		logger: logger,
	}

	sess := intengine.NewTestSession("test", intengine.SessionStatusStarting)

	ctx := context.Background()

	// Send status change in goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.SetStatus(intengine.SessionStatusReady)
	}()

	err := eng.waitForSession(ctx, sess, "test-session")
	if err != nil {
		t.Errorf("waitForSession error: %v", err)
	}
}
