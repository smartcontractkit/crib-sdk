package configmapv2

import (
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScalar(t *testing.T) {
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
	_ = testValues // to avoid unused variable error

	tests := []struct {
		desc        string
		opts        []ConfigMapOpt
		expectErr   assert.ErrorAssertionFunc
		errContains string
	}{
		{
			desc: "with static data",
			opts: []ConfigMapOpt{
				WithName("test-cm"),
				WithNamespace("test-namespace"),
				WithData(map[string]string{
					"foo.conf": dry.RemoveIndentation(multilineString),
				}),
				WithAppName("test-app"),
				WithAppInstance("test-app-123"),
			},
			expectErr: assert.NoError,
		},
		//{
		//	name: "with data loader",
		//	setupProps: func() *Props {
		//		return &Props{
		//			Name:        "test-cm",
		//			AppName:     "test-app",
		//			AppInstance: "test-app-123",
		//			Namespace:   "test-namespace",
		//			DataLoader:  internal.NewTestYAMLLoader(testValues),
		//		}
		//	},
		//	expectErr: assert.NoError,
		//},
		//{
		//	name: "with both data and data loader",
		//	setupProps: func() *Props {
		//		return &Props{
		//			Name:        "test-cm",
		//			AppName:     "test-app",
		//			AppInstance: "test-app-123",
		//			Namespace:   "test-namespace",
		//			Data: &map[string]*string{
		//				"foo.conf": dry.ToPtr(multilineString),
		//			},
		//			DataLoader: internal.NewTestYAMLLoader(testValues),
		//		}
		//	},
		//	expectErr:   assert.Error,
		//	errContains: "cannot use both Data and DataLoader at the same time",
		//},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			is := assert.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			component := Scalar("", tc.opts...)()
			_, err := component.Apply(ctx, service.NewChartFactory())
			tc.expectErr(t, err)
			_ = tc.errContains != "" && is.ErrorContains(err, tc.errContains)
			if err == nil {
				is.NotNil(component)
				internal.SynthAndSnapYamls(t, app)
			}
		})
	}
}
