package configmapv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_mapComponents(t *testing.T) {
	t.Parallel()
	cm := new(ComponentMapper)

	tests := []struct {
		desc         string
		input        []*Component
		wantElements []*Component
		errAssert    assert.ErrorAssertionFunc
	}{
		{
			desc:      "empty input",
			errAssert: assert.NoError,
		},
		{
			desc: "single component",
			input: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "value1", "key2": "value2"},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			errAssert: assert.NoError,
		},
		{
			desc: "unique components",
			input: []*Component{
				{
					Name:      "test-cm-1",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "test-cm-2",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key3": "value3",
						"key4": "value4",
					},
					AppName:     "test-app",
					AppInstance: "test-app-456",
				},
			},
			wantElements: []*Component{
				{
					Name:        "test-cm-1",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "value1", "key2": "value2"},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:        "test-cm-2",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key3": "value3", "key4": "value4"},
					AppName:     "test-app",
					AppInstance: "test-app-456",
				},
			},
			errAssert: assert.NoError,
		},
		{
			desc: "duplicate components with different data",
			input: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key3": "value3",
						"key4": "value4",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			wantElements: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
						"key3": "value3",
						"key4": "value4",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			errAssert: assert.NoError,
		},
		{
			desc: "duplicate components with conflicting data",
			input: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key2": "value3",
						"key3": "value3",
						"key4": "value4",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			errAssert: assert.Error,
		},
		{
			desc: "test sorted capability",
			input: []*Component{
				{
					Name:      "b",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "a",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "b",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			wantElements: []*Component{
				{
					Name:      "b",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
				{
					Name:      "a",
					Namespace: "test-namespace",
					Data: map[string]string{
						"key2": "value2",
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
			errAssert: assert.NoError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got, err := cm.mapComponents(tc.input)
			if tc.errAssert(t, err) {
				assert.Equal(t, tc.wantElements, got, "expected and got should match")
			}
		})
	}
}

func Test_calculateDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc        string
		m1          map[string]string
		m2          map[string]string
		errContains string
	}{
		{
			desc: "no diff",
			m1:   map[string]string{"key1": "value1", "key2": "value2"},
			m2:   map[string]string{"key3": "value1", "key4": "value2"},
		},
		{
			desc:        "diff from m1",
			m1:          map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			m2:          map[string]string{"key1": "value1", "key2": "value2"},
			errContains: `["key1" "key2"]`,
		},
		{
			desc:        "diff from m2",
			m1:          map[string]string{"key1": "value1", "key2": "value2"},
			m2:          map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			errContains: `["key1" "key2"]`,
		},
		{
			desc:        "ordering does not matter", // Note: Map order is not guaranteed, but should give a good indication.
			m1:          map[string]string{"key1": "value1", "key2": "value2"},
			m2:          map[string]string{"key2": "value2", "key1": "value1"},
			errContains: `["key1" "key2"]`, // This is a bit of a hack, but it works for the test.
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			errCheck := assert.NoError
			if tc.errContains != "" {
				errCheck = assert.Error
			}
			err := calculateDiff(identifier{Name: "name", Namespace: "Namespace"}, tc.m1, tc.m2)
			if !errCheck(t, err) {
				assert.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}
