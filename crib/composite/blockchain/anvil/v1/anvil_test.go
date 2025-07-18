package anvilv1

import (
	"fmt"
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
)

func TestAnvilComponent(t *testing.T) {
	tests := []struct {
		name        string
		props       *Props
		propOptions []PropOpt
	}{
		{
			name: "minimal options",
			props: &Props{
				Namespace: "foo-bar-baz",
				ChainID:   "1234",
			},
			propOptions: []PropOpt{},
		},
		{
			name: "with persistence and ingress config",
			props: &Props{
				Namespace: "foo-bar-baz",
				ChainID:   "1234",
			},
			propOptions: []PropOpt{UsePersistence, UseIngress},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			internal.JSIIKernelMutex.Lock()
			t.Cleanup(internal.JSIIKernelMutex.Unlock)

			is := assert.New(t)
			must := require.New(t)

			app := internal.NewTestApp(t)
			ctx := internal.ContextWithConstruct(t.Context(), app.Chart)

			componentFunc := Component(tt.props, tt.propOptions...)
			result, err := componentFunc(ctx)
			must.NoError(err)
			must.NotNil(result)

			// Validate metadata
			castResult := dry.MustAs[Result](result)
			is.Equal(fmt.Sprintf("anvil-%s", tt.props.ChainID), castResult.appInstanceName)

			chart := dry.MustAs[cdk8s.Chart](castResult.Component)
			is.NotNil(chart)

			internal.SynthAndSnapYamls(t, app)
		})
	}
}
