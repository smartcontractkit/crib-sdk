package blockchainnodev1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"

	anvilv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/anvil/v1"
	aptosv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/aptos/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
)

const (
	BlockchainTypeAnvil BlockchainType = "anvil"
	BlockchainTypeAptos BlockchainType = "aptos"
)

type BlockchainType string

// BlockchainImpl defines the interface that all blockchain node implementations must satisfy.
type BlockchainImpl interface {
	// CreateComponent creates a new blockchain node component with the given chart props
	CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error)
}

// anvilImpl implements BlockchainImpl for Anvil blockchain node.
type anvilImpl struct{}

// aptosImpl implements BlockchainImpl for Aptos blockchain node.
type aptosImpl struct{}

// Props contains the configuration for a blockchain node component.
type Props struct {
	Values        map[string]any
	Namespace     string
	ReleaseName   string
	Type          BlockchainType
	ChainID       string
	ValuesPatches [][]string
}

// CreateComponent creates a new Anvil blockchain node component with the given chart props.
func (i *anvilImpl) CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error) {
	return anvilv1.Component(props)(ctx)
}

// CreateComponent creates a new Aptos blockchain node component with the given chart props.
func (i *aptosImpl) CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error) {
	return aptosv1.Component(props)(ctx)
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	if p.Type == "" {
		return fmt.Errorf("blockchain type must be specified")
	}
	if p.Type != BlockchainTypeAnvil && p.Type != BlockchainTypeAptos {
		return fmt.Errorf("invalid blockchain type: %s, must be one of: %s, %s", p.Type, BlockchainTypeAnvil, BlockchainTypeAptos)
	}
	if p.ChainID == "" {
		return fmt.Errorf("chain ID must be specified")
	}
	return nil
}

// Component returns a new component function that creates a new blockchain node component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return blockchainNode(ctx, props)
	}
}

// blockchainNode creates and returns a new blockchain node composite component.
func blockchainNode(ctx context.Context, props crib.Props) (crib.Component, error) {
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.Blockchain", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	blockchainProps := dry.MustAs[*Props](props)

	var blockchainImpl BlockchainImpl
	switch blockchainProps.Type {
	case BlockchainTypeAnvil:
		blockchainImpl = &anvilImpl{}
	case BlockchainTypeAptos:
		blockchainImpl = &aptosImpl{}
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainProps.Type)
	}

	chartProps := &helmchartv1.ChartProps{
		Namespace:     blockchainProps.Namespace,
		ReleaseName:   blockchainProps.ReleaseName,
		Values:        blockchainProps.Values,
		ValuesPatches: blockchainProps.ValuesPatches,
	}

	b, err := blockchainImpl.CreateComponent(ctx, chartProps)
	if err != nil {
		return nil, err
	}

	labels := []string{
		fmt.Sprintf("app.kubernetes.io/component=%s", blockchainProps.ReleaseName),
		"app.kubernetes.io/name=devspace-app",
	}
	waitFor, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: blockchainProps.Namespace,
		OnFailure: "abort",
		Action:    "kubectl",
		Args: []string{
			"wait",
			"-n", blockchainProps.Namespace,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, err
	}

	waitFor.Node().AddDependency(b)
	return b, nil
}
