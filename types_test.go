package hotplex

import (
	"testing"
	"unicode/utf8"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input       string
		maxLen      int
		shouldTrunc bool
	}{
		{"hello", 10, false},
		{"hello world", 5, true},
		{"", 5, false},
		{"short", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)

			// Check that result doesn't exceed maxLen
			if len(result) > tt.maxLen {
				t.Errorf("TruncateString(%q, %d) = %q (len=%d), exceeds maxLen",
					tt.input, tt.maxLen, result, len(result))
			}

			// For non-empty input that fits, result should be unchanged
			if !tt.shouldTrunc && tt.input != "" && result != tt.input {
				t.Errorf("TruncateString(%q, %d) = %q, want unchanged", tt.input, tt.maxLen, result)
			}
		})
	}
}

func TestTruncateString_UTF8(t *testing.T) {
	// Test UTF-8 safety - Truncate handles UTF-8 correctly when maxLen >= 4
	utf8str := "中文测试很长的一段文字"
	result := TruncateString(utf8str, 10)

	// Result should be valid UTF-8
	if !isValidUTF8(result) {
		t.Errorf("TruncateString produced invalid UTF-8: %q", result)
	}
}

func TestTruncateString_NoTruncation(t *testing.T) {
	// When string fits, it should be returned unchanged
	shortStr := "hello"
	result := TruncateString(shortStr, 100)
	if result != shortStr {
		t.Errorf("TruncateString(%q, 100) = %q, want %q", shortStr, result, shortStr)
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == utf8.RuneError {
			return false
		}
	}
	return true
}

func TestSummarizeInput(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]any
		wantNotEmpty bool
	}{
		{"nil input", nil, false},
		{"empty map", map[string]any{}, false},
		{"with command", map[string]any{"command": "ls -la"}, true},
		{"with query", map[string]any{"query": "SELECT * FROM users"}, true},
		{"with path", map[string]any{"path": "/tmp/file.txt"}, true},
		{"with other", map[string]any{"foo": "bar"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeInput(tt.input)
			if tt.wantNotEmpty && result == "" {
				t.Errorf("SummarizeInput() returned empty for %v", tt.input)
			}
			if !tt.wantNotEmpty && result != "" {
				t.Errorf("SummarizeInput() = %q, want empty", result)
			}
		})
	}
}

func TestSummarizeInput_Truncation(t *testing.T) {
	// Long command should be truncated
	longCmd := "this is a very long command that should be truncated to fit within the limit"
	input := map[string]any{"command": longCmd}
	result := SummarizeInput(input)

	if len(result) > 50 {
		t.Errorf("SummarizeInput() result too long: %d chars", len(result))
	}
}

func TestStreamMessage_GetContentBlocks(t *testing.T) {
	tests := []struct {
		name    string
		msg     StreamMessage
		wantLen int
	}{
		{
			name: "from Message field",
			msg: StreamMessage{
				Message: &AssistantMessage{
					Content: []ContentBlock{{Type: "text", Text: "hello"}},
				},
			},
			wantLen: 1,
		},
		{
			name: "from Content field",
			msg: StreamMessage{
				Content: []ContentBlock{{Type: "text", Text: "world"}},
			},
			wantLen: 1,
		},
		{
			name: "Message takes precedence",
			msg: StreamMessage{
				Message: &AssistantMessage{
					Content: []ContentBlock{{Type: "text", Text: "from message"}},
				},
				Content: []ContentBlock{{Type: "text", Text: "from content"}},
			},
			wantLen: 1,
		},
		{
			name:    "empty",
			msg:     StreamMessage{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := tt.msg.GetContentBlocks()
			if len(blocks) != tt.wantLen {
				t.Errorf("GetContentBlocks() returned %d blocks, want %d", len(blocks), tt.wantLen)
			}
		})
	}
}

func TestContentBlock_GetUnifiedToolID(t *testing.T) {
	tests := []struct {
		name     string
		block    ContentBlock
		expected string
	}{
		{"ToolUseID takes precedence", ContentBlock{ToolUseID: "tool-123", ID: "block-456"}, "tool-123"},
		{"ID as fallback", ContentBlock{ID: "block-456"}, "block-456"},
		{"empty", ContentBlock{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.block.GetUnifiedToolID()
			if result != tt.expected {
				t.Errorf("GetUnifiedToolID() = %q, want %q", result, tt.expected)
			}
		})
	}
}
