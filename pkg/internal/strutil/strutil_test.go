package strutil

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello world", 5, "he..."},
		{"hello world", 11, "hello world"},
		{"hello world", 2, "he"},
		{"hello world", 0, ""},
		{"你好，世界", 4, "你..."}, // "你好，世界" is 5 runes. Truncate to 4-3=1 rune + "..."
	}

	for _, tt := range tests {
		result := Truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("Truncate(%q, %d) = %q; want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
