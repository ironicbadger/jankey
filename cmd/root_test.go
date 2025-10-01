package cmd

import (
	"testing"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single tag with prefix",
			input:    "tag:container",
			expected: []string{"tag:container"},
		},
		{
			name:     "single tag without prefix",
			input:    "container",
			expected: []string{"tag:container"},
		},
		{
			name:     "multiple tags with prefix",
			input:    "tag:docker,tag:ci",
			expected: []string{"tag:docker", "tag:ci"},
		},
		{
			name:     "multiple tags mixed",
			input:    "tag:docker,ci,tag:production",
			expected: []string{"tag:docker", "tag:ci", "tag:production"},
		},
		{
			name:     "tags with spaces",
			input:    "tag:docker, ci, tag:production",
			expected: []string{"tag:docker", "tag:ci", "tag:production"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseTags(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.input, i, tag, tt.expected[i])
				}
			}
		})
	}
}
