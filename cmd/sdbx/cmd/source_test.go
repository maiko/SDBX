package cmd

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello world", 5, "he..."},
		{"", 5, ""},
		{"ab", 5, "ab"},
		{"exactly10!", 10, "exactly10!"},
		{"exactly11!!", 10, "exactly..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestGetSourceConfigPath(t *testing.T) {
	path := getSourceConfigPath()
	if path == "" {
		t.Error("getSourceConfigPath should return a non-empty path")
	}
}
