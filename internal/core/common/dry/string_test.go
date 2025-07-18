package dry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single line",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name: "simple indentation",
			input: `
				line 1
				line 2
				line 3
			`,
			expected: "line 1\nline 2\nline 3",
		},
		{
			name: "mixed indentation",
			input: `
				line 1
				  line 2
				line 3
			`,
			expected: "line 1\n  line 2\nline 3",
		},
		{
			name: "no indentation",
			input: `
line 1
line 2
line 3
			`,
			expected: "line 1\nline 2\nline 3",
		},
		{
			name: "empty lines preserved",
			input: `
				line 1

				line 2
			`,
			expected: "line 1\n\nline 2",
		},
		{
			name: "only empty lines",
			input: `


			`,
			expected: "",
		},
		{
			name: "single empty line",
			input: `
			`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveIndentation(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
