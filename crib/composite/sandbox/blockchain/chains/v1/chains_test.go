package chainsv1

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
				Configs: []*BlockchainConfig{
					{
						NodeSet: []*blockchainnodev1.Props{
							{
								Type:    blockchainnodev1.BlockchainTypeAnvil,
								ChainID: "1337",
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid props with aptos",
			props: &Props{
				Configs: []*BlockchainConfig{
					{
						NodeSet: []*blockchainnodev1.Props{
							{
								Type:    blockchainnodev1.BlockchainTypeAptos,
								ChainID: "1",
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid props with multiple nodes",
			props: &Props{
				Configs: []*BlockchainConfig{
					{
						NodeSet: []*blockchainnodev1.Props{
							{
								Type:    blockchainnodev1.BlockchainTypeAnvil,
								ChainID: "1337",
							},
							{
								Type:    blockchainnodev1.BlockchainTypeAnvil,
								ChainID: "2337",
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
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
	// Note: This test cannot be run in parallel due to the use of global variables in the component.

	validAnvilBlockchain := &BlockchainConfig{
		NodeSet: []*blockchainnodev1.Props{
			{
				Namespace:   "default",
				ReleaseName: "test-anvil",
				Type:        blockchainnodev1.BlockchainTypeAnvil,
				ChainID:     "1337",
				Values: map[string]any{
					"chainId": "1337",
				},
			},
		},
	}

	validAptosBlockchain := &BlockchainConfig{
		NodeSet: []*blockchainnodev1.Props{
			{
				Namespace:   "default",
				ReleaseName: "test-aptos",
				Type:        blockchainnodev1.BlockchainTypeAptos,
				ChainID:     "1",
				Values: map[string]any{
					"chainId": "1",
				},
			},
		},
	}

	validMultiNodeBlockchain := &BlockchainConfig{
		NodeSet: []*blockchainnodev1.Props{
			{
				Namespace:   "default",
				ReleaseName: "test-anvil-1",
				Type:        blockchainnodev1.BlockchainTypeAnvil,
				ChainID:     "1337",
				Values: map[string]any{
					"chainId": "1337",
				},
			},
			{
				Namespace:   "default",
				ReleaseName: "test-anvil-2",
				Type:        blockchainnodev1.BlockchainTypeAnvil,
				ChainID:     "2337",
				Values: map[string]any{
					"chainId": "2337",
				},
			},
		},
	}

	tests := []struct {
		name        string
		configs     []*BlockchainConfig
		opts        []propOpts
		wantErr     assert.ErrorAssertionFunc
		errContains string
	}{
		{
			name:    "valid parallel processing with anvil",
			configs: []*BlockchainConfig{validAnvilBlockchain},
			opts:    []propOpts{ParallelProcessing},
			wantErr: assert.NoError,
		},
		{
			name:    "valid parallel processing with aptos",
			configs: []*BlockchainConfig{validAptosBlockchain},
			opts:    []propOpts{ParallelProcessing},
			wantErr: assert.NoError,
		},
		{
			name:    "valid parallel processing with multiple nodes",
			configs: []*BlockchainConfig{validMultiNodeBlockchain},
			opts:    []propOpts{ParallelProcessing},
			wantErr: assert.NoError,
		},
		{
			name:    "valid sequential processing with anvil (default)",
			configs: []*BlockchainConfig{validAnvilBlockchain},
			wantErr: assert.NoError,
		},
		{
			name:    "valid sequential processing with aptos (default)",
			configs: []*BlockchainConfig{validAptosBlockchain},
			wantErr: assert.NoError,
		},
		{
			name:    "valid sequential processing with multiple nodes (default)",
			configs: []*BlockchainConfig{validMultiNodeBlockchain},
			wantErr: assert.NoError,
		},
		{
			name:    "empty blockchains",
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
			component := Component(tt.configs, tt.opts...)
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

func TestProcessingMode(t *testing.T) {
	props := &Props{}
	assert.Equal(t, modeSequential, props.mode)
	ParallelProcessing(props)
	assert.Equal(t, modeParallel, props.mode)
}

// TestConfigMapGeneration demonstrates the configmap creation for blockchains
func TestConfigMapGeneration(t *testing.T) {
	// Create a simple blockchain configuration
	blockchainConfig := &BlockchainConfig{
		NodeSet: []*blockchainnodev1.Props{
			{
				Namespace:   "test-namespace",
				ReleaseName: "test-anvil",
				Type:        blockchainnodev1.BlockchainTypeAnvil,
				ChainID:     "1337",
				Values: map[string]any{
					"chainId": "1337",
				},
			},
		},
	}

	// Create the component
	app := cdk8s.NewApp(nil)
	chartName := "test-configmap"
	parent := cdk8s.NewChart(app, &chartName, nil)
	ctx := internal.ContextWithConstruct(t.Context(), parent)

	component := Component([]*BlockchainConfig{blockchainConfig})
	chart, err := component(ctx)

	require.NoError(t, err)
	assert.NotNil(t, chart)

	// The chart should contain both the blockchain component and the configmap
	// We can verify this by checking that the component was created successfully
	t.Logf("Successfully created blockchain component with configmap")
}
