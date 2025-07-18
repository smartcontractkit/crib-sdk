package filehandler

import (
	"testing"
	"testing/fstest"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
)

func TestBufferedReader(t *testing.T) {
	t.Parallel()

	data := gofakeit.HipsterParagraph(3, 5, 5, "\n")
	fs := fstest.MapFS{
		"test.txt": &fstest.MapFile{
			Data: []byte(data),
			Mode: 0o644,
		},
	}

	tests := []struct {
		name     string
		reader   Reader
		expected string
	}{
		{
			name:     "DefaultReader",
			reader:   DefaultReader,
			expected: data,
		},
		{
			name:     "SmallLimitReader",
			reader:   LimitReader(10),
			expected: data[:10],
		},
		{
			name:     "LargeLimitReader",
			reader:   LimitReader(4096),
			expected: data,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewBufferedReader(fs, tc.reader)
			buf, err := r.ReadFile("test.txt")

			assert.NoError(t, err)
			assert.Equal(t, string(buf), tc.expected)
		})
	}
}
