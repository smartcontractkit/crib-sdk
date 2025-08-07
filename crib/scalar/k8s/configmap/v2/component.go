package configmapv2

import (
	"context"
	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"
	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/service"
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
)

func newScalar(name string, opts ...ConfigMapOpt) *Component {
	c := &Component{
		Name: name,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			panic(err) // TODO(polds): Handle errors properly once the Composite API supports it.
		}
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

// String returns the name of the component.
func (c *Component) String() string {
	return "sdk.ConfigMapV2"
}

func (c *Component) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(c)
}

func (c *Component) Apply(ctx context.Context, cf service.ChartFactory) (k8s.KubeConfigMap, error) {
	if err := c.Validate(ctx); err != nil {
		return nil, err
	}
	chart := cf.CreateChart(c)

	configMapResourceID := crib.ResourceID(domain.CDK8sResource, c)
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

	return k8s.NewKubeConfigMap(chart, configMapResourceID, &k8s.KubeConfigMapProps{
		Data:     dry.PtrMapping(c.Data),
		Metadata: metadataFactory.K8sResourceMetadata(),
	}), nil
}
