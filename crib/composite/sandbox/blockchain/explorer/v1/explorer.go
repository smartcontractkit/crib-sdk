package blockchainexplorerv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"

	otterscanv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/charts/otterscan/v1"
	clientsideapply "github.com/smartcontractkit/crib-sdk/crib/scalar/clientsideapply/v1"
	helmchartv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/helmchart/v1"
)

const (
	// ExplorerTypeOtterscan represents the Otterscan explorer.
	ExplorerTypeOtterscan ExplorerType = "otterscan"
	// ExplorerTypeBlockscout represents the Blockscout explorer (future).
	ExplorerTypeBlockscout ExplorerType = "blockscout"
)

// ExplorerType represents the type of blockchain explorer.
type ExplorerType string

// ExplorerImpl defines the interface that all blockchain explorer implementations must satisfy.
type ExplorerImpl interface {
	// CreateComponent creates a new blockchain explorer component with the given chart props
	CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error)
}

// otterscanImpl implements ExplorerImpl for Otterscan blockchain explorer.
type otterscanImpl struct{}

// blockscoutImpl implements ExplorerImpl for Blockscout blockchain explorer.
type blockscoutImpl struct{}

type Props struct {
	Namespace     string
	ReleaseName   string
	Type          ExplorerType `validate:"required,oneof=otterscan blockscout"`
	Values        map[string]any
	ValuesPatches [][]string
}

// CreateComponent creates a new Otterscan blockchain explorer component with the given chart props.
func (i *otterscanImpl) CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error) {
	return otterscanv1.Component(props)(ctx)
}

// CreateComponent creates a new Blockscout blockchain explorer component with the given chart props.
func (i *blockscoutImpl) CreateComponent(ctx context.Context, props *helmchartv1.ChartProps) (crib.Component, error) {
	return nil, fmt.Errorf("blockscout explorer type is not yet implemented")
}

// Validate validates the props.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

// Component returns a new component function that creates a new blockchain explorer component.
func Component(props *Props) crib.ComponentFunc {
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}
		return blockchainExplorer(ctx, props)
	}
}

// blockchainExplorer creates and returns a new blockchain explorer composite component.
func blockchainExplorer(ctx context.Context, props crib.Props) (crib.Component, error) {
	if err := props.Validate(ctx); err != nil {
		return nil, err
	}
	parent := internal.ConstructFromContext(ctx)
	chart := cdk8s.NewChart(parent, crib.ResourceID("sdk.BlockExplorer", props), nil)
	ctx = internal.ContextWithConstruct(ctx, chart)

	blockExplorerProps := dry.MustAs[*Props](props)

	var explorerImpl ExplorerImpl
	switch blockExplorerProps.Type {
	case ExplorerTypeOtterscan:
		explorerImpl = &otterscanImpl{}
	case ExplorerTypeBlockscout:
		explorerImpl = &blockscoutImpl{}
	default:
		return nil, fmt.Errorf("unsupported explorer type: %s", blockExplorerProps.Type)
	}

	chartProps := &helmchartv1.ChartProps{
		Namespace:     blockExplorerProps.Namespace,
		ReleaseName:   blockExplorerProps.ReleaseName,
		Values:        blockExplorerProps.Values,
		ValuesPatches: blockExplorerProps.ValuesPatches,
	}

	be, err := explorerImpl.CreateComponent(ctx, chartProps)
	if err != nil {
		return nil, err
	}

	labels := []string{
		fmt.Sprintf("app.kubernetes.io/component=%s", blockExplorerProps.ReleaseName),
		"app.kubernetes.io/name=devspace-app",
	}
	waitFor, err := clientsideapply.New(ctx, &clientsideapply.Props{
		Namespace: blockExplorerProps.Namespace,
		OnFailure: domain.FailureAbort,
		Action:    domain.ActionKubectl,
		Args: []string{
			"wait",
			"-n", blockExplorerProps.Namespace,
			"--for=condition=ready",
			"pod",
			fmt.Sprintf("-l=%s", strings.Join(labels, ",")),
			"--timeout=600s",
		},
	})
	if err != nil {
		return nil, err
	}

	waitFor.Node().AddDependency(be)

	return chart, nil
}
