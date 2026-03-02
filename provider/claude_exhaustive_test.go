package provider

import (
	"testing"
)

func TestClaudeCodeProvider_ExhaustiveParsing(t *testing.T) {
	p, _ := NewClaudeCodeProvider(ProviderConfig{}, nil)

	// exhaustiveCases defines all known JSON variations observed in the wild
	exhaustiveCases := []struct {
		name     string
		json     string
		wantType []ProviderEventType
		check    func(*testing.T, []*ProviderEvent)
	}{
		{
			name:     "thinking_basic",
			json:     `{"type": "thinking", "content": [{"type": "text", "text": "I'll start by listing the files."}]}`,
			wantType: []ProviderEventType{EventTypeThinking},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Content != "I'll start by listing the files." {
					t.Errorf("Expected content match, got %q", evts[0].Content)
				}
			},
		},
		{
			name:     "thinking_with_subtype_plan",
			json:     `{"type": "thinking", "subtype": "plan_generation", "status": "Planning", "content": [{"type": "text", "text": "Step 1"}]}`,
			wantType: []ProviderEventType{EventTypePlanMode},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Content != "Step 1" {
					t.Errorf("Expected plan content, got %q", evts[0].Content)
				}
			},
		},
		{
			name:     "assistant_with_embedded_tool",
			json:     `{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Checking..."}, {"type": "tool_use", "id": "call_1", "name": "ls", "input": {"path": "."}}]}}`,
			wantType: []ProviderEventType{EventTypeAnswer, EventTypeToolUse},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if len(evts) != 2 {
					t.Fatalf("Expected 2 events, got %d", len(evts))
				}
				if evts[0].Type != EventTypeAnswer || evts[0].Content != "Checking..." {
					t.Errorf("Wrong first event: %+v", evts[0])
				}
				if evts[1].Type != EventTypeToolUse || evts[1].ToolName != "ls" {
					t.Errorf("Wrong second event: %+v", evts[1])
				}
			},
		},
		{
			name:     "tool_result_from_content_block",
			json:     `{"type": "tool_result", "content": [{"type": "tool_result", "tool_use_id": "call_1", "name": "ls", "content": "file.go", "is_error": false}]}`,
			wantType: []ProviderEventType{EventTypeToolResult},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].ToolName != "ls" || evts[0].Content != "file.go" {
					t.Errorf("Failed to extract from block: %+v", evts[0])
				}
				if evts[0].Status != "success" {
					t.Errorf("Expected success, got %s", evts[0].Status)
				}
			},
		},
		{
			name:     "result_with_durations",
			json:     `{"type": "result", "result": "Done", "duration": 1500, "usage": {"input_tokens": 10}}`,
			wantType: []ProviderEventType{EventTypeResult},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Metadata == nil || evts[0].Metadata.TotalDurationMs != 1500 {
					t.Errorf("Duration not captured correctly: %+v", evts[0].Metadata)
				}
			},
		},
		{
			name:     "result_alternate_duration",
			json:     `{"type": "result", "result": "Done", "duration_ms": 750, "usage": {"input_tokens": 5}}`,
			wantType: []ProviderEventType{EventTypeResult},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Metadata == nil || evts[0].Metadata.TotalDurationMs != 750 {
					t.Errorf("Duration_ms not captured correctly: %+v", evts[0].Metadata)
				}
			},
		},
		{
			name:     "result_with_full_usage",
			json:     `{"type": "result", "result": "Task completed", "duration_ms": 2000, "usage": {"input_tokens": 1200, "output_tokens": 350, "cache_creation_input_tokens": 100, "cache_read_input_tokens": 50}, "total_cost_usd": 0.05}`,
			wantType: []ProviderEventType{EventTypeResult},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Metadata == nil {
					t.Fatal("Expected non-nil metadata")
				}
				if evts[0].Metadata.InputTokens != 1200 {
					t.Errorf("Expected input_tokens=1200, got %d", evts[0].Metadata.InputTokens)
				}
				if evts[0].Metadata.OutputTokens != 350 {
					t.Errorf("Expected output_tokens=350, got %d", evts[0].Metadata.OutputTokens)
				}
				if evts[0].Metadata.CacheWriteTokens != 100 {
					t.Errorf("Expected cache_write_tokens=100, got %d", evts[0].Metadata.CacheWriteTokens)
				}
				if evts[0].Metadata.CacheReadTokens != 50 {
					t.Errorf("Expected cache_read_tokens=50, got %d", evts[0].Metadata.CacheReadTokens)
				}
				if evts[0].Metadata.TotalCostUSD != 0.05 {
					t.Errorf("Expected total_cost_usd=0.05, got %f", evts[0].Metadata.TotalCostUSD)
				}
			},
		},
		{
			name:     "result_without_usage",
			json:     `{"type": "result", "result": "Done", "duration_ms": 1000}`,
			wantType: []ProviderEventType{EventTypeResult},
			check: func(t *testing.T, evts []*ProviderEvent) {
				if evts[0].Metadata == nil {
					t.Fatal("Expected non-nil metadata even without usage")
				}
				if evts[0].Metadata.TotalDurationMs != 1000 {
					t.Errorf("Expected duration=1000, got %d", evts[0].Metadata.TotalDurationMs)
				}
				// Tokens should be 0 when usage is not provided
				if evts[0].Metadata.InputTokens != 0 {
					t.Errorf("Expected input_tokens=0, got %d", evts[0].Metadata.InputTokens)
				}
				if evts[0].Metadata.OutputTokens != 0 {
					t.Errorf("Expected output_tokens=0, got %d", evts[0].Metadata.OutputTokens)
				}
			},
		},
	}

	for _, tc := range exhaustiveCases {
		t.Run(tc.name, func(t *testing.T) {
			events, err := p.ParseEvent(tc.json)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			if len(events) != len(tc.wantType) {
				t.Fatalf("Expected %d events, got %d", len(tc.wantType), len(events))
			}

			for i, want := range tc.wantType {
				if events[i].Type != want {
					t.Errorf("Event[%d] type mismatch: want %s, got %s", i, want, events[i].Type)
				}
			}

			if tc.check != nil {
				tc.check(t, events)
			}
		})
	}
}

func TestClaudeCodeProvider_MalformedJSON(t *testing.T) {
	p, _ := NewClaudeCodeProvider(ProviderConfig{}, nil)
	line := "not json at all"
	events, err := p.ParseEvent(line)
	if err != nil {
		t.Fatal("Expected no error for raw fallback")
	}
	if len(events) != 1 || events[0].Type != EventTypeRaw {
		t.Errorf("Expected EventTypeRaw, got %v", events[0].Type)
	}
}
