package configmapv2

import (
	"context"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

type (
	// Component represents a single Kubernetes ConfigMap component.
	Component struct {
		Name      string `validate:"required"`
		Namespace string `validate:"required"`

		Data        map[string]string
		AppName     string `validate:"required"`
		AppInstance string `validate:"required"`
	}

	// Result holds the result of applying the ConfigMap component.
	Result struct {
		ConfigMap k8s.KubeConfigMap
	}
)

func newScalar(name string, opts ...ConfigMapOpt) *Component {
	c := &Component{
		Name: name,
	}
	for _, opt := range opts {
		_ = opt(c) // TODO(polds): Handle errors properly once the Composite API supports it.
	}
	return c
}

// Scalar initializes a new ConfigMap component with the provided name and options.
// TODO(polds): Return the error returned by the ConfigMapOpt functions. The Composite API doesn't support this yet.
func Scalar(name string, opts ...ConfigMapOpt) func() *Component {
	return func() *Component {
		return newScalar(name, opts...)
	}
}

func (c *Component) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(c)
}

func (c *Component) Apply(ctx context.Context) (*Result, error) {
	if err := c.Validate(ctx); err != nil {
		return nil, err
	}
	chart := crib.NewChart(ctx, c)

	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    c.Namespace,
		AppName:      c.AppName,
		AppInstance:  c.AppInstance,
		ResourceName: c.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, err
	}

	configMapResourceID := crib.ResourceID(domain.CDK8sResource, c)
	return &Result{
		ConfigMap: k8s.NewKubeConfigMap(chart, configMapResourceID, &k8s.KubeConfigMapProps{
			Data:     dry.PtrMapping(c.Data),
			Metadata: metadataFactory.K8sResourceMetadata(),
		}),
	}, nil
}

// String returns the name of the component.
func (c *Component) String() string {
	return "sdk.ConfigMapV2"
}
