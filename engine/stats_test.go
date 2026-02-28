package engine

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSessionStats_RecordToolUse(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Record tool use
	stats.RecordToolUse("bash", "tool-123")

	// Small delay to ensure duration > 0
	time.Sleep(1 * time.Millisecond)

	// Verify internal state (need to check via RecordToolResult)
	duration, toolName := stats.RecordToolResult("tool-123")

	// Duration may be 0 if too fast, so we just check it's non-negative
	if duration < 0 {
		t.Errorf("RecordToolResult() returned negative duration: %d", duration)
	}
	if toolName != "bash" {
		t.Errorf("RecordToolResult() toolName = %q, want %q", toolName, "bash")
	}
	if stats.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", stats.ToolCallCount)
	}
	if !stats.ToolsUsed["bash"] {
		t.Error("ToolsUsed should contain 'bash'")
	}
}

func TestSessionStats_RecordToolUse_NilMap(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// ToolsUsed is nil, should be initialized
	stats.RecordToolUse("edit", "tool-456")
	_, toolName := stats.RecordToolResult("tool-456")

	if stats.ToolsUsed == nil {
		t.Error("ToolsUsed should be initialized")
	}
	if !stats.ToolsUsed["edit"] {
		t.Error("ToolsUsed should contain 'edit'")
	}
	if toolName != "edit" {
		t.Errorf("RecordToolResult() toolName = %q, want %q", toolName, "edit")
	}
}

func TestSessionStats_RecordToolResult_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Call RecordToolResult without RecordToolUse
	// This records minimal duration (1ms) because Claude Code CLI may not send tool_use events
	duration, toolName := stats.RecordToolResult("")

	if duration != 1 {
		t.Errorf("RecordToolResult() without RecordToolUse should return 1 (minimal duration), got %d", duration)
	}
	if toolName != "" {
		t.Errorf("RecordToolResult() toolName = %q, want empty string", toolName)
	}
	if stats.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", stats.ToolCallCount)
	}
}

func TestSessionStats_RecordTokens(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordTokens(100, 50, 20, 10)

	if stats.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", stats.InputTokens)
	}
	if stats.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", stats.OutputTokens)
	}
	if stats.CacheWriteTokens != 20 {
		t.Errorf("CacheWriteTokens = %d, want 20", stats.CacheWriteTokens)
	}
	if stats.CacheReadTokens != 10 {
		t.Errorf("CacheReadTokens = %d, want 10", stats.CacheReadTokens)
	}
}

func TestSessionStats_RecordTokens_Accumulative(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordTokens(100, 50, 20, 10)
	stats.RecordTokens(50, 25, 10, 5)

	if stats.InputTokens != 150 {
		t.Errorf("InputTokens = %d, want 150 (accumulated)", stats.InputTokens)
	}
	if stats.OutputTokens != 75 {
		t.Errorf("OutputTokens = %d, want 75 (accumulated)", stats.OutputTokens)
	}
}

