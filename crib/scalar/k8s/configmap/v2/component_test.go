package configmapv2

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestScalar(t *testing.T) {
	t.Parallel()

	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	var (
		multilineString = dry.RemoveIndentation(`
			chainsCount: 200
			bufferSize: 160MB
		`)
		testValues = map[string]any{
			"foo.conf": multilineString,
		}
	)

	tests := []struct {
		desc        string
		opts        []ConfigMapOpt
		expectErr   assert.ErrorAssertionFunc
		errContains string
	}{
		{
			desc: "no values provided",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr: assert.NoError,
		},
		{
			desc: "static data",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithData(map[string]string{
					"foo.conf": multilineString,
				}),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr: assert.NoError,
		},
		{
			desc: "values loader",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr: assert.NoError,
		},
		{
			desc: "both static data and values loader",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithData(map[string]string{
					"bar.conf": multilineString,
				}),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr: assert.NoError,
		},
		{
			desc: "missing name",
			opts: []ConfigMapOpt{
				WithNamespace("test-namespace"),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr:   assert.Error,
			errContains: "Component.Name",
		},
		{
			desc: "missing namespace",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr:   assert.Error,
			errContains: "Component.Namespace",
		},
		{
			desc: "missing app name",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppInstance("test-app-123"),
			},
			expectErr:   assert.Error,
			errContains: "Component.AppName",
		},
		{
			desc: "missing app instance",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithValuesLoader(internal.NewTestYAMLLoader(testValues)),
				WithAppName("test-app"),
			},
			expectErr:   assert.Error,
			errContains: "Component.AppInstance",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			is := assert.New(t)
			app := internal.NewTestApp(t)

			component := Scalar("", tc.opts...)()
			_, err := component.Apply(app.Context())
			tc.expectErr(t, err)
			_ = tc.errContains != "" && is.ErrorContains(err, tc.errContains)
			if err == nil {
				is.NotNil(component)
				app.SynthYaml()
			}
		})
	}
}
