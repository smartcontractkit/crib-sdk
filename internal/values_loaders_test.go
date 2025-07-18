package internal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
)

func Test_TestYAMLLoader(t *testing.T) {
	t.Parallel()

	in := map[string]any{
		"key1": "value1",
	}
	loader := NewTestYAMLLoader(in)
	got, err := loader.Values()
	assert.NoError(t, err, "Values() should not return an error")
	assert.Equal(t, in, got)
}

func TestYAMLLoader(t *testing.T) {
	t.Parallel()

	testdata := `---
key1: value1
key2: true
key3:
  - item1
key4: 3
key5:
  nestedKey: nestedValue
`
	want := map[string]any{
		"key1": "value1",
		"key2": true,
		"key3": []any{"item1"},
		"key4": 3,
		"key5": map[string]any{
			"nestedKey": "nestedValue",
		},
	}

	buf := bytes.NewBufferString(testdata)

	loader := NewYAMLLoader()
	t.Run("Values should error on empty buffer", func(t *testing.T) {
		got, err := loader.Values()
		assert.Error(t, err, "Values() should return an error on empty buffer")
		assert.Len(t, got, 0, "Values() should return an empty map on error")
	})
	t.Run("Parse should error on invalid YAML", func(t *testing.T) {
		invalidBuf := bytes.NewBufferString("key1: value1\nkey2: true\nkey3: [item1\n")
		err := loader.Parse(invalidBuf)
		assert.Error(t, err, "Parse() should return an error on invalid YAML")
	})
	t.Run("Values should parse valid YAML", func(t *testing.T) {
		err := loader.Parse(buf)
		assert.NoError(t, err, "Parse() should not return an error")
		got, err := loader.Values()
		assert.NoError(t, err, "Values() should not return an error")
		assert.Equal(t, want, got, "Values() should return the expected map")
	})
	t.Run("Parse should error if called again", func(t *testing.T) {
		err := loader.Parse(buf)
		assert.Error(t, err, "Parse() should return an error if called again after successful parse")
	})
}

// TestHelmValuesValuesLoader exercises the YAMLLoader and the FileLoader.
func TestHelmValuesLoader(t *testing.T) {
	t.Parallel()

	want := map[string]any{
		"key1": "a",
		"key2": true,
		"key3": 3,
		"key4": []any{"a"},
		"key5": map[string]any{
			"nestedKey": "nestedValue",
		},
	}

	loader, err := NewHelmValuesLoader(t.Context(), "testdata")
	assert.NoError(t, err, "NewHelmValuesLoader should not return an error")
	got, err := loader.Values()
	assert.NoError(t, err, "Values() should not return an error")
	assert.Equal(t, want, got, "Values() should return the expected map")
}

func TestEnvLoader(t *testing.T) {
	prefix := gofakeit.BuzzWord()
	want := map[string]any{
		"DEBUG":      false,
		"PORT":       8080,
		"USER":       "Peter",
		"GAS":        1e6,
		"RATE":       0.5,
		"TIMEOUT":    "3m0s",
		"USERS":      []any{"rob", "ken", "robert"},
		"COLORCODES": map[string]any{"red": 1, "green": 2, "blue": 3},
	}
	vars := []string{
		"DEBUG=false",
		"PORT=8080",
		"USER=Peter",
		"GAS=1e6",
		"RATE=0.5",
		"TIMEOUT=3m",
		"USERS=rob,ken,robert",
		"COLORCODES=red:1,green:2,blue:3",
	}
	for _, v := range vars {
		key, value, _ := strings.Cut(v, "=")
		t.Setenv(prefix+"_"+key, value)
	}

	loader := NewEnvLoader(prefix)
	got, err := loader.Values()
	assert.NoError(t, err, "Values() should not return an error")
	assert.Equal(t, want, got, "Values() should return the expected map")
}