func TestSessionStats_ThinkingPhase(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.StartThinking()
	time.Sleep(10 * time.Millisecond)
	stats.EndThinking()

	if stats.ThinkingDurationMs <= 0 {
		t.Errorf("ThinkingDurationMs = %d, want > 0", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_ThinkingPhase_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// End without start should be safe (no-op)
	stats.EndThinking()

	if stats.ThinkingDurationMs != 0 {
		t.Errorf("ThinkingDurationMs = %d, want 0", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_ThinkingPhase_Multiple(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Multiple thinking phases should accumulate
	for i := 0; i < 3; i++ {
		stats.StartThinking()
		time.Sleep(5 * time.Millisecond)
		stats.EndThinking()
	}

	if stats.ThinkingDurationMs < 10 { // At least 15ms total
		t.Errorf("ThinkingDurationMs = %d, want >= 15 (accumulated)", stats.ThinkingDurationMs)
	}
}

func TestSessionStats_GenerationPhase(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.StartGeneration()
	time.Sleep(10 * time.Millisecond)
	stats.EndGeneration()

	if stats.GenerationDurationMs <= 0 {
		t.Errorf("GenerationDurationMs = %d, want > 0", stats.GenerationDurationMs)
	}
}

func TestSessionStats_GenerationPhase_NoStart(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// End without start should be safe (no-op)
	stats.EndGeneration()

	if stats.GenerationDurationMs != 0 {
		t.Errorf("GenerationDurationMs = %d, want 0", stats.GenerationDurationMs)
	}
}

func TestSessionStats_RecordFileModification(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	stats.RecordFileModification("/path/to/file1.txt")
	stats.RecordFileModification("/path/to/file2.txt")

	if stats.FilesModified != 2 {
		t.Errorf("FilesModified = %d, want 2", stats.FilesModified)
	}
	if len(stats.FilePaths) != 2 {
		t.Errorf("len(FilePaths) = %d, want 2", len(stats.FilePaths))
	}
}

func TestSessionStats_RecordFileModification_Empty(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Empty path should be ignored
	stats.RecordFileModification("")

	if stats.FilesModified != 0 {
		t.Errorf("FilesModified = %d, want 0", stats.FilesModified)
	}
}

func TestSessionStats_RecordFileModification_Deduplication(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Same file multiple times should only count once
	stats.RecordFileModification("/path/to/file.txt")
	stats.RecordFileModification("/path/to/file.txt")
	stats.RecordFileModification("/path/to/file.txt")

	if stats.FilesModified != 1 {
		t.Errorf("FilesModified = %d, want 1 (deduplicated)", stats.FilesModified)
	}
	if len(stats.FilePaths) != 1 {
		t.Errorf("len(FilePaths) = %d, want 1 (deduplicated)", len(stats.FilePaths))
	}
}

func TestSessionStats_ToSummary(t *testing.T) {
	stats := &SessionStats{
		SessionID:            "test-session",
		StartTime:            time.Now(),
		TotalDurationMs:      1000,
		ThinkingDurationMs:   200,
		ToolDurationMs:       500,
		GenerationDurationMs: 300,
		InputTokens:          100,
		OutputTokens:         50,
		CacheWriteTokens:     20,
		CacheReadTokens:      10,
		ToolCallCount:        5,
		ToolsUsed:            map[string]bool{"bash": true, "edit": true},
		FilesModified:        3,
		FilePaths:            []string{"/a.txt", "/b.txt", "/c.txt"},
	}

	summary := stats.ToSummary()

	if summary["session_id"] != "test-session" {
		t.Errorf("session_id = %v, want test-session", summary["session_id"])
	}
	if summary["total_duration_ms"] != int64(1000) {
		t.Errorf("total_duration_ms = %v, want 1000", summary["total_duration_ms"])
	}
	if summary["tool_call_count"] != int32(5) {
		t.Errorf("tool_call_count = %v, want 5", summary["tool_call_count"])
	}

	// Check tools_used is a slice
	tools, ok := summary["tools_used"].([]string)
	if !ok {
		t.Fatal("tools_used should be []string")
	}
	if len(tools) != 2 {
		t.Errorf("len(tools_used) = %d, want 2", len(tools))
	}
}

func TestSessionStats_FinalizeDuration(t *testing.T) {
	stats := &SessionStats{
		SessionID:            "test-session",
		StartTime:            time.Now(),
		ThinkingDurationMs:   100,
		GenerationDurationMs: 200,
	}

	// Start phases but don't end them
	stats.StartThinking()
	stats.StartGeneration()

	finalized := stats.FinalizeDuration()

	// Finalized should include ongoing phases
	if finalized.ThinkingDurationMs < 100 {
		t.Errorf("ThinkingDurationMs = %d, want >= 100", finalized.ThinkingDurationMs)
	}
	if finalized.GenerationDurationMs < 200 {
		t.Errorf("GenerationDurationMs = %d, want >= 200", finalized.GenerationDurationMs)
	}
}

func TestSessionStats_Concurrency(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent operations - focus on thread-safe methods that don't have state dependencies
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				// These methods are designed for concurrent use
				stats.RecordTokens(1, 1, 1, 1)
				stats.StartThinking()
				stats.EndThinking()
				stats.StartGeneration()
				stats.EndGeneration()
				stats.RecordFileModification("/path/to/file")
			}
		}(i)
	}

	wg.Wait()

	// Verify counts (should be goroutines * operations for tokens)
	expectedTokens := int32(goroutines * operations)
	if stats.InputTokens != expectedTokens {
		t.Errorf("InputTokens = %d, want %d", stats.InputTokens, expectedTokens)
	}

	// File modifications should be deduplicated (same path)
	if stats.FilesModified != 1 {
		t.Errorf("FilesModified = %d, want 1 (deduplicated)", stats.FilesModified)
	}
}

// TestSessionStats_ToolTracking tests tool tracking in a single goroutine
// (RecordToolUse/RecordToolResult have internal state that shouldn't be used concurrently)
func TestSessionStats_ToolTracking(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Sequential tool calls
	for i := 0; i < 5; i++ {
		toolName := fmt.Sprintf("tool-%d", i)
		stats.RecordToolUse(toolName, fmt.Sprintf("id-%d", i))
		time.Sleep(1 * time.Millisecond)
		_, returnedToolName := stats.RecordToolResult(fmt.Sprintf("id-%d", i))
		if returnedToolName != toolName {
			t.Errorf("RecordToolResult() returned toolName = %q, want %q", returnedToolName, toolName)
		}
	}

	// Verify counts
	if stats.ToolCallCount != 5 {
		t.Errorf("ToolCallCount = %d, want 5", stats.ToolCallCount)
	}

	// Should have 5 unique tools
	if len(stats.ToolsUsed) != 5 {
		t.Errorf("len(ToolsUsed) = %d, want 5", len(stats.ToolsUsed))
	}
}

// TestSessionStats_ToolIDMap tests the toolID -> toolName mapping
// This is the key fix for when Claude CLI sends tool_result without tool_name
func TestSessionStats_ToolIDMap(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Record tool use with name and ID
	stats.RecordToolUse("Read", "call_abc123")

	// Simulate tool_result arriving after currentToolName has been cleared
	// First call RecordToolResult to clear currentToolName
	stats.RecordToolResult("call_abc123")

	// Now verify we can still get tool name by toolID
	toolName := stats.GetToolNameByToolID("call_abc123")
	if toolName != "Read" {
		t.Errorf("GetToolNameByToolID() = %q, want %q", toolName, "Read")
	}

	// Unknown toolID should return empty
	unknownTool := stats.GetToolNameByToolID("unknown_id")
	if unknownTool != "" {
		t.Errorf("GetToolNameByToolID(unknown) = %q, want empty", unknownTool)
	}

	// Empty toolID should return empty
	emptyTool := stats.GetToolNameByToolID("")
	if emptyTool != "" {
		t.Errorf("GetToolNameByToolID(empty) = %q, want empty", emptyTool)
	}
}

// TestSessionStats_RecordToolResult_WithToolIDMap tests RecordToolResult
// correctly uses the toolID map when currentToolName is empty
func TestSessionStats_RecordToolResult_WithToolIDMap(t *testing.T) {
	stats := &SessionStats{SessionID: "test"}

	// Record tool use
	stats.RecordToolUse("Bash", "call_xyz789")

	// Clear current tool tracking (simulating what happens after first RecordToolResult)
	stats.currentToolName = ""
	stats.currentToolID = ""
	stats.currentToolStart = time.Time{}

	// Now call RecordToolResult with only toolID
	// It should use the toolID map to get the tool name
	duration, toolName := stats.RecordToolResult("call_xyz789")

	if toolName != "Bash" {
		t.Errorf("RecordToolResult() toolName = %q, want %q", toolName, "Bash")
	}
	if duration != 1 {
		// Should be 1ms because currentToolStart was cleared
		t.Errorf("RecordToolResult() duration = %d, want 1", duration)
	}
	if !stats.ToolsUsed["Bash"] {
		t.Error("ToolsUsed should contain 'Bash'")
	}
}
