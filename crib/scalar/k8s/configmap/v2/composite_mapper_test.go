package configmapv2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type FakeProducer struct {
	Name      string
	Namespace string
}

type FakeProducerResult struct {
	Name      string
	Namespace string
}

func NewFakeProducer(name, namespace string) func() *FakeProducer {
	return func() *FakeProducer {
		return &FakeProducer{
			Name:      name,
			Namespace: namespace,
		}
	}
}

func (p *FakeProducer) Validate(context.Context) error {
	return nil
}

func (p *FakeProducer) Apply() *FakeProducerResult {
	// no-op, return a FakeProducerResult.
	return &FakeProducerResult{
		Name:      p.Name,
		Namespace: p.Namespace,
	}
}

func (p *FakeProducerResult) String() string {
	return "sdk.composite.FakeProducer"
}

// Implement the IConfigMap interface for the FakeProducer.
func (p *FakeProducerResult) ConfigMap() *Component {
	return &Component{
		Name:      p.Name,
		Namespace: p.Namespace,
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		AppName:     "fake-app",
		AppInstance: "fake-app-instance",
	}
}

func TestE2E(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	app := crib.NewTestApp(t)

	composite := crib.NewComposite(
		NewFakeProducer("cm-1", domain.DefaultNamespace),
		NewFakeProducer("cm-2", domain.DefaultNamespace),
		NewFakeProducer("cm-1", "test-namespace"),
		Components(
			WithAppName("test-app"),
			WithAppInstance("test-app-123"),
		),
	)

	res, err := composite(app.Context())
	hasErr := assert.NoError(t, err, "expected no error when creating composite")
	isNil := assert.NotNil(t, res, "expected composite to be created")
	if !hasErr || !isNil {
		return
	}
	app.SynthYaml()
}

type configMapImpl struct {
	component *Component
}

func (c *configMapImpl) ConfigMap() *Component {
	return c.component
}

type fakeValuesLoader struct {
	returnErr bool
}

func (f *fakeValuesLoader) Values() (map[string]any, error) {
	if f.returnErr {
		return nil, assert.AnError
	}
	return map[string]any{
		"key1": "value1",
		"key2": "value2",
	}, nil
}

func Test_createComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc         string
		input        []IConfigMap
		opts         []ConfigMapOpt
		wantElements []*Component
	}{
		{
			desc: "no satisfying components",
		},
		{
			desc: "satisfied component returning nil component",
			input: []IConfigMap{
				&configMapImpl{},
			},
		},
		{
			desc: "valid component",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
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
		},
		{
			desc: "multiple valid components",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm-1",
						Namespace: "test-namespace",
						Data: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
				&configMapImpl{
					component: &Component{
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
			},
			wantElements: []*Component{
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
		},
		{
			desc: "opt: WithName",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
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
			},
			opts: []ConfigMapOpt{
				WithName("custom-cm-name"),
			},
			wantElements: []*Component{
				{
					Name:        "custom-cm-name",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "value1", "key2": "value2"},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: WithNamespace",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
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
			},
			opts: []ConfigMapOpt{
				WithNamespace("custom-namespace"),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "custom-namespace",
					Data:        map[string]string{"key1": "value1", "key2": "value2"},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: WithData - No initial values",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:        "test-cm",
						Namespace:   "test-namespace",
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithData(map[string]string{
					"key1": "value1",
					"key2": "value2",
				}),
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
		},
		{
			desc: "opt: WithData - Merged and Overwritten Values",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"key1": "value1",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithData(map[string]string{
					"key1": "new-value1", // This should overwrite the existing value
					"key2": "value2",     // This should be added
				}),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "new-value1", "key2": "value2"},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: WithAppName",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"key1": "value1",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithAppName("custom-app-name"),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "value1"},
					AppName:     "custom-app-name",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: WithAppInstance",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"key1": "value1",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithAppInstance("custom-app-instance"),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					Data:        map[string]string{"key1": "value1"},
					AppName:     "test-app",
					AppInstance: "custom-app-instance",
				},
			},
		},
		{
			desc: "opt: WithValuesLoader - No Error",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:        "test-cm",
						Namespace:   "test-namespace",
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithValuesLoader(&fakeValuesLoader{}),
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
		},
		{
			desc: "opt: WithValuesLoader - Has Error",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:        "test-cm",
						Namespace:   "test-namespace",
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithValuesLoader(&fakeValuesLoader{
					returnErr: true,
				}),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: WithValuesLoader - Nil ValuesLoader",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:        "test-cm",
						Namespace:   "test-namespace",
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithValuesLoader(nil),
			},
			wantElements: []*Component{
				{
					Name:        "test-cm",
					Namespace:   "test-namespace",
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: Values - Base + WithData + WithValuesLoader",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"baseKey":  "baseValue",
							"baseKey2": "baseValue2",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithData(map[string]string{
					"baseKey": "overriddenValue", // This should overwrite the existing value
					"newKey":  "newValue",        // This should be added
				}),
				WithValuesLoader(&fakeValuesLoader{}), // This should add more data
			},
			wantElements: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"baseKey":  "overriddenValue", // Overridden by WithData
						"baseKey2": "baseValue2",      // Provided from Component and not overridden
						"newKey":   "newValue",        // Added by WithData
						"key1":     "value1",          // Added by WithValuesLoader
						"key2":     "value2",          // Provided by WithValuesLoader and not overridden
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: Values - Base + WithValuesLoader + WithData",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"baseKey":  "baseValue",
							"baseKey2": "baseValue2",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithValuesLoader(&fakeValuesLoader{}), // This should add more data
				WithData(map[string]string{
					"baseKey": "overriddenValue", // This should overwrite the existing value
					"newKey":  "newValue",        // This should be added
				}),
			},
			wantElements: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"baseKey":  "overriddenValue", // Overridden by WithData
						"baseKey2": "baseValue2",      // Provided from Component and not overridden
						"newKey":   "newValue",        // Added by WithData
						"key1":     "value1",          // Added by WithValuesLoader
						"key2":     "value2",          // Provided by WithValuesLoader and not overridden
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
		{
			desc: "opt: Values - Base + nil WithData + nil WithValuesLoader",
			input: []IConfigMap{
				&configMapImpl{
					component: &Component{
						Name:      "test-cm",
						Namespace: "test-namespace",
						Data: map[string]string{
							"baseKey":  "baseValue",
							"baseKey2": "baseValue2",
						},
						AppName:     "test-app",
						AppInstance: "test-app-123",
					},
				},
			},
			opts: []ConfigMapOpt{
				WithData(map[string]string{}),
				WithValuesLoader(nil),
			},
			wantElements: []*Component{
				{
					Name:      "test-cm",
					Namespace: "test-namespace",
					Data: map[string]string{
						"baseKey":  "baseValue",  // Provided from Component and not overridden
						"baseKey2": "baseValue2", // Provided from Component and not overridden
					},
					AppName:     "test-app",
					AppInstance: "test-app-123",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			cm := &ComponentMapper{
				opts: tc.opts,
			}
			got := cm.createComponents(tc.input)
			assert.ElementsMatch(t, tc.wantElements, got, "expected and got should match")
		})
	}
}

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
