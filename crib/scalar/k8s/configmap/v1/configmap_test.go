package configmapv1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestConfigMap(t *testing.T) {
	t.Parallel()
	internal.JSIIKernelMutex.Lock()
	t.Cleanup(internal.JSIIKernelMutex.Unlock)

	var (
		multilineString = `chainsCount: 200
bufferSize: 160MB
`
		testValues = map[string]any{
			"foo.conf": multilineString,
		}
	)

	tests := []struct {
		name        string
		setupProps  func() *Props
		expectErr   assert.ErrorAssertionFunc
		errContains string
	}{
		{
			name: "with static data",
			setupProps: func() *Props {
				return &Props{
					Name:        "test-cm",
					AppName:     "test-app",
					AppInstance: "test-app-123",
					Namespace:   "test-namespace",
					Data: &map[string]*string{
						"foo.conf": dry.ToPtr(multilineString),
					},
				}
			},
			expectErr: assert.NoError,
		},
		{
			name: "with data loader",
			setupProps: func() *Props {
				return &Props{
					Name:        "test-cm",
					AppName:     "test-app",
					AppInstance: "test-app-123",
					Namespace:   "test-namespace",
					DataLoader:  internal.NewTestYAMLLoader(testValues),
				}
			},
			expectErr: assert.NoError,
		},
		{
			name: "with both data and data loader",
			setupProps: func() *Props {
				return &Props{
					Name:        "test-cm",
					AppName:     "test-app",
					AppInstance: "test-app-123",
					Namespace:   "test-namespace",
					Data: &map[string]*string{
						"foo.conf": dry.ToPtr(multilineString),
					},
					DataLoader: internal.NewTestYAMLLoader(testValues),
				}
			},
			expectErr:   assert.Error,
			errContains: "cannot use both Data and DataLoader at the same time",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := assert.New(t)

			app := internal.NewTestApp(t)
			testProps := tc.setupProps()

			component, err := Component(testProps)(app.Context())
			tc.expectErr(t, err)
			_ = tc.errContains != "" && is.ErrorContains(err, tc.errContains)
			if err == nil {
				is.NotNil(component)
				app.SynthYaml()
			}
		})
	}
}
