package networkinfov1

import (
	"testing"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal"

	blockchainnodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/node/v1"
)

func TestProps_Validate(t *testing.T) {
	v, err := internal.NewValidator()
	require.NoError(t, err)

	tests := []struct {
		name    string
		props   *Props
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid props with anvil",
			props: &Props{
				BlockchainIndex: 0,
				NodeIndex:       0,
				NodeProps: &blockchainnodev1.Props{
					Type:        blockchainnodev1.BlockchainTypeAnvil,
					ReleaseName: "test-anvil",
					Namespace:   "default",
					ChainID:     "1337",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid props with aptos",
			props: &Props{
				BlockchainIndex: 1,
				NodeIndex:       0,
				NodeProps: &blockchainnodev1.Props{
					Type:        blockchainnodev1.BlockchainTypeAptos,
					ReleaseName: "test-aptos",
					Namespace:   "default",
					ChainID:     "1",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "nil node props",
			props: &Props{
				BlockchainIndex: 0,
				NodeIndex:       0,
				NodeProps:       nil,
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := internal.ContextWithValidator(t.Context(), v)

			err = tt.props.Validate(ctx)
			tt.wantErr(t, err)
		})
	}
}

func TestComponent(t *testing.T) {
	validProps := &Props{
		BlockchainIndex: 0,
		NodeIndex:       0,
		NodeProps: &blockchainnodev1.Props{
			Type:        blockchainnodev1.BlockchainTypeAnvil,
			ReleaseName: "test-anvil",
			Namespace:   "default",
			ChainID:     "1337",
		},
	}

	tests := []struct {
		name        string
		props       *Props
		wantErr     assert.ErrorAssertionFunc
		errContains string
	}{
		{
			name:    "valid network info component",
			props:   validProps,
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a parent app and chart for testing
			app := cdk8s.NewApp(nil)
			chartName := "test"
			parent := cdk8s.NewChart(app, &chartName, nil)
			ctx := internal.ContextWithConstruct(t.Context(), parent)

			// Create the component
			component := Component(tt.props)
			chart, err := component(ctx)

			if tt.wantErr(t, err) {
				if err == nil {
					assert.NotNil(t, chart)
				}
				if err != nil {
					assert.ErrorContains(t, err, tt.errContains)
				}
			}
		})
	}
}
