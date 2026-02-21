package hotplex

import (
	"errors"
	"log/slog"
	"os"
	"testing"
)

func TestNewEventWithMeta(t *testing.T) {
	meta := &EventMeta{
		DurationMs:   100,
		ToolName:     "bash",
		ToolID:       "tool-123",
		Status:       "success",
		InputTokens:  50,
		OutputTokens: 100,
	}

	event := NewEventWithMeta("tool_use", "executing command", meta)

	if event.EventType != "tool_use" {
		t.Errorf("EventType = %q, want tool_use", event.EventType)
	}
	if event.EventData != "executing command" {
		t.Errorf("EventData = %q, want 'executing command'", event.EventData)
	}
	if event.Meta.DurationMs != 100 {
		t.Errorf("Meta.DurationMs = %d, want 100", event.Meta.DurationMs)
	}
	if event.Meta.ToolName != "bash" {
		t.Errorf("Meta.ToolName = %q, want bash", event.Meta.ToolName)
	}
}

func TestNewEventWithMeta_NilMeta(t *testing.T) {
	event := NewEventWithMeta("thinking", "AI is thinking", nil)

	if event == nil {
		t.Fatal("NewEventWithMeta returned nil")
	}
	if event.Meta == nil {
		t.Fatal("Meta should not be nil")
	}
	// Meta should be zero-valued struct
	if event.Meta.DurationMs != 0 {
		t.Errorf("Meta.DurationMs = %d, want 0 (zero value)", event.Meta.DurationMs)
	}
}

func TestWrapSafe_NilCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	wrapped := WrapSafe(logger, nil)
	if wrapped != nil {
		t.Error("WrapSafe with nil callback should return nil")
	}
}

func TestWrapSafe_SuccessfulCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	called := false

	cb := func(eventType string, data any) error {
		called = true
		return nil
	}

	wrapped := WrapSafe(logger, cb)
	err := wrapped("test", "data")

	if err != nil {
		t.Errorf("wrapped callback returned error: %v", err)
	}
	if !called {
		t.Error("callback was not called")
	}
}

func TestWrapSafe_ErrorCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	expectedErr := errors.New("callback error")

	cb := func(eventType string, data any) error {
		return expectedErr
	}

	wrapped := WrapSafe(logger, cb)
	err := wrapped("test", "data")

	// WrapSafe should suppress the error (return nil)
	if err != nil {
		t.Errorf("WrapSafe should suppress error, got: %v", err)
	}
}
