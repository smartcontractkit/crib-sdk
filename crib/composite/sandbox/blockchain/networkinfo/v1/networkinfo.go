package networkinfov1

import (
	"context"
	"fmt"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	blockchainnodev1 "github.com/smartcontractkit/crib-sdk/crib/composite/sandbox/blockchain/node/v1"
	configmapv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/configmap/v1"
)

// Props contains the configuration for a network info component.
type Props struct {
	NodeProps       *blockchainnodev1.Props
	BlockchainIndex int
	NodeIndex       int
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	if p.NodeProps == nil {
		return fmt.Errorf("node props cannot be nil")
	}
	if err := p.NodeProps.Validate(ctx); err != nil {
		return fmt.Errorf("invalid node props: %w", err)
	}
	return nil
}

// Component returns a new NetworkInfo composite component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return networkInfo(ctx, props)
	}
}

// networkInfo creates and returns a new network info composite component that generates
// a configmap with network information for a blockchain node.
func networkInfo(ctx context.Context, props *Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.NetworkInfo", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	// Create configmap data with network information
	configMapData := map[string]*string{
		fmt.Sprintf("01-network-%s.toml", props.NodeProps.ReleaseName): dry.ToPtr(dry.RemoveIndentation(fmt.Sprintf(`
			[[EVM]]
			ChainID = "%s"

			[[EVM.Nodes]]
			Name = '%s_primary_chainlink_local'
			HTTPURL = "http://%s.%s.svc.cluster.local:8545"
			WSURL = "ws://%s.%s.svc.cluster.local:8545"
		`, props.NodeProps.ChainID, props.NodeProps.ChainID, props.NodeProps.ReleaseName, props.NodeProps.Namespace, props.NodeProps.ReleaseName, props.NodeProps.Namespace))),
	}

	configMapName := fmt.Sprintf("network-anvil-%s", props.NodeProps.ChainID)
	component := configmapv1.Component(&configmapv1.Props{
		Name:        configMapName,
		AppName:     "blockchain-networks",
		AppInstance: "blockchain-networks-instance",
		Namespace:   props.NodeProps.Namespace,
		Data:        &configMapData,
	})

	configMap, err := component(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create configmap: %w", err)
	}

	return configMap, nil
}
