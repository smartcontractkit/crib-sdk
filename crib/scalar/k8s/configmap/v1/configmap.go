package configmapv1

import (
	"context"
	"errors"

	"github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2/k8s"

	"github.com/smartcontractkit/crib-sdk/crib"
	"github.com/smartcontractkit/crib-sdk/internal"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

type (
	Props struct {
		DataLoader  port.ValuesLoader
		Data        *map[string]*string
		Namespace   string `validate:"required"`
		AppName     string `validate:"required"`
		AppInstance string `validate:"required"`
		Name        string `validate:"required"`
	}
)

// Validate ensures that the Props struct satisfies the crib.Props interface.
func (p *Props) Validate(ctx context.Context) error {
	return internal.ValidatorFromContext(ctx).Struct(p)
}

func (p *Props) String() string {
	return "sdk.ConfigMapV1"
}

// Component returns a ComponentFunc for configmap.
func Component(props *Props) crib.ComponentFunc {
	props = dry.When(props != nil, props, &Props{})
	return func(ctx context.Context) (crib.Component, error) {
		if err := props.Validate(ctx); err != nil {
			return nil, err
		}

		if props.DataLoader != nil {
			if err := loadData(props); err != nil {
				return nil, err
			}
		}

		return New(ctx, props)
	}
}

// loadData invokes the DataLoader function and populates the values.
func loadData(props *Props) error {
	if props.Data != nil {
		return errors.New("cannot use both Data and DataLoader at the same time")
	}
	values, err := props.DataLoader.Values()
	if err != nil {
		return dry.Wrapf(err, "failed to load data values")
	}
	data := make(map[string]*string, len(values))
	for key, value := range values {
		data[key] = dry.ToPtr(dry.MustAs[string](value))
	}
	props.Data = dry.ToPtr(data)
	return nil
}

// New creates a new kubernetes configmap component. The resulting [crib.Component] represents a full intent to
// install a single configmap resource.
func New(ctx context.Context, props crib.Props) (crib.Component, error) {
	scalarProps := dry.MustAs[*Props](props)
	chart := crib.NewChart(ctx, scalarProps)

	configMapResourceID := crib.ResourceID(domain.CDK8sResource, props)
	resourceMetadataProps := &domain.DefaultResourceMetadataProps{
		Namespace:    scalarProps.Namespace,
		AppName:      scalarProps.AppName,
		AppInstance:  scalarProps.AppInstance,
		ResourceName: scalarProps.Name,
	}
	metadataFactory, err := domain.NewMetadataFactory(resourceMetadataProps)
	if err != nil {
		return nil, dry.Wrapf(err, "failed to create default metadata for configmap")
	}
	metadata := metadataFactory.K8sResourceMetadata()
	configMap := k8s.NewKubeConfigMap(chart, configMapResourceID, &k8s.KubeConfigMapProps{
		Data:     scalarProps.Data,
		Metadata: metadata,
	})

	return configMap, nil
}
