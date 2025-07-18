package chainsv1

import (
	"context"
	"fmt"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"

	blockchainexplorerv1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/explorer/v1"
	networkinfov1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/networkinfo/v1"
	blockchainnodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/node/v1"
)

// Package chainsv1 provides a composite component for managing multiple blockchain instances.
//
// The Blockchains component creates and manages multiple blockchain nodes and their associated
// explorers. For each blockchain node, it also creates a ConfigMap containing network information
// that can be used by other components to discover and connect to the blockchain networks.
//
// Features:
// - Support for multiple blockchain types (Anvil, Aptos)
// - Parallel or sequential processing modes
// - Optional blockchain explorers
// - Automatic ConfigMap generation with network information for each blockchain
//
// Example usage:
//
//	configs := []*BlockchainConfig{
//		{
//			NodeSet: []*blockchainnodev1.Props{
//				{
//					Type:        blockchainnodev1.BlockchainTypeAnvil,
//					ReleaseName: "my-anvil",
//					Namespace:   "default",
//				},
//			},
//		},
//	}
//
//	component := chainsv1.Component(configs, chainsv1.ParallelProcessing)

const (
	modeSequential ProcessingMode = iota
	modeParallel
)

type ProcessingMode uint8

type BlockchainConfig struct {
	Explorer *blockchainexplorerv1.Props
	NodeSet  []*blockchainnodev1.Props `validate:"required,dive"`
}

type Props struct {
	Configs []*BlockchainConfig `validate:"dive"`
	mode    ProcessingMode
}

type propOpts func(*Props)

// ParallelProcessing sets the processing mode to parallel for the blockchains component.
// When enabled, blockchain components will be processed concurrently rather than sequentially.
func ParallelProcessing(p *Props) {
	p.mode = modeParallel
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	if err := internal.ValidatorFromContext(ctx).Struct(p); err != nil {
		return err
	}

	// Validate each blockchain config
	for i, config := range p.Configs {
		if len(config.NodeSet) == 0 {
			return fmt.Errorf("blockchain config %d: node set cannot be empty", i)
		}
		for j, node := range config.NodeSet {
			if node == nil {
				return fmt.Errorf("blockchain config %d, node %d: node props cannot be nil", i, j)
			}
			if err := node.Validate(ctx); err != nil {
				return fmt.Errorf("blockchain config %d, node %d: %w", i, j, err)
			}
		}
	}
	return nil
}

// Component returns a new Blockchains composite component.
func Component(configs []*BlockchainConfig, modeOpt ...propOpts) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		p := &Props{
			Configs: configs,
		}
		for _, opt := range modeOpt {
			opt(p)
		}
		if err := p.Validate(ctx); err != nil {
			return nil, err
		}
		return blockchains(ctx, p)
	}
}

// blockchains creates and returns a new blockchains composite component that manages multiple blockchain instances.
func blockchains(ctx context.Context, props *Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.Blockchains", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// Calculate total number of components to pre-allocate slice
	totalComponents := 0
	for _, config := range props.Configs {
		totalComponents += len(config.NodeSet) // blockchain nodes
		if config.Explorer != nil {
			totalComponents++ // explorer
		}
		totalComponents++ // network info
	}

	// Create all blockchain components and their associated explorers
	allComponents := make([]crib.Component, 0, totalComponents)

	for i, config := range props.Configs {
		// Create blockchain components for each node in the set
		for j, nodeProps := range config.NodeSet {
			// Create blockchain component
			b, err := blockchainnodev1.Component(nodeProps)(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create blockchain %d, node %d: %w", i, j, err)
			}
			allComponents = append(allComponents, b)

			// Create explorer if configured (only for the first node in the set)
			if config.Explorer != nil && j == 0 {
				be, err := blockchainexplorerv1.Component(config.Explorer)(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to create explorer for blockchain %d: %w", i, err)
				}
				// Make explorer depend on its blockchain
				be.Node().AddDependency(b)
				allComponents = append(allComponents, be)
			}
		}

		// Create network info configmap for the first node in the set
		networkInfo, err := networkinfov1.Component(&networkinfov1.Props{
			BlockchainIndex: i,
			NodeIndex:       0,
			NodeProps:       config.NodeSet[0],
		})(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create network info for blockchain %d, node %d: %w", i, 0, err)
		}
		allComponents = append(allComponents, networkInfo)
	}

	// Handle dependencies based on processing mode
	if props.mode == modeSequential && len(allComponents) > 1 {
		// Create sequential dependencies between all blockchain components
		for i := 1; i < len(allComponents); i++ {
			allComponents[i].Node().AddDependency(allComponents[i-1])
		}
	}

	// Return the chart as the component that represents all blockchains
	return chart, nil
}
