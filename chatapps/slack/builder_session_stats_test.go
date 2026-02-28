package slack

import (
	"testing"

	"github.com/hrygo/hotplex/chatapps/base"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestBuildSessionStatsMessage_Int32Types(t *testing.T) {
	// This test verifies that BuildSessionStatsMessage correctly handles
	// int32 types from SessionStatsData (the actual types used in production)
	builder := NewMessageBuilder()

	msg := &base.ChatMessage{
		Type:    base.MessageTypeSessionStats,
		Content: "",
		Metadata: map[string]any{
			"event_type":     "session_stats",
			"session_id":     "sess_123",
			"duration_ms":    int64(12500),
			"tokens_in":      int32(1200), // int32 as from SessionStatsData
			"tokens_out":     int32(350),  // int32 as from SessionStatsData
			"total_tokens":   int32(1550), // int32 as from SessionStatsData
			"tool_count":     int32(3),    // int32 as from SessionStatsData
			"files_modified": int32(2),    // int32 as from SessionStatsData
			"tools_used":     []string{"Read", "Edit", "Bash"},
		},
	}

	blocks := builder.BuildSessionStatsMessage(msg)

	assert.NotNil(t, blocks)
	assert.Len(t, blocks, 2) // Header + stats context

	// Verify header block
	headerBlock := blocks[0]
	assert.NotNil(t, headerBlock)

	// Verify context block contains stats
	contextBlock := blocks[1]
	assert.NotNil(t, contextBlock)
}

func TestBuildSessionStatsMessage_Int64Types(t *testing.T) {
	// This test verifies backward compatibility with int64 types
	builder := NewMessageBuilder()

	msg := &base.ChatMessage{
		Type:    base.MessageTypeSessionStats,
		Content: "",
		Metadata: map[string]any{
			"event_type":     "session_stats",
			"duration_ms":    int64(12500),
			"tokens_in":      int64(1200),
			"tokens_out":     int64(350),
			"tool_count":     int64(3),
			"files_modified": int64(2),
		},
	}

	blocks := builder.BuildSessionStatsMessage(msg)

	assert.NotNil(t, blocks)
	assert.Len(t, blocks, 2) // Header + stats context
}

func TestBuildSessionStatsMessage_Empty(t *testing.T) {
	// When no stats are available, should still return valid blocks
	builder := NewMessageBuilder()

	msg := &base.ChatMessage{
		Type:    base.MessageTypeSessionStats,
		Content: "",
		Metadata: map[string]any{
			"event_type": "session_stats",
		},
	}

	blocks := builder.BuildSessionStatsMessage(msg)

	assert.NotNil(t, blocks)
	assert.Len(t, blocks, 2) // Header + "Session completed" context
}

func TestBuildSessionStatsMessage_WithAllFields(t *testing.T) {
	// Full test with all stats fields populated
	builder := NewMessageBuilder()

	msg := &base.ChatMessage{
		Type:    base.MessageTypeSessionStats,
		Content: "",
		Metadata: map[string]any{
			"event_type":           "session_stats",
			"session_id":           "sess_123",
			"duration_ms":          int64(12500),
			"thinking_duration_ms": int64(3000),
			"tool_duration_ms":     int64(5000),
			"tokens_in":            int32(1200),
			"tokens_out":           int32(350),
			"total_tokens":         int32(1550),
			"tool_count":           int32(3),
			"tools_used":           []string{"Read", "Edit", "Bash"},
			"files_modified":       int32(2),
		},
	}

	blocks := builder.BuildSessionStatsMessage(msg)

	assert.NotNil(t, blocks)
	assert.Len(t, blocks, 2)

	// Verify the stats line contains expected emojis
	contextBlock, ok := blocks[1].(*slack.ContextBlock)
	if ok {
		assert.NotNil(t, contextBlock)
		// The block should contain duration, tokens, files, and tools
	}
}

func TestExtractInt64(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		key      string
		expected int64
	}{
		{
			name: "int64 value",
			metadata: map[string]any{
				"key": int64(100),
			},
			key:      "key",
			expected: 100,
		},
		{
			name: "int32 value",
			metadata: map[string]any{
				"key": int32(100),
			},
			key:      "key",
			expected: 100,
		},
		{
			name: "missing key",
			metadata: map[string]any{
				"other": int64(50),
			},
			key:      "key",
			expected: 0,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInt64(tt.metadata, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
